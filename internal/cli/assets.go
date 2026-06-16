package cli

import (
	"context"
	"fmt"
	"strings"

	"github.com/canpok1/vox-radio/internal/assemble"
	"github.com/canpok1/vox-radio/internal/config"
	"github.com/spf13/cobra"
)

func newAssetsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "assets",
		Short: "アセット設定ファイルを管理するコマンド群",
		Long: `アセット設定ファイル（assets.yaml）の管理操作を提供します。

サブコマンド:
  check    アセット設定ファイルを strict モードで検証する
  preview  素材ID単体にパラメータを適用した音声をプレビュー生成する`,
	}
	cmd.AddCommand(newAssetsCheckCmd())
	cmd.AddCommand(newAssetsPreviewCmd())
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

func newAssetsPreviewCmd() *cobra.Command {
	var id string
	var outPath string
	var maxLengthSec float64

	cmd := &cobra.Command{
		Use:   "preview <path>",
		Short: "素材ID単体にパラメータを適用した音声をプレビュー生成する",
		Long: `指定した assets.yaml から素材IDを検索し、パラメータを適用したプレビュー音声を MP3 で生成します。

loudnorm/alimiter は適用されないため、各パラメータの素の効果を確認できます。

デフォルトでは末尾の打ち切りを行わず素材の全長を出力します（30秒を超える BGM 全体を確認できます）。
--max-length-sec に正の秒数を指定したときのみ、その長さで末尾を打ち切ります。
loop=true の BGM は、--max-length-sec 未指定時はループせず素材を1回分出力します。

例:
  vox-radio assets preview assets.yaml --id jingle:opening --out preview.mp3
  vox-radio assets preview assets.yaml --id bgm:talk --out preview.mp3 --max-length-sec 15`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			assetsPath := args[0]

			parts := strings.SplitN(id, ":", 2)
			if len(parts) != 2 {
				return fmt.Errorf("--id must be in the format type:key (e.g. jingle:opening), got %q", id)
			}
			assetType, assetKey := parts[0], parts[1]

			assets, err := config.LoadAssetsFileStrict(assetsPath)
			if err != nil {
				return err
			}

			pctx := assemble.PreviewContext{
				AssetType:    assetType,
				AssetKey:     assetKey,
				Assets:       assets,
				OutPath:      outPath,
				MaxLengthSec: maxLengthSec,
			}

			p := assemble.NewPreviewer()
			return p.Run(context.Background(), pctx, cmd.ErrOrStderr())
		},
	}

	cmd.Flags().StringVar(&id, "id", "", "{type}:{key} 形式の素材ID（type: jingle/se/bgm）（必須）")
	cmd.Flags().StringVar(&outPath, "out", "", "MP3 出力先パス（必須）")
	cmd.Flags().Float64Var(&maxLengthSec, "max-length-sec", 0, "プレビュー出力の最大長（秒）。未指定（0以下）なら打ち切らず全長を出力し、正の値を指定したときのみその長さで末尾を打ち切る")
	_ = cmd.MarkFlagRequired("id")
	_ = cmd.MarkFlagRequired("out")

	return cmd
}
