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
		Short: "Run the full podcast production pipeline",
		Long: `Run collect → script → synth → assemble → manifest in one shot.

Intermediate files are written to <out-dir>/intermediate/ and the final
episode.mp3 is placed directly under <out-dir>/.

vox-radio.yaml is automatically loaded from the current directory.

Example:
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
				Profile:   p,
				Config:    cfg,
				Collector: collect.New(nil),
				Scripter:  scripter,
				Synther:   synth.New(engineURL, cfg),
				Assembler: assemble.New(p.Assets, p.Program),
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

	cmd.Flags().StringVar(&outDir, "out-dir", "output", "output directory (episode.mp3 placed here, intermediate files in <out-dir>/intermediate/)")
	registerProfileFlag(cmd, &profilePath)
	cmd.Flags().StringVar(&promptsDir, "prompts", "prompts", "directory containing prompt templates")

	return cmd
}
