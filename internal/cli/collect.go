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
		Short: "コーナーごとに RSS/Atom フィードと URL から記事を収集する",
		Long: `corners[].source に定義された RSS/Atom フィードや Web URL から記事を収集し、
本文テキストを抽出して articles.json に書き出します。

source フィールドのないコーナーはスキップされます。

例:
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
	cmd.Flags().StringVar(&out, "out", "", "articles.json の出力先パス（必須）")
	_ = cmd.MarkFlagRequired("out")

	return cmd
}
