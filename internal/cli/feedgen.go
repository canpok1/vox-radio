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

	cmd.AddCommand(newFeedgenCheckCmd())

	return cmd
}

func newFeedgenCheckCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "check <path>",
		Short: "feed-spec.yaml を strict モードでフル検証する",
		Long: `指定した feed-spec.yaml を strict モードでパースし、以下を検証します:

  (a) strict パース: 未知キー（typo）をエラー化
  (b) 必須フィールド: feed.language / feed.author / feed.email /
      feed.site_url / feed.audio_url_template の存在チェック
  (c) URL / email 形式: 各フィールドの値が正しい形式かチェック
  (d) プレースホルダ: audio_url_template に {episode_number} と {audio_file} が含まれるかチェック

意味検証エラー (b)(c)(d) は全件収集してまとめて報告します。

成功時は標準出力に OK メッセージを出力し、ゼロで終了します。
失敗時は非ゼロで終了します（CI での自動検知に使用できます）。`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			path := args[0]

			spec, err := feed.LoadFeedSpecStrict(path)
			if err != nil {
				return err
			}

			if err := feed.ValidateFeedSpec(spec); err != nil {
				return err
			}

			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "OK: %s\n", path)
			return nil
		},
	}
}
