package cli

import (
	"embed"

	"github.com/spf13/cobra"
)

//go:embed all:templates
var templatesFS embed.FS

//go:embed all:templates-sample
var sampleFS embed.FS

func newInitCmd() *cobra.Command {
	var sample bool

	cmd := &cobra.Command{
		Use:   "init",
		Short: "カレントディレクトリにテンプレート設定ファイルを生成する",
		Long: `vox-radio.yaml（共通設定）・episode-spec.yaml（エピソード仕様）・feed-spec.yaml（フィード生成設定）・slack-spec.yaml（Slack 投稿設定）・assets/assets.yaml（アセット設定）を
カレントディレクトリに生成します。

--sample を指定すると、ずんだもん・めたんが MC を務めるお天気番組（気象庁の防災情報XMLを利用）の
「すぐ動くサンプル設定一式」を sample/ ディレクトリに生成します。生成されるのは
sample/vox-radio.yaml・sample/episode-spec.yaml・sample/feed-spec.yaml・
sample/slack-spec.yaml・sample/assets/assets.yaml の 5 ファイルです。生成後は次のコマンドで
番組生成を試せます:

  vox-radio --config sample/vox-radio.yaml episodegen --spec sample/episode-spec.yaml

既存ファイルは上書きを防ぐため個別にスキップされます。
すべてのファイルがすでに存在する場合は何も生成されません。

生成後は LLM API キー・番組内容・音声アセットパスを設定ファイルに記入し、
次のコマンドでパイプラインを実行してください:

  vox-radio episodegen --spec episode-spec.yaml`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if sample {
				return writeEmbeddedTree(cmd, sampleFS, "templates-sample", "sample", false)
			}
			return writeEmbeddedTree(cmd, templatesFS, "templates", ".", false)
		},
	}

	cmd.Flags().BoolVar(&sample, "sample", false, "ずんだもん・めたんMCのお天気番組サンプル一式を sample/ に生成する")

	return cmd
}
