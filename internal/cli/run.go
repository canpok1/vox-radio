package cli

import (
	"context"
	"fmt"
	"log/slog"
	"path/filepath"

	"github.com/canpok1/vox-radio/internal/assemble"
	"github.com/canpok1/vox-radio/internal/cache"
	"github.com/canpok1/vox-radio/internal/collect"
	"github.com/canpok1/vox-radio/internal/config"
	"github.com/canpok1/vox-radio/internal/fileio"
	"github.com/canpok1/vox-radio/internal/model"
	"github.com/canpok1/vox-radio/internal/pipeline"
	"github.com/canpok1/vox-radio/internal/rundown"
	sel "github.com/canpok1/vox-radio/internal/rundown/select"
	"github.com/canpok1/vox-radio/internal/script"
	"github.com/canpok1/vox-radio/internal/script/direct"
	"github.com/canpok1/vox-radio/internal/script/summarize"
	programsummary "github.com/canpok1/vox-radio/internal/script/summary"
	"github.com/canpok1/vox-radio/internal/script/write"
	"github.com/canpok1/vox-radio/internal/synth"
	"github.com/spf13/cobra"
)

func newRunCmd() *cobra.Command {
	var outDir string
	var profilePath string
	var promptsDir string

	cmd := &cobra.Command{
		Use:   "run",
		Short: "ポッドキャスト制作パイプラインをすべて実行する",
		Long: `collect → rundown → script → synth → assemble → manifest を一括実行します。

中間ファイルは <out-dir>/intermediate/ に書き出され、
最終的な episode.mp3 は <out-dir>/ 直下に配置されます。

vox-radio.yaml はカレントディレクトリから自動読み込みされます。

例:
  vox-radio run
  vox-radio run --out-dir output --profile sample-profiles/tech_profile.yaml`,
		RunE: func(cmd *cobra.Command, args []string) error {
			logger, logFile, err := setupLogger("run", "")
			if err != nil {
				return fmt.Errorf("setup logger: %w", err)
			}
			defer func() { _ = logFile.Close() }()

			cfg, p, err := loadConfigAndProfile(profilePath)
			if err != nil {
				return err
			}

			llmClient := newLLMClient(cfg)

			prompts, err := loadPrompts(promptsDir)
			if err != nil {
				return fmt.Errorf("load prompts: %w", err)
			}

			assetCatalog := buildAssetCatalog(p.Assets)
			intermediateDir := fileio.IntermediateDir(outDir)

			selector := sel.NewLLMSelector(llmClient, prompts["select"], stepTemp(cfg.LLM, "select"))
			writer := write.NewLLMWriter(llmClient, prompts["write"], stepTemp(cfg.LLM, "write"), cfg)

			var cacheMgr *cache.Manager
			if cfg.Cache.Enabled && p.Program.ID != "" {
				cachePath := filepath.Join(".vox-radio", "cache", p.Program.ID+".jsonl")
				cacheMgr = cache.New(cachePath)
				entries, err := cacheMgr.Load()
				if err != nil {
					return fmt.Errorf("load cache: %w", err)
				}
				recent := cache.Recent(entries, cfg.Cache.EffectiveLLMContextEntries())
				selector.SetPastURLs(cache.PastURLs(recent))
				writer.SetPastEpisodes(recent)
			}

			summarizer := summarize.NewLLMSummarizer(llmClient, prompts["summarize"], stepTemp(cfg.LLM, "summarize"))
			rundowner := rundown.NewLLMRundowner(selector, summarizer)

			scripter := script.NewLLMScriptGenerator(
				writer,
				direct.NewLLMDirector(llmClient, prompts["direct"], stepTemp(cfg.LLM, "direct")),
				assetCatalog,
				intermediateDir,
				script.WithLogger(logger),
			)

			engineURL := cfg.Voicevox.URL
			if engineURL == "" {
				engineURL = "http://localhost:50021"
			}

			runner := &pipeline.Runner{
				Profile:           p,
				Config:            cfg,
				Collector:         collect.New(nil, collect.WithLogger(logger)),
				Rundowner:         rundowner,
				Scripter:          scripter,
				Synther:           synth.New(engineURL, cfg, synth.WithLogger(logger)),
				Assembler:         assemble.New(p.Assets, p.Program, assemble.WithLogger(logger), assemble.WithFFmpegWriter(logFile)),
				ProgramSummarizer: programsummary.NewLLMProgramSummarizer(llmClient, prompts["summary"], stepTemp(cfg.LLM, "summary")),
				Logger:            logger,
			}

			if err := runner.Run(context.Background(), pipeline.Options{
				OutDir: outDir,
			}); err != nil {
				return err
			}

			if cacheMgr != nil {
				if err := appendToCache(cacheMgr, p.Program.ID, outDir, cfg.Cache, logger); err != nil {
					logger.Warn(fmt.Sprintf("cache append failed (non-fatal): %v", err))
				}
			}

			fmt.Printf("pipeline complete: episode at %s\n", fileio.EpisodePath(outDir))
			return nil
		},
	}

	cmd.Flags().StringVar(&outDir, "out-dir", "output", "出力ディレクトリ（episode.mp3 をここに配置し、中間ファイルは <out-dir>/intermediate/ に配置）")
	registerProfileFlag(cmd, &profilePath)
	cmd.Flags().StringVar(&promptsDir, "prompts", "prompts", "プロンプトテンプレートを含むディレクトリ")

	return cmd
}

func appendToCache(mgr *cache.Manager, programID string, outDir string, cacheCfg config.CacheConfig, logger *slog.Logger) error {
	var m model.Manifest
	if err := fileio.ReadJSON(fileio.ManifestPath(outDir), &m); err != nil {
		return fmt.Errorf("read manifest: %w", err)
	}
	var rd model.Rundown
	if err := fileio.ReadJSON(fileio.RundownPath(outDir), &rd); err != nil {
		return fmt.Errorf("read rundown: %w", err)
	}

	entry := cache.BuildEntryFromManifest(programID, m, rd)
	if err := mgr.Append(entry, cacheCfg.EffectiveMaxEntries(), cacheCfg.EffectiveRetentionDays()); err != nil {
		return err
	}
	logger.Info("cache entry appended", "program_id", programID)
	return nil
}
