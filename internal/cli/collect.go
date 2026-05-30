package cli

import (
	"context"
	"fmt"

	"github.com/canpok1/vox-radio/internal/collect"
	"github.com/canpok1/vox-radio/internal/config"
	"github.com/spf13/cobra"
)

func newCollectCmd() *cobra.Command {
	var profilePath string
	var out string

	cmd := &cobra.Command{
		Use:   "collect",
		Short: "Collect articles from RSS feeds and URLs",
		Long: `Collect articles from RSS feeds and web URLs defined in the profile,
extract their body text, and write the result to articles.json.

Example:
  vox-radio collect --out work/articles.json
  vox-radio collect --out work/articles.json --profile profiles/tech/profile.yaml`,
		RunE: func(cmd *cobra.Command, args []string) error {
			p, err := config.LoadProfile(profilePath)
			if err != nil {
				return fmt.Errorf("load profile: %w", err)
			}

			c := collect.New(nil)
			articles, err := c.Run(context.Background(), config.FeedsConfig{
				Feeds:    p.Feeds,
				Articles: p.Articles,
			})
			if err != nil {
				return err
			}

			if err := writeJSON(out, articles); err != nil {
				return err
			}

			fmt.Printf("collected %d articles to %s\n", len(articles.Articles), out)
			return nil
		},
	}

	cmd.Flags().StringVar(&profilePath, "profile", "profiles/test/profile.yaml", "profile YAML file path")
	cmd.Flags().StringVar(&out, "out", "", "output articles.json path (required)")
	_ = cmd.MarkFlagRequired("out")

	return cmd
}
