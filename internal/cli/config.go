package cli

import (
	"fmt"

	"github.com/canpok1/vox-radio/internal/config"
	"github.com/spf13/cobra"
)

func newConfigCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "設定ファイル（vox-radio.yaml）を操作するサブコマンド群",
		Long: `vox-radio.yaml（共通設定）に関連するサブコマンドを提供します。
設定ファイルのパスは --config フラグで指定します（省略時は vox-radio.yaml）。

現在利用可能なサブコマンド:
  check  設定ファイルの内容を検証します`,
	}
	cmd.AddCommand(newConfigCheckCmd())
	return cmd
}

func newConfigCheckCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "check",
		Short: "設定ファイル（vox-radio.yaml）を strict モードで検証する",
		Long: `共通設定ファイルを strict モードでパースし、
未知のキー（typo）や設定値の不整合をエラーとして報告します。
設定ファイルのパスは --config フラグで指定します（省略時は vox-radio.yaml）。

成功時は標準出力に OK メッセージを出力し、ゼロで終了します。
失敗時は非ゼロで終了します（CI での自動検知に使用できます）。`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			path := configPath(cmd)
			if _, err := config.LoadConfigStrict(path); err != nil {
				return err
			}
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "OK: %s\n", path)
			return nil
		},
	}
}
