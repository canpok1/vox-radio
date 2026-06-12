package cli

import (
	"fmt"

	"github.com/canpok1/vox-radio/internal/config"
	"github.com/spf13/cobra"
)

func newEpisodegenCheckCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "check <path>",
		Short: "エピソード仕様ファイルを strict モードでフル検証する",
		Long: `指定したエピソード仕様ファイルを strict モードでパースし、以下を検証します:

  (a) strict パース: 未知キー（typo）をエラー化
  (b) program.id が設定されているか（キャッシュキーのため必須）
  (c) アセット参照: corners[].start_audio / end_audio の type+id と bgm が assets に存在するか
  (d) corners[].cast のキーが casts に宣言済みであるか
  (e) casts のキャラ ID が共通設定ファイルの characters に存在するか、type/condition が正しいか

共通設定ファイルのパスは --config フラグで指定します（省略時は vox-radio.yaml）。

成功時は標準出力に OK メッセージを出力し、ゼロで終了します。
失敗時は非ゼロで終了します（CI での自動検知に使用できます）。`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			path := args[0]

			p, err := config.LoadEpisodeSpecStrict(path)
			if err != nil {
				return err
			}

			cfg, err := config.LoadConfig(configPath(cmd))
			if err != nil {
				return fmt.Errorf("load config for cast validation: %w", err)
			}

			if err := p.Validate(cfg.Characters); err != nil {
				return err
			}

			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "OK: %s\n", path)
			return nil
		},
	}
}
