package cli

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/canpok1/vox-radio/internal/assemble"
	"github.com/canpok1/vox-radio/internal/cache"
	"github.com/canpok1/vox-radio/internal/collect"
	"github.com/canpok1/vox-radio/internal/config"
	"github.com/canpok1/vox-radio/internal/fileio"
	"github.com/canpok1/vox-radio/internal/mediainfo"
	"github.com/canpok1/vox-radio/internal/model"
	"github.com/canpok1/vox-radio/internal/pipeline"
	"github.com/canpok1/vox-radio/internal/rundown"
	"github.com/canpok1/vox-radio/internal/rundown/flow"
	sel "github.com/canpok1/vox-radio/internal/rundown/select"
	"github.com/canpok1/vox-radio/internal/script"
	"github.com/canpok1/vox-radio/internal/script/direct"
	"github.com/canpok1/vox-radio/internal/script/summarize"
	programsummary "github.com/canpok1/vox-radio/internal/script/summary"
	"github.com/canpok1/vox-radio/internal/script/write"
	"github.com/canpok1/vox-radio/internal/synth"
	"github.com/spf13/cobra"
)

// assemblerAdapter wraps *assemble.Assembler to satisfy pipeline.Assembler.
type assemblerAdapter struct {
	inner *assemble.Assembler
}

func (a *assemblerAdapter) Run(ctx context.Context, scr model.Script, clips model.ClipsMeta, clipsDir, outPath string, meta model.EpisodeMeta) (map[string]float64, error) {
	result, err := a.inner.Run(ctx, scr, clips, clipsDir, outPath, meta)
	if err != nil {
		return nil, err
	}
	return result.CornerDurations, nil
}

func newEpisodegenCmd() *cobra.Command {
	var outDir string
	var specPath string
	var force bool

	cmd := &cobra.Command{
		Use:   "episodegen",
		Short: "ポッドキャスト制作パイプラインをすべて実行する",
		Args:  cobra.NoArgs,
		Long: `collect → rundown → script → synth → assemble → manifest を一括実行します。

実行には ffmpeg および ffprobe が必要です。インストール手順は vox-radio の README を参照してください:
https://github.com/canpok1/vox-radio#readme

最終的な {program.id}_ep{NNN}.mp3 とマニフェスト {program.id}_ep{NNN}_manifest.json は
<out-dir>/ 直下に、中間ファイルは <out-dir>/intermediate/{program.id}_ep{NNN}/ に配置されます。
回ごとに別名・別ディレクトリになるため、過去回の成果物は上書きされません。

mp3・マニフェスト・中間ディレクトリのいずれかが既に存在する場合はエラーで終了します。
上書きするには --force を指定してください（--force 指定時は中間ディレクトリを削除して作り直します）。

共通設定ファイルのパスは --config フラグで指定します（省略時は vox-radio.yaml）。
環境変数 VOX_RADIO_VOICEVOX_URL を設定すると、設定ファイルの voicevox.url より優先して VOICEVOX エンジンの URL を上書きできます。

例:
  vox-radio episodegen
  vox-radio episodegen --out-dir output --spec episode-spec.yaml
  vox-radio episodegen --force --spec episode-spec.yaml
  vox-radio --config /path/to/vox-radio.yaml episodegen --spec episode-spec.yaml`,
		RunE: func(cmd *cobra.Command, args []string) error {
			logger, logFile, err := setupLogger("episodegen", logDirFlag(cmd))
			if err != nil {
				return fmt.Errorf("setup logger: %w", err)
			}
			defer func() { _ = logFile.Close() }()

			if err := requireMediaTools(); err != nil {
				return err
			}

			cfg, p, err := loadConfigAndSpec(configPath(cmd), specPath)
			if err != nil {
				return err
			}

			// program.id is required (validated in loadConfigAndSpec), so the cache is always used.
			entries, episodeNumber, err := loadCacheEntries(p.Program.ID)
			if err != nil {
				return err
			}

			layout := fileio.EpisodeLayout{
				OutDir:        outDir,
				ProgramID:     p.Program.ID,
				EpisodeNumber: episodeNumber,
			}

			if force {
				// mp3 and manifest are single files overwritten in place by the
				// pipeline, but the intermediate dir can accumulate stale files
				// across runs (e.g. fewer corners), so remove it up front.
				// RemoveAll is a no-op if it is absent.
				if err := layout.RemoveIntermediateDir(); err != nil {
					return fmt.Errorf("remove intermediate dir: %w", err)
				}
			} else {
				for _, path := range []string{layout.Episode(), layout.Manifest(), layout.IntermediateDir()} {
					if _, err := os.Stat(path); err == nil {
						return fmt.Errorf("%s は既に存在します。上書きするには --force を指定してください", path)
					}
				}
			}

			llmClient := newLLMClient(cfg)

			prompts, err := loadPrompts()
			if err != nil {
				return fmt.Errorf("load prompts: %w", err)
			}

			assetCatalog := buildAssetCatalog(p.Assets)

			selector := sel.NewLLMSelector(llmClient, prompts["select"], stepTemp(cfg.LLM, "select"))
			flowDesigner := flow.NewLLMDesigner(llmClient, prompts["flow"], stepTemp(cfg.LLM, "flow"))
			loc := resolveLocation(p.Program, logger)
			writer := write.NewLLMWriter(llmClient, prompts["write"], stepTemp(cfg.LLM, "write"), cfg)
			writer.SetRecordedAt(time.Now(), loc)

			cacheMgr := cache.New(programCachePath(p.Program.ID))
			recent := cache.Recent(entries, cfg.Cache.EffectiveLLMContextEntries())
			excludedDedupKeys := cache.PastDedupKeys(entries)
			castAppearances := cache.CastAppearances(entries)
			cornerAppearances := cache.CornerAppearances(entries)
			writer.SetPastEpisodes(recent)
			writer.SetEpisodeNumber(episodeNumber)

			selectedCasts := selectCasts(p.Casts, episodeNumber, castAppearances)
			writer.SetCasts(selectedCasts)
			selector.SetCasts(selectedCasts)

			p.Corners = resolveCorners(p.Corners, episodeNumber)

			collector := collect.New(nil, collect.WithLogger(logger), collect.WithLocation(loc), collect.WithSanitizePolicy(cfg.Security.PromptInjection))
			summarizer := summarize.NewLLMSummarizer(llmClient, prompts["summarize"], stepTemp(cfg.LLM, "summarize"))
			rundowner := rundown.NewLLMRundowner(selector, summarizer, flowDesigner, excludedDedupKeys, rundown.WithLogger(logger))
			rundowner.SetCornerAppearances(cornerAppearances)

			scripter := script.NewLLMScriptGenerator(
				writer,
				direct.NewLLMDirector(llmClient, prompts["direct"], stepTemp(cfg.LLM, "direct"),
					direct.WithProofread(prompts["proofread"], stepTemp(cfg.LLM, "proofread")),
				),
				assetCatalog,
				script.WithLogger(logger),
			)

			engineURL := cfg.Voicevox.EffectiveURL()

			runner := &pipeline.Runner{
				Spec:              p,
				Config:            cfg,
				Collector:         collector,
				ExcludedDedupKeys: excludedDedupKeys,
				Rundowner:         rundowner,
				Scripter:          scripter,
				Synther:           synth.New(engineURL, cfg, synth.WithLogger(logger)),
				Assembler:         &assemblerAdapter{inner: assemble.New(p.Assets, p.Program, assemble.WithLogger(logger), assemble.WithFFmpegWriter(logFile))},
				ProgramSummarizer: programsummary.NewLLMProgramSummarizer(llmClient, prompts["summary"], stepTemp(cfg.LLM, "summary"), p.Program.EffectiveSummaryLength(), programsummary.WithLogger(logger)),
				CornerSummarizer:  programsummary.NewLLMCornerSummarizer(llmClient, prompts["corner_summary"], stepTemp(cfg.LLM, "corner_summary"), programsummary.WithLogger(logger)),
			}

			if err := runner.Run(context.Background(), pipeline.Options{
				OutDir:        outDir,
				EpisodeNumber: episodeNumber,
				Casts:         selectedCasts,
			}); err != nil {
				return err
			}

			if err := appendToCache(cacheMgr, layout, cfg.Cache, logger); err != nil {
				logger.Warn("cache append failed (non-fatal)", "err", err)
			}

			fmt.Printf("pipeline complete: episode at %s\n", layout.Episode())
			return nil
		},
	}

	cmd.Flags().StringVar(&outDir, "out-dir", "output", "出力ディレクトリ（{program.id}_ep{NNN}.mp3 と {program.id}_ep{NNN}_manifest.json をここに配置し、中間ファイルは <out-dir>/intermediate/{program.id}_ep{NNN}/ に配置）")
	cmd.Flags().BoolVar(&force, "force", false, "既存の出力（mp3・マニフェスト・中間ディレクトリ）を上書きする")
	registerSpecFlag(cmd, &specPath)

	cmd.AddCommand(
		newCollectCmd(),
		newRundownCmd(),
		newScriptCmd(),
		newSynthCmd(),
		newAssembleCmd(),
		newManifestCmd(),
		newEpisodegenCheckCmd(),
	)

	return cmd
}

func appendToCache(mgr *cache.Manager, layout fileio.EpisodeLayout, cacheCfg config.CacheConfig, logger *slog.Logger) error {
	var m model.Manifest
	if err := fileio.ReadJSON(layout.Manifest(), &m); err != nil {
		return fmt.Errorf("read manifest: %w", err)
	}
	var rd model.Rundown
	if err := fileio.ReadJSON(layout.Rundown(), &rd); err != nil {
		return fmt.Errorf("read rundown: %w", err)
	}

	episodePath := layout.Episode()
	var bytes int64
	var durationSec int
	if b, err := mediainfo.FileSize(episodePath); err != nil {
		logger.Warn("mediainfo.FileSize failed (non-fatal)", "err", err)
	} else {
		bytes = b
	}
	if d, err := mediainfo.Duration(episodePath); err != nil {
		logger.Warn("mediainfo.Duration failed (non-fatal)", "err", err)
	} else {
		durationSec = int(d)
	}

	entry := cache.BuildEntryFromManifest(layout.ProgramID, m, rd, bytes, durationSec)
	if err := mgr.Append(entry, cacheCfg.EffectiveMaxEntries(), cacheCfg.EffectiveRetentionDays()); err != nil {
		return err
	}
	logger.Info("cache entry appended", "program_id", layout.ProgramID)
	return nil
}
