package cli

import (
	"fmt"
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
投稿先チャンネル ID は slack-spec.yaml の slack.channel_env で指定した環境変数から取得します。
環境変数 VOX_RADIO_SLACK_API_URL を設定すると、Slack API の接続先 URL を上書きできます（テスト・検証用）。

アップロード前に必要なスコープ（files:write / files:read / chat:write）を検証し、
不足していれば投稿せずに不足スコープを表示して終了します（音声の二重投稿を防ぎます）。

実行進捗は状態ファイルに記録されます。タイムアウト後に再実行すると、音声の二重投稿なしに
未完了の返信投稿から再開します。状態ファイルの既定パスは manifest と同じディレクトリです。
実行ログは --log-dir で指定したディレクトリに出力されます。

例:
  vox-radio slackpost --manifest output/manifest.json --spec config/slack-spec.yaml
  vox-radio slackpost --manifest output/manifest.json --spec config/slack-spec.yaml --dry-run
  vox-radio slackpost --manifest output/manifest.json --spec config/slack-spec.yaml --state /tmp/state.json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			logger, logFile, err := setupLogger("slackpost", logDirFlag(cmd))
			if err != nil {
				return fmt.Errorf("setup logger: %w", err)
			}
			defer func() { _ = logFile.Close() }()

			cfg, err := config.LoadConfig(configPath(cmd))
			if err != nil {
				return fmt.Errorf("load config: %w", err)
			}

			token, err := requireEnv(cfg.Slack.BotTokenEnv, dryRun)
			if err != nil {
				return fmt.Errorf("bot token: %w", err)
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

			channel, err := requireEnv(spec.Slack.ChannelEnv, dryRun)
			if err != nil {
				return fmt.Errorf("channel: %w", err)
			}

			if statePath == "" {
				statePath = slack.DefaultStatePath(manifestPath)
			}

			return slack.Run(slack.Options{
				Manifest:  manifest,
				AudioPath: audioPath,
				Spec:      spec,
				Token:     token,
				Channel:   channel,
				APIURL:    cfg.Slack.EffectiveAPIURL(),
				StatePath: statePath,
				DryRun:    dryRun,
				Out:       cmd.OutOrStdout(),
				Logger:    logger,
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
  (b) 必須フィールド: slack.channel_env の存在チェック
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
