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
	var outputDir string

	cmd := &cobra.Command{
		Use:   "init",
		Short: "テンプレート設定ファイルを生成する",
		Long: `vox-radio.yaml（共通設定）・episode-spec.yaml（エピソード仕様）・feed-spec.yaml（フィード生成設定）・slack-spec.yaml（Slack 投稿設定）・assets/assets.yaml（アセット設定）を
生成します。出力先は --output-dir で指定します（省略時はカレントディレクトリ）。

--sample を指定すると、ずんだもん・めたんが MC を務めるお天気番組（気象庁の防災情報XMLを利用）の
「すぐ動くサンプル設定一式」のテンプレートを生成します。--sample を使っても出力先は --output-dir で
決まり、省略時はカレントディレクトリです。旧来の sample/ 配下への出力は次のコマンドで再現できます:

  vox-radio init --sample --output-dir sample

生成後は次のコマンドで番組生成を試せます:

  vox-radio episodegen --spec episode-spec.yaml

既存ファイルは上書きを防ぐため個別にスキップされます。
すべてのファイルがすでに存在する場合は何も生成されません。

生成後は LLM API キー・番組内容・音声アセットパスを設定ファイルに記入し、
次のコマンドでパイプラインを実行してください:

  vox-radio episodegen --spec episode-spec.yaml`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if sample {
				return writeEmbeddedTree(cmd, sampleFS, "templates-sample", outputDir, false)
			}
			return writeEmbeddedTree(cmd, templatesFS, "templates", outputDir, false)
		},
	}

	cmd.Flags().BoolVar(&sample, "sample", false, "ずんだもん・めたんMCのお天気番組サンプル一式のテンプレートを生成する")
	cmd.Flags().StringVar(&outputDir, "output-dir", ".", "テンプレートの出力先ディレクトリ（省略時はカレントディレクトリ）")

	return cmd
}
