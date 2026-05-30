package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/canpok1/vox-radio/internal/collect"
	"github.com/canpok1/vox-radio/internal/config"
	"github.com/spf13/cobra"
)

func newCollectCmd() *cobra.Command {
	var configDir string
	var out string

	cmd := &cobra.Command{
		Use:   "collect",
		Short: "Collect articles from RSS feeds and URLs",
		Long: `Collect articles from RSS feeds and web URLs defined in feeds.yaml,
extract their body text, and write the result to articles.json.

Example:
  vox-radio collect --out work/articles.json
  vox-radio collect --out work/articles.json --config config`,
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load(configDir)
			if err != nil {
				return fmt.Errorf("load config: %w", err)
			}

			c := collect.New(nil)
			articles, err := c.Run(context.Background(), cfg.Feeds)
			if err != nil {
				return err
			}

			if err := os.MkdirAll(filepath.Dir(out), 0o755); err != nil {
				return fmt.Errorf("create output dir: %w", err)
			}

			data, err := json.MarshalIndent(articles, "", "  ")
			if err != nil {
				return fmt.Errorf("marshal articles: %w", err)
			}
			if err := os.WriteFile(out, data, 0o644); err != nil {
				return fmt.Errorf("write articles: %w", err)
			}

			fmt.Printf("collected %d articles to %s\n", len(articles.Articles), out)
			return nil
		},
	}

	cmd.Flags().StringVar(&configDir, "config", "config", "config directory containing feeds.yaml")
	cmd.Flags().StringVar(&out, "out", "", "output articles.json path (required)")
	_ = cmd.MarkFlagRequired("out")

	return cmd
}
