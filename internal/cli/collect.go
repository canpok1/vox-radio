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
		Short: "Collect articles from RSS feeds and URLs per corner",
		Long: `Collect articles from RSS feeds and web URLs defined in corners[].source,
extract their body text, and write the result to articles.json.

Corners without a source field are skipped.

Example:
  vox-radio collect --out work/articles.json
  vox-radio collect --out work/articles.json --profile sample-profiles/tech_profile.yaml`,
		RunE: func(cmd *cobra.Command, args []string) error {
			p, err := config.LoadProfile(profilePath)
			if err != nil {
				return fmt.Errorf("load profile: %w", err)
			}

			c := collect.New(nil)
			articles, err := c.RunAll(context.Background(), p.Corners)
			if err != nil {
				return err
			}

			if err := writeJSON(out, articles); err != nil {
				return err
			}

			total := 0
			for _, ca := range articles.Corners {
				total += len(ca.Articles)
			}
			fmt.Printf("collected %d articles across %d corners to %s\n", total, len(articles.Corners), out)
			return nil
		},
	}

	registerProfileFlag(cmd, &profilePath)
	cmd.Flags().StringVar(&out, "out", "", "output articles.json path (required)")
	_ = cmd.MarkFlagRequired("out")

	return cmd
}
