package cli

import (
	"fmt"

	"github.com/canpok1/vox-radio/internal/config"
	"github.com/spf13/cobra"
)

func newProfileCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "profile",
		Short: "プロファイルファイルを操作するサブコマンド群",
		Long: `番組プロファイル YAML に関連するサブコマンドを提供します。

現在利用可能なサブコマンド:
  check  プロファイルファイルの内容を検証します`,
	}
	cmd.AddCommand(newProfileCheckCmd())
	return cmd
}

func newProfileCheckCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "check <path>",
		Short: "プロファイルファイルを strict モードでフル検証する",
		Long: `指定したプロファイルファイルを strict モードでパースし、以下を検証します:

  (a) strict パース: 未知キー（typo）をエラー化
  (b) アセット参照: corners[].start_jingle / end_jingle / bgm が assets に存在するか
  (c) キャラ参照: corners[].cast のキャラ ID がカレントディレクトリの vox-radio.yaml に存在するか

成功時は標準出力に OK メッセージを出力し、ゼロで終了します。
失敗時は非ゼロで終了します（CI での自動検知に使用できます）。`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			path := args[0]

			p, err := config.LoadProfileStrict(path)
			if err != nil {
				return err
			}

			if err := config.ValidateProfileAssets(p); err != nil {
				return err
			}

			cfg, err := config.LoadConfig("vox-radio.yaml")
			if err != nil {
				return fmt.Errorf("load vox-radio.yaml for cast validation: %w", err)
			}

			if err := config.ValidateProfileCast(p, cfg.Characters); err != nil {
				return err
			}

			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "OK: %s\n", path)
			return nil
		},
	}
}
