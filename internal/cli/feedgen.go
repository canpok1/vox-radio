package cli

import (
	"fmt"

	"github.com/canpok1/vox-radio/internal/feed"
	"github.com/spf13/cobra"
)

func newFeedgenCmd() *cobra.Command {
	var cachePath string
	var specPath string

	cmd := &cobra.Command{
		Use:   "feedgen",
		Short: "キャッシュから RSS フィード（feed.xml）を生成する",
		Long: `cache ファイルと feed-spec.yaml から RSS 2.0 + iTunes フィード（feed.xml）を生成します。

cache はエピソード状態の正データです。manifest や mp3 は必要ありません。
生成された feed.xml は feed-spec.yaml の output.public ディレクトリに書き出されます。

例:
  vox-radio feedgen --cache .vox-radio/cache/zundamon-tech-radio.jsonl --spec config/feed-spec.yaml`,
		RunE: func(cmd *cobra.Command, args []string) error {
			path, n, err := feed.Run(feed.Options{
				CachePath: cachePath,
				SpecPath:  specPath,
			})
			if err != nil {
				return err
			}
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "feed.xml written to %s (%d items)\n", path, n)
			return nil
		},
	}

	cmd.Flags().StringVar(&cachePath, "cache", "", "キャッシュ JSONL ファイルのパス（必須）")
	cmd.Flags().StringVar(&specPath, "spec", "", "feed-spec.yaml ファイルのパス（必須）")
	_ = cmd.MarkFlagRequired("cache")
	_ = cmd.MarkFlagRequired("spec")

	return cmd
}
