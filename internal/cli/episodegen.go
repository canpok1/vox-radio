package cli

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/canpok1/vox-radio/internal/cache"
	"github.com/canpok1/vox-radio/internal/config"
	"github.com/canpok1/vox-radio/internal/fileio"
	"github.com/canpok1/vox-radio/internal/gather"
	"github.com/canpok1/vox-radio/internal/mediainfo"
	"github.com/canpok1/vox-radio/internal/mix"
	"github.com/canpok1/vox-radio/internal/model"
	"github.com/canpok1/vox-radio/internal/pipeline"
	"github.com/canpok1/vox-radio/internal/retro"
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

// mixerAdapter wraps *mix.Mixer to satisfy pipeline.Mixer.
type mixerAdapter struct {
	inner *mix.Mixer
}

func (a *mixerAdapter) Run(ctx context.Context, scr model.Script, clips model.ClipsMeta, clipsDir, outPath string, meta model.EpisodeMeta) (map[string]model.CornerTiming, error) {
	result, err := a.inner.Run(ctx, scr, clips, clipsDir, outPath, meta)
	if err != nil {
		return nil, err
	}
	timings := make(map[string]model.CornerTiming, len(result.CornerDurations))
	for id, total := range result.CornerDurations {
		timings[id] = model.CornerTiming{
			ID:           id,
			DurationSec:  total,
			SpeechSec:    result.SpeechDurations[id],
			NonSpeechSec: result.NonSpeechDurations[id],
		}
	}
	return timings, nil
}

func newEpisodegenCmd() *cobra.Command {
	var outDir string
	var specPath string
	var force bool

	refURL := referenceURL("README.md")
	cmd := &cobra.Command{
		Use:   "episodegen",
		Short: "ポッドキャスト制作パイプラインをすべて実行する",
		Args:  cobra.NoArgs,
		Long: fmt.Sprintf(`gather → rundown → script → synth → mix → manifest を一括実行します。

実行には ffmpeg および ffprobe が必要です。インストール手順は vox-radio の README を参照してください:
%s

最終的な {program.id}_ep{NNN}.mp3 とマニフェスト {program.id}_ep{NNN}_manifest.json は
<out-dir>/ 直下に、中間ファイルは <out-dir>/intermediate/{program.id}_ep{NNN}/ に配置されます。
回ごとに別名・別ディレクトリになるため、過去回の成果物は上書きされません。

program.single_shot を true にすると単発番組モードになり、回番号（第N回）を採番・露出せず、
出力名はサフィックス無し（{program.id}.mp3）になります。キャッシュにも保存しないため、
同名で上書き運用となります（再生成は --force）。

mp3・マニフェスト・中間ディレクトリのいずれかが既に存在する場合はエラーで終了します。
上書きするには --force を指定してください（--force 指定時は中間ディレクトリを削除して作り直します）。

共通設定ファイルのパスは --config フラグで指定します（省略時は vox-radio.yaml）。
環境変数 VOX_RADIO_VOICEVOX_URL を設定すると、設定ファイルの voicevox.url より優先して VOICEVOX エンジンの URL を上書きできます。
voicevox.engines で名前付きの複数 VOICEVOX 互換サーバー（例: VOICEVOX NEMO）を定義し、
characters.<id>.engine でキャラクターごとに使用サーバーを指定できます（省略時は default）。
サーバーごとの URL は環境変数 VOX_RADIO_VOICEVOX_URL_<サーバー名（大文字）> でも上書きできます。

例:
  vox-radio episodegen
  vox-radio episodegen --out-dir output --spec episode-spec.yaml
  vox-radio episodegen --force --spec episode-spec.yaml
  vox-radio --config /path/to/vox-radio.yaml episodegen --spec episode-spec.yaml`, refURL),
		RunE: func(cmd *cobra.Command, args []string) error {
			logger, logFile, err := setupLogger("episodegen", logDirFlag(cmd))
			if err != nil {
				return fmt.Errorf("setup logger: %w", err)
			}
			defer func() { _ = logFile.Close() }()

			cfg, p, err := loadConfigAndSpec(configPath(cmd), specPath, logger)
			if err != nil {
				return err
			}

			// single_shot（単発番組モード）では採番せず episodeNumber=0 を供給し、
			// キャッシュも参照しない（past_episodes なし・出演回数は初回相当）。
			// 通常モードは program.id をキーにキャッシュから次回番号を採番する。
			var entries []cache.Entry
			var episodeNumber int
			if !p.Program.SingleShot {
				entries, episodeNumber, err = loadCacheEntries(p.Program.ID)
				if err != nil {
					return err
				}
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

			// 必要な外部リソースをまとめて検証し、LLM でコストを消費する前に早期失敗させる
			// （VOICEVOX 未到達を synth 段まで見逃さない）。安価な既存出力ガードの後段に置き、
			// 出力が既存で即失敗するケースで VOICEVOX 待機を無駄に発生させない。
			ctx := context.Background()
			engineURLs := cfg.Voicevox.EffectiveURLs()
			if err := checkResources(
				requireMediaTools,
				func() error { return synth.CheckReadiness(ctx, engineURLs, cfg) },
				func() error { return requireLLMKey(cfg) },
			); err != nil {
				return err
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

			tryFile, err := retro.LoadTryFile(programTryPath(p.Program.ID))
			if err != nil {
				return fmt.Errorf("load try file: %w", err)
			}
			if items := retroTryItems(tryFile); items != nil {
				writer.SetRetroTry(write.FormatRetroTry(items))
				logger.Info("retro施策を注入", "count", len(items))
			}

			keepFile, err := retro.LoadKeepFile(programKeepPath(p.Program.ID))
			if err != nil {
				return fmt.Errorf("load keep file: %w", err)
			}
			if items := retroKeepItems(keepFile); items != nil {
				writer.SetRetroKeep(write.FormatRetroKeep(items))
				logger.Info("keep方針を注入", "count", len(items))
			}

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

			gatherer := gather.New(nil, gather.WithLogger(logger), gather.WithLocation(loc), gather.WithSanitizePolicy(cfg.Security.PromptInjection))
			summarizer := summarize.NewLLMSummarizer(llmClient, prompts["summarize"], stepTemp(cfg.LLM, "summarize"))
			rundowner := rundown.NewLLMRundowner(selector, summarizer, flowDesigner, excludedDedupKeys, rundown.WithLogger(logger))
			rundowner.SetCornerAppearances(cornerAppearances)

			scripter := script.NewLLMScriptGenerator(
				writer,
				direct.NewLLMDirector(llmClient, prompts["direct"], stepTemp(cfg.LLM, "direct"), cfg.Voicevox.EffectivePresets(),
					direct.WithProofread(prompts["proofread"], stepTemp(cfg.LLM, "proofread")),
					direct.WithPronunciation(cfg.EffectivePronunciation()),
				),
				assetCatalog,
				script.WithLogger(logger),
				script.WithRegenConfig(cfg.Script.EffectiveRegenThreshold(), cfg.Script.EffectiveRegenMaxRetries()),
			)

			runner := &pipeline.Runner{
				Spec:              p,
				Config:            cfg,
				Gatherer:          gatherer,
				ExcludedDedupKeys: excludedDedupKeys,
				Rundowner:         rundowner,
				Scripter:          scripter,
				Synther:           synth.New(engineURLs, cfg, synth.WithLogger(logger)),
				Mixer:             &mixerAdapter{inner: mix.New(p.Assets, p.Program, mix.WithLogger(logger), mix.WithFFmpegWriter(logFile))},
				ProgramSummarizer: programsummary.NewLLMProgramSummarizer(llmClient, prompts["summary"], stepTemp(cfg.LLM, "summary"), p.Program.EffectiveSummaryLength(), programsummary.WithLogger(logger)),
				CornerSummarizer:  programsummary.NewLLMCornerSummarizer(llmClient, prompts["corner_summary"], stepTemp(cfg.LLM, "corner_summary"), programsummary.WithLogger(logger)),
			}

			if err := runner.Run(ctx, pipeline.Options{
				OutDir:        outDir,
				EpisodeNumber: episodeNumber,
				Casts:         selectedCasts,
			}); err != nil {
				return err
			}

			// analyze は completed mp3 を捨てないよう失敗してもパイプライン全体は継続する
			// （ADR-0098）。single_shot でも 07_analysis.json は出力するが、キャッシュには
			// 保存しない（後段の !p.Program.SingleShot 分岐で appendToCache 自体を呼ばない）。
			if err := runAndSaveAnalysis(ctx, cfg, prompts, llmClient, p, layout, logger); err != nil {
				logger.Warn("分析に失敗（処理は継続）", "err", err)
			}

			// single_shot は連続性のない単発番組のためキャッシュへ保存しない
			// （次回の採番・past_episodes・feed 連載のいずれにも載せない）。
			if !p.Program.SingleShot {
				if err := appendToCache(cacheMgr, layout, cfg.Cache, cfg.Retro.EffectiveAnalysisEntries(), logger); err != nil {
					logger.Warn("キャッシュ追記に失敗（処理は継続）", "err", err)
				}
			}

			fmt.Printf("パイプライン完了: 番組を %s に出力しました\n", layout.Episode())
			return nil
		},
	}

	cmd.Flags().StringVar(&outDir, "out-dir", "output", "出力ディレクトリ（{program.id}_ep{NNN}.mp3 と {program.id}_ep{NNN}_manifest.json をここに配置し、中間ファイルは <out-dir>/intermediate/{program.id}_ep{NNN}/ に配置。single_shot 時は回番号サフィックス無し）")
	cmd.Flags().BoolVar(&force, "force", false, "既存の出力（mp3・マニフェスト・中間ディレクトリ）を上書きする")
	registerSpecFlag(cmd, &specPath)

	cmd.AddCommand(
		newGatherCmd(),
		newRundownCmd(),
		newScriptCmd(),
		newSynthCmd(),
		newMixCmd(),
		newManifestCmd(),
		newAnalyzeCmd(),
		newEpisodegenCheckCmd(),
	)

	return cmd
}

// scriptEntries is how many of the most recent entries keep their script (retro.analysis_entries;
// see cache.Compact).
func appendToCache(mgr *cache.Manager, layout fileio.EpisodeLayout, cacheCfg config.CacheConfig, scriptEntries int, logger *slog.Logger) error {
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
		logger.Warn("ファイルサイズ取得に失敗（処理は継続）", "err", err)
	} else {
		bytes = b
	}
	if d, err := mediainfo.Duration(episodePath); err != nil {
		logger.Warn("再生時間取得に失敗（処理は継続）", "err", err)
	} else {
		durationSec = int(d)
	}

	analysis, err := readOptionalJSON[model.Analysis](layout.Analysis())
	if err != nil {
		logger.Warn("分析ファイルの読み込みに失敗（処理は継続）", "err", err)
	}

	var lines model.ScriptLines
	if err := fileio.ReadJSON(layout.Lines(), &lines); err != nil {
		logger.Warn("台本ファイルの読み込みに失敗（処理は継続）", "err", err)
		lines = model.ScriptLines{}
	}

	entry := cache.BuildEntryFromManifest(layout.ProgramID, m, rd, bytes, durationSec, analysis, lines)
	if err := mgr.Append(entry, cacheCfg.EffectiveMaxEntries(), cacheCfg.EffectiveRetentionDays(), scriptEntries); err != nil {
		return err
	}
	logger.Info("キャッシュに追記", "program_id", layout.ProgramID)
	return nil
}
