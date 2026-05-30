package cli

import (
	"context"
	"fmt"

	"github.com/canpok1/vox-radio/internal/config"
	"github.com/canpok1/vox-radio/internal/publish"
	"github.com/canpok1/vox-radio/internal/publish/hosting/local"
	"github.com/spf13/cobra"
)

func newPublishCmd() *cobra.Command {
	var in string
	var date string
	var titleFlag string
	var descFlag string
	var profilePath string
	var outDir string
	var baseURL string

	cmd := &cobra.Command{
		Use:   "publish",
		Short: "Publish an episode to the local hosting directory",
		Long: `Copy the MP3 file into the hosting directory, update episodes.json,
and regenerate feed.xml for RSS distribution.

Example:
  vox-radio publish --in work/episode.mp3 --out-dir public
  vox-radio publish --in work/episode.mp3 --out-dir public --date 2026-01-01 --title "Episode title"
  vox-radio publish --in work/episode.mp3 --out-dir public --profile profiles/tech/profile.yaml`,
		RunE: func(cmd *cobra.Command, args []string) error {
			p, err := config.LoadProfile(profilePath)
			if err != nil {
				return fmt.Errorf("load profile: %w", err)
			}

			h := local.New(outDir, resolveSiteURL(baseURL, p.Podcast.SiteURL))
			publisher := publish.New(h, p.Podcast)

			opts := publish.Options{
				Date:        date,
				Title:       titleFlag,
				Description: descFlag,
			}

			if err := publisher.Run(context.Background(), in, opts); err != nil {
				return err
			}

			effectiveDate := date
			if effectiveDate == "" {
				effectiveDate = "(today)"
			}
			fmt.Printf("published episode for %s to %s\n", effectiveDate, outDir)
			return nil
		},
	}

	cmd.Flags().StringVar(&in, "in", "", "input mp3 path (required)")
	cmd.Flags().StringVar(&date, "date", "", "episode date YYYY-MM-DD (default: today)")
	cmd.Flags().StringVar(&titleFlag, "title", "", "episode title (default: <date> <podcast.title>)")
	cmd.Flags().StringVar(&descFlag, "description", "", "episode description")
	cmd.Flags().StringVar(&profilePath, "profile", "profiles/test/profile.yaml", "profile YAML file path")
	cmd.Flags().StringVar(&outDir, "out-dir", "", "output directory for local hosting (required)")
	cmd.Flags().StringVar(&baseURL, "base-url", "", "base URL for audio/feed URLs (default: site_url from profile)")
	_ = cmd.MarkFlagRequired("in")
	_ = cmd.MarkFlagRequired("out-dir")

	return cmd
}
