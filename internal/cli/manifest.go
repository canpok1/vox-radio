package cli

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/canpok1/vox-radio/internal/config"
	"github.com/canpok1/vox-radio/internal/manifest"
	"github.com/canpok1/vox-radio/internal/model"
	"github.com/canpok1/vox-radio/internal/script/llm"
	programsummary "github.com/canpok1/vox-radio/internal/script/summary"
	"github.com/spf13/cobra"
)

func newManifestCmd() *cobra.Command {
	var profilePath string
	var articlesPath string
	var audioPath string
	var out string
	var scriptPath string
	var promptsDir string

	cmd := &cobra.Command{
		Use:   "manifest",
		Short: "Generate a content manifest JSON alongside an episode",
		Long: `Build a manifest.json that describes the episode content: title, description,
summary, datetime, audio filename, and corners with their articles.

The manifest is intended for use by a separate publishing service to generate
RSS feeds without re-running the full pipeline.

When --script is provided, an LLM-generated summary is added to the manifest
using vox-radio.yaml for LLM configuration.

Example:
  vox-radio manifest --profile sample-profiles/tech_profile.yaml --audio output/episode.mp3 --out output/manifest.json
  vox-radio manifest --profile sample-profiles/tech_profile.yaml --articles output/intermediate/articles.json --audio output/episode.mp3 --out output/manifest.json
  vox-radio manifest --profile sample-profiles/tech_profile.yaml --script output/intermediate/script.json --audio output/episode.mp3 --out output/manifest.json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			p, err := config.LoadProfile(profilePath)
			if err != nil {
				return fmt.Errorf("load profile: %w", err)
			}

			var articles model.Articles
			if articlesPath != "" {
				var err error
				articles, err = readJSON[model.Articles](articlesPath)
				if err != nil {
					return fmt.Errorf("read articles: %w", err)
				}
			}

			var programSummary string
			if scriptPath != "" {
				scr, err := readJSON[model.Script](scriptPath)
				if err != nil {
					return fmt.Errorf("read script: %w", err)
				}

				cfg, err := config.LoadConfig("vox-radio.yaml")
				if err != nil {
					return fmt.Errorf("load config: %w", err)
				}

				summaryPromptData, err := os.ReadFile(filepath.Join(promptsDir, "summary.md"))
				if err != nil {
					return fmt.Errorf("read summary.md: %w", err)
				}

				apiKey := os.Getenv(cfg.LLM.APIKeyEnv)
				llmClient := llm.NewClient(llm.Config{
					BaseURL:     cfg.LLM.BaseURL,
					APIKey:      apiKey,
					Model:       cfg.LLM.Model,
					Temperature: cfg.LLM.Temperature,
					MaxRetries:  cfg.LLM.MaxRetries,
				})

				s := programsummary.NewLLMProgramSummarizer(llmClient, string(summaryPromptData), stepTemp(cfg.LLM, "summary"))
				programSummary, err = s.Summarize(context.Background(), scr)
				if err != nil {
					return fmt.Errorf("summarize program: %w", err)
				}
			}

			m := manifest.Build(p.Program, p.Corners, articles, filepath.Base(audioPath), time.Now().UTC(), programSummary)

			if err := writeJSON(out, m); err != nil {
				return err
			}

			fmt.Printf("manifest written to %s\n", out)
			return nil
		},
	}

	registerProfileFlag(cmd, &profilePath)
	cmd.Flags().StringVar(&articlesPath, "articles", "", "articles.json path (optional; corners get empty articles when omitted)")
	cmd.Flags().StringVar(&audioPath, "audio", "", "audio file path; basename is stored in manifest (required)")
	cmd.Flags().StringVar(&out, "out", "", "output manifest.json path (required)")
	cmd.Flags().StringVar(&scriptPath, "script", "", "script.json path (optional; when provided, LLM generates a summary from the script)")
	cmd.Flags().StringVar(&promptsDir, "prompts", "prompts", "directory containing prompt templates (used when --script is provided)")
	_ = cmd.MarkFlagRequired("audio")
	_ = cmd.MarkFlagRequired("out")

	return cmd
}
