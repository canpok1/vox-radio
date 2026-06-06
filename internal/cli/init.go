package cli

import (
	"embed"
	"fmt"
	"io/fs"
	"path/filepath"

	"github.com/spf13/cobra"
)

//go:embed all:templates
var templatesFS embed.FS

func newInitCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "init",
		Short: "カレントディレクトリにテンプレート設定ファイルを生成する",
		Long: `vox-radio.yaml（共通設定）・episode-spec.yaml（エピソード仕様）・feed-spec.yaml（フィード生成設定）・slack-spec.yaml（Slack 投稿設定）・assets/assets.yaml（アセット設定）を
カレントディレクトリに生成します。

既存ファイルは上書きを防ぐため個別にスキップされます。
すべてのファイルがすでに存在する場合は何も生成されません。

生成後は LLM API キー・番組内容・音声アセットパスを設定ファイルに記入し、
次のコマンドでパイプラインを実行してください:

  vox-radio episodegen --spec episode-spec.yaml`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return fs.WalkDir(templatesFS, "templates", func(path string, d fs.DirEntry, err error) error {
				if err != nil {
					return err
				}
				if d.IsDir() {
					return nil
				}
				content, err := templatesFS.ReadFile(path)
				if err != nil {
					return fmt.Errorf("read template %s: %w", path, err)
				}
				outPath, err := filepath.Rel("templates", path)
				if err != nil {
					return fmt.Errorf("rel path: %w", err)
				}
				return writeFile(cmd, outPath, content, false)
			})
		},
	}
}
