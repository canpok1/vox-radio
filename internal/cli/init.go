package cli

import (
	_ "embed"
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

//go:embed templates/vox-radio.yaml
var configTemplate []byte

//go:embed templates/profile.yaml
var profileTemplate []byte

func newInitCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "init",
		Short: "カレントディレクトリにテンプレート設定ファイルを生成する",
		Long: `vox-radio.yaml（共通設定）と profile.yaml（プログラムプロファイル）を
カレントディレクトリに生成します。

既存ファイルは上書きを防ぐため個別にスキップされます。
両ファイルがすでに存在する場合は何も生成されません。

生成後は LLM API キー・番組内容・音声アセットパスを設定ファイルに記入し、
次のコマンドでパイプラインを実行してください:

  vox-radio run --profile profile.yaml`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := generateFile(cmd, "vox-radio.yaml", configTemplate); err != nil {
				return err
			}
			return generateFile(cmd, "profile.yaml", profileTemplate)
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
