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
	var configDir string
	var baseURL string

	cmd := &cobra.Command{
		Use:   "prune",
		Short: "Remove old episodes, keeping only the most recent N",
		Long: `Delete audio files and episode entries beyond the configured max_items limit,
then update episodes.json and regenerate feed.xml.

Example:
  vox-radio prune --out-dir public
  vox-radio prune --out-dir public --config config`,
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load(configDir)
			if err != nil {
				return fmt.Errorf("load config: %w", err)
			}

			keep := cfg.Podcast.MaxItems
			if keep <= 0 {
				keep = publish.DefaultKeep
			}

			h := local.New(outDir, resolveSiteURL(baseURL, cfg.Podcast.SiteURL))
			pruner := publish.NewPruner(h, keep)

			if err := pruner.Run(context.Background()); err != nil {
				return err
			}

			fmt.Printf("pruned to %d episodes in %s\n", keep, outDir)
			return nil
		},
	}

	cmd.Flags().StringVar(&outDir, "out-dir", "", "output directory for local hosting (required)")
	cmd.Flags().StringVar(&configDir, "config", "config", "config directory containing podcast.yaml")
	cmd.Flags().StringVar(&baseURL, "base-url", "", "base URL for audio/feed URLs (default: site_url from podcast.yaml)")
	_ = cmd.MarkFlagRequired("out-dir")

	return cmd
}

func resolveSiteURL(override, configURL string) string {
	if override != "" {
		return override
	}
	return configURL
}
