package cli

import (
	"fmt"

	"github.com/canpok1/vox-radio/internal/config"
	"github.com/spf13/cobra"
)

func newAssetsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "assets",
		Short: "アセット設定ファイルを管理するコマンド群",
		Long: `アセット設定ファイル（assets.yaml）の管理操作を提供します。

サブコマンド:
  check  アセット設定ファイルを strict モードで検証する`,
	}
	cmd.AddCommand(newAssetsCheckCmd())
	return cmd
}

func newAssetsCheckCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "check <path>",
		Short: "アセット設定ファイルを strict モードでフル検証する",
		Long: `指定したアセット設定ファイルを strict モードでパースし、以下を検証します:

  (a) strict パース: 未知キー（typo）をエラー化
  (b) 参照ファイルの実在確認: jingle/se/bgm の file フィールドが示すファイルの存在確認
  (c) 値の範囲検証: volume/fade_in/fade_out/duck_ratio の正当性確認

成功時は標準出力に OK メッセージを出力し、ゼロで終了します。
失敗時は非ゼロで終了します（CI での自動検知に使用できます）。`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			path := args[0]

			assets, err := config.LoadAssetsFileStrict(path)
			if err != nil {
				return err
			}

			if err := config.ValidateAssetsConfig(&assets); err != nil {
				return err
			}

			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "OK: %s\n", path)
			return nil
		},
	}
}
