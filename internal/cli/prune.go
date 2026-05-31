package cli

import (
	"context"
	"fmt"

	"github.com/canpok1/vox-radio/internal/config"
	"github.com/canpok1/vox-radio/internal/publish"
	"github.com/canpok1/vox-radio/internal/publish/hosting/local"
	"github.com/spf13/cobra"
)

func newPruneCmd() *cobra.Command {
	var outDir string
	var profilePath string
	var baseURL string

	cmd := &cobra.Command{
		Use:   "prune",
		Short: "Remove old episodes, keeping only the most recent N",
		Long: `Delete audio files and episode entries beyond the configured max_items limit,
then update episodes.json and regenerate feed.xml.

Example:
  vox-radio prune --out-dir public
  vox-radio prune --out-dir public --profile sample-profiles/tech/profile.yaml`,
		RunE: func(cmd *cobra.Command, args []string) error {
			p, err := config.LoadProfile(profilePath)
			if err != nil {
				return fmt.Errorf("load profile: %w", err)
			}

			keep := resolveKeep(p.Program.MaxItems)

			h := local.New(outDir, resolveSiteURL(baseURL, p.Program.SiteURL))
			pruner := publish.NewPruner(h, keep)

			if err := pruner.Run(context.Background()); err != nil {
				return err
			}

			fmt.Printf("pruned to %d episodes in %s\n", keep, outDir)
			return nil
		},
	}

	cmd.Flags().StringVar(&outDir, "out-dir", "", "output directory for local hosting (required)")
	cmd.Flags().StringVar(&profilePath, "profile", "", "profile YAML file path (required)")
	cmd.Flags().StringVar(&baseURL, "base-url", "", "base URL for audio/feed URLs (default: site_url from profile)")
	_ = cmd.MarkFlagRequired("out-dir")
	_ = cmd.MarkFlagRequired("profile")

	return cmd
}
