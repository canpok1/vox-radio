package cli

import (
	"embed"
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

//go:embed templates/*.yaml
var templatesFS embed.FS

func newInitCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "init",
		Short: "カレントディレクトリにテンプレート設定ファイルを生成する",
		Long: `vox-radio.yaml（共通設定）・episode-spec.yaml（エピソード仕様）・feed-spec.yaml（フィード生成設定）・slack-spec.yaml（Slack 投稿設定）を
カレントディレクトリに生成します。

既存ファイルは上書きを防ぐため個別にスキップされます。
すべてのファイルがすでに存在する場合は何も生成されません。

生成後は LLM API キー・番組内容・音声アセットパスを設定ファイルに記入し、
次のコマンドでパイプラインを実行してください:

  vox-radio episodegen --spec episode-spec.yaml`,
		RunE: func(cmd *cobra.Command, args []string) error {
			entries, err := templatesFS.ReadDir("templates")
			if err != nil {
				return fmt.Errorf("read templates: %w", err)
			}
			for _, e := range entries {
				content, err := templatesFS.ReadFile("templates/" + e.Name())
				if err != nil {
					return fmt.Errorf("read template %s: %w", e.Name(), err)
				}
				if err := generateFile(cmd, e.Name(), content); err != nil {
					return err
				}
			}
			return nil
		},
	}
}

func generateFile(cmd *cobra.Command, path string, content []byte) error {
	if _, err := os.Stat(path); err == nil {
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "skip: %s already exists\n", path)
		return nil
	}
	if err := os.WriteFile(path, content, 0644); err != nil {
		return fmt.Errorf("write %s: %w", path, err)
	}
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "created: %s\n", path)
	return nil
}
