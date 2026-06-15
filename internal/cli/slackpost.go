package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/canpok1/vox-radio/internal/config"
	"github.com/canpok1/vox-radio/internal/model"
	"github.com/canpok1/vox-radio/internal/slack"
	"github.com/spf13/cobra"
)

func newSlackpostCmd() *cobra.Command {
	var manifestPath string
	var specPath string
	var statePath string
	var dryRun bool

	cmd := &cobra.Command{
		Use:   "slackpost",
		Short: "manifest を入力に mp3 を Slack へ投稿する",
		Long: `manifest.json と slack-spec.yaml を入力に、mp3 ファイルを Slack へ投稿します。

mp3 ファイルは manifest と同じディレクトリの audio_file から自動解決します。
投稿は 2 段構成です: ①親メッセージ（mp3 + 初期コメント）、②スレッド返信（要約 + コーナー）。

Bot トークンは共通設定の slack.bot_token_env で指定した環境変数から取得します。
環境変数 VOX_RADIO_SLACK_API_URL を設定すると、Slack API の接続先 URL を上書きできます（テスト・検証用）。

実行進捗は状態ファイルに記録されます。タイムアウト後に再実行すると、音声の二重投稿なしに
未完了の返信投稿から再開します。状態ファイルの既定パスは manifest と同じディレクトリです。

例:
  vox-radio slackpost --manifest output/manifest.json --spec config/slack-spec.yaml
  vox-radio slackpost --manifest output/manifest.json --spec config/slack-spec.yaml --dry-run
  vox-radio slackpost --manifest output/manifest.json --spec config/slack-spec.yaml --state /tmp/state.json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.LoadConfig(configPath(cmd))
			if err != nil {
				return fmt.Errorf("load config: %w", err)
			}

			token := os.Getenv(cfg.Slack.BotTokenEnv)
			if token == "" && !dryRun {
				return fmt.Errorf("bot token env var %q is not set", cfg.Slack.BotTokenEnv)
			}

			manifest, err := readJSON[model.Manifest](manifestPath)
			if err != nil {
				return fmt.Errorf("load manifest: %w", err)
			}

			audioPath := filepath.Join(filepath.Dir(manifestPath), manifest.AudioFile)

			spec, err := slack.LoadSlackSpec(specPath)
			if err != nil {
				return err
			}
			if err := slack.ValidateSlackSpec(spec); err != nil {
				return err
			}

			if statePath == "" {
				statePath = slack.DefaultStatePath(manifestPath)
			}

			return slack.Run(slack.Options{
				Manifest:  manifest,
				AudioPath: audioPath,
				Spec:      spec,
				Token:     token,
				APIURL:    cfg.Slack.EffectiveAPIURL(),
				StatePath: statePath,
				DryRun:    dryRun,
				Out:       cmd.OutOrStdout(),
			}, nil)
		},
	}

	cmd.Flags().StringVar(&manifestPath, "manifest", "", "manifest.json ファイルのパス（必須）")
	cmd.Flags().StringVar(&specPath, "spec", "", "slack-spec.yaml ファイルのパス（必須）")
	cmd.Flags().StringVar(&statePath, "state", "", "状態ファイルのパス（省略時は manifest と同じディレクトリ）")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "API 非呼び出しで出力内容を確認する")
	_ = cmd.MarkFlagRequired("manifest")
	_ = cmd.MarkFlagRequired("spec")

	cmd.AddCommand(newSlackpostCheckCmd())

	return cmd
}

func newSlackpostCheckCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "check <path>",
		Short: "slack-spec.yaml を strict モードでフル検証する",
		Long: `指定した slack-spec.yaml を strict モードでパースし、以下を検証します:

  (a) strict パース: 未知キー（typo）をエラー化
  (b) 必須フィールド: slack.channel の存在チェック
  (c) テンプレートファイル: slack.message.{parent,thread,fallback} 指定時にファイルの存在・読み込み・構文を検証

成功時は標準出力に OK メッセージを出力し、ゼロで終了します。
失敗時は非ゼロで終了します（CI での自動検知に使用できます）。`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			path := args[0]

			spec, err := slack.LoadSlackSpecStrict(path)
			if err != nil {
				return err
			}

			if err := slack.ValidateSlackSpec(spec); err != nil {
				return err
			}

			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "OK: %s\n", path)
			return nil
		},
	}
}
