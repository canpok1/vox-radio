package cli

import (
	"context"
	"fmt"
	"os"

	"github.com/canpok1/vox-radio/internal/assemble"
	"github.com/canpok1/vox-radio/internal/collect"
	"github.com/canpok1/vox-radio/internal/fileio"
	"github.com/canpok1/vox-radio/internal/pipeline"
	"github.com/canpok1/vox-radio/internal/script"
	"github.com/canpok1/vox-radio/internal/script/direct"
	"github.com/canpok1/vox-radio/internal/script/llm"
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
		Long: `collect → script → synth → assemble → manifest を一括実行します。

中間ファイルは <out-dir>/intermediate/ に書き出され、
最終的な episode.mp3 は <out-dir>/ 直下に配置されます。

vox-radio.yaml はカレントディレクトリから自動読み込みされます。

例:
  vox-radio run
  vox-radio run --out-dir output --profile sample-profiles/tech_profile.yaml`,
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, p, err := loadConfigAndProfile(profilePath)
			if err != nil {
				return err
			}

			apiKey := os.Getenv(cfg.LLM.APIKeyEnv)
			llmClient := llm.NewClient(llm.Config{
				BaseURL:     cfg.LLM.BaseURL,
				APIKey:      apiKey,
				Model:       cfg.LLM.Model,
				Temperature: cfg.LLM.Temperature,
				MaxRetries:  cfg.LLM.MaxRetries,
			})

			prompts, err := loadPrompts(promptsDir)
			if err != nil {
				return fmt.Errorf("load prompts: %w", err)
			}

			seCatalog := buildSECatalog(p.Assets)
			intermediateDir := fileio.IntermediateDir(outDir)

			scripter := script.NewLLMScriptGenerator(
				summarize.NewLLMSummarizer(llmClient, prompts["summarize"], stepTemp(cfg.LLM, "summarize")),
				write.NewLLMWriter(llmClient, prompts["write"], stepTemp(cfg.LLM, "write")),
				direct.NewLLMDirector(llmClient, prompts["direct"], stepTemp(cfg.LLM, "direct")),
				seCatalog,
				intermediateDir,
			)

			engineURL := cfg.Voicevox.URL
			if engineURL == "" {
				engineURL = "http://localhost:50021"
			}

			runner := &pipeline.Runner{
				Profile:           p,
				Config:            cfg,
				Collector:         collect.New(nil),
				Scripter:          scripter,
				Synther:           synth.New(engineURL, cfg),
				Assembler:         assemble.New(p.Assets, p.Program),
				ProgramSummarizer: programsummary.NewLLMProgramSummarizer(llmClient, prompts["summary"], stepTemp(cfg.LLM, "summary")),
			}

			if err := runner.Run(context.Background(), pipeline.Options{
				OutDir: outDir,
			}); err != nil {
				return err
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
