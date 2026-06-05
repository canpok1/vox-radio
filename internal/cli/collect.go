package cli

import (
	"context"
	"fmt"

	"github.com/canpok1/vox-radio/internal/collect"
	"github.com/canpok1/vox-radio/internal/config"
	"github.com/spf13/cobra"
)

func newCollectCmd() *cobra.Command {
	var specPath string
	var out string

	cmd := &cobra.Command{
		Use:   "collect",
		Short: "コーナーごとに RSS/Atom フィードと URL から記事を収集する",
		Long: `corners[].source に定義された RSS/Atom フィードや Web URL から記事を収集し、
本文テキストを抽出して articles.json に書き出します。

source フィールドのないコーナーはスキップされます。

例:
  vox-radio episodegen collect --out work/articles.json
  vox-radio episodegen collect --out work/articles.json --spec examples/tech.yaml`,
		RunE: func(cmd *cobra.Command, args []string) error {
			logger, logFile, err := setupLogger("collect", "")
			if err != nil {
				return fmt.Errorf("setup logger: %w", err)
			}
			defer func() { _ = logFile.Close() }()

			p, err := config.LoadEpisodeSpec(specPath)
			if err != nil {
				return fmt.Errorf("load spec: %w", err)
			}

			if err := config.ValidateEpisodeSpecCorners(p); err != nil {
				return fmt.Errorf("spec validation: %w", err)
			}

			// 回番号を持たないため全コーナーを superset として収集する（rundown 側で絞る）
			c := collect.New(nil, collect.WithLogger(logger))
			articles, err := c.RunAll(context.Background(), p.Corners, nil)
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

	registerSpecFlag(cmd, &specPath)
	cmd.Flags().StringVar(&out, "out", "", "articles.json の出力先パス（必須）")
	_ = cmd.MarkFlagRequired("out")

	return cmd
}
