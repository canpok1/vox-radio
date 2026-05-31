package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/canpok1/vox-radio/internal/config"
	"github.com/canpok1/vox-radio/internal/manifest"
	"github.com/canpok1/vox-radio/internal/model"
	"github.com/spf13/cobra"
)

func newManifestCmd() *cobra.Command {
	var profilePath string
	var articlesPath string
	var audioPath string
	var out string

	cmd := &cobra.Command{
		Use:   "manifest",
		Short: "Generate a content manifest JSON alongside an episode",
		Long: `Build a manifest.json that describes the episode content: title, description,
datetime, audio filename, and corners with their articles.

The manifest is intended for use by a separate publishing service to generate
RSS feeds without re-running the full pipeline.

Example:
  vox-radio manifest --profile profiles/tech/profile.yaml --audio output/episode.mp3 --out output/manifest.json
  vox-radio manifest --profile profiles/tech/profile.yaml --articles output/intermediate/articles.json --audio output/episode.mp3 --out output/manifest.json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			p, err := config.LoadProfile(profilePath)
			if err != nil {
				return fmt.Errorf("load profile: %w", err)
			}

			var articles model.Articles
			if articlesPath != "" {
				data, err := os.ReadFile(articlesPath)
				if err != nil {
					return fmt.Errorf("read articles: %w", err)
				}
				if err := json.Unmarshal(data, &articles); err != nil {
					return fmt.Errorf("parse articles: %w", err)
				}
			}

			m := manifest.Build(p.Program, p.Corners, articles, filepath.Base(audioPath), time.Now().UTC())

			if err := writeJSON(out, m); err != nil {
				return err
			}

			fmt.Printf("manifest written to %s\n", out)
			return nil
		},
	}

	cmd.Flags().StringVar(&profilePath, "profile", "profiles/test/profile.yaml", "profile YAML file path")
	cmd.Flags().StringVar(&articlesPath, "articles", "", "articles.json path (optional; corners get empty articles when omitted)")
	cmd.Flags().StringVar(&audioPath, "audio", "", "audio file path; basename is stored in manifest (required)")
	cmd.Flags().StringVar(&out, "out", "", "output manifest.json path (required)")
	_ = cmd.MarkFlagRequired("audio")
	_ = cmd.MarkFlagRequired("out")

	return cmd
}
