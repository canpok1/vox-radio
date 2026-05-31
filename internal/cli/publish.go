package cli

import (
	"context"
	"fmt"

	"github.com/canpok1/vox-radio/internal/config"
	"github.com/canpok1/vox-radio/internal/publish"
	"github.com/canpok1/vox-radio/internal/publish/hosting"
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
	var hostingType string

	cmd := &cobra.Command{
		Use:   "publish",
		Short: "Publish an episode to the hosting directory",
		Long: `Copy the MP3 file into the hosting directory, update episodes.json,
and regenerate feed.xml for RSS distribution.

Hosting types:
  local    Write files to a local directory (default).
  ghpages  Write files to a local git working tree and push to gh-pages as an orphan commit.

Example:
  vox-radio publish --in work/episode.mp3 --out-dir public
  vox-radio publish --in work/episode.mp3 --out-dir public --date 2026-01-01 --title "Episode title"
  vox-radio publish --in work/episode.mp3 --out-dir public --profile sample-profiles/tech/profile.yaml
  vox-radio publish --in work/episode.mp3 --out-dir public --hosting ghpages`,
		RunE: func(cmd *cobra.Command, args []string) error {
			p, err := config.LoadProfile(profilePath)
			if err != nil {
				return fmt.Errorf("load profile: %w", err)
			}

			siteURL := resolveSiteURL(baseURL, p.Program.SiteURL)
			opts := publish.Options{
				Date:        date,
				Title:       titleFlag,
				Description: descFlag,
			}

			h, err := newHosting(hostingType, outDir, siteURL)
			if err != nil {
				return err
			}

			ctx := context.Background()
			publisher := publish.New(h, p.Program)
			if err := publisher.Run(ctx, in, opts); err != nil {
				return err
			}

			if pusher, ok := h.(hosting.Pusher); ok {
				if err := pusher.Push(ctx); err != nil {
					return fmt.Errorf("push to gh-pages: %w", err)
				}
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
	cmd.Flags().StringVar(&titleFlag, "title", "", "episode title (default: <date> <program.title>)")
	cmd.Flags().StringVar(&descFlag, "description", "", "episode description")
	cmd.Flags().StringVar(&profilePath, "profile", "", "profile YAML file path (required)")
	cmd.Flags().StringVar(&outDir, "out-dir", "", "output directory for hosting (required)")
	cmd.Flags().StringVar(&baseURL, "base-url", "", "base URL for audio/feed URLs (default: site_url from profile)")
	cmd.Flags().StringVar(&hostingType, "hosting", "local", "hosting type: local or ghpages")
	_ = cmd.MarkFlagRequired("in")
	_ = cmd.MarkFlagRequired("profile")
	_ = cmd.MarkFlagRequired("out-dir")

	return cmd
}
