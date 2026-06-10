## vox-radio slackpost

manifest を入力に mp3 を Slack へ投稿する

### Synopsis

manifest.json と slack-spec.yaml を入力に、mp3 ファイルを Slack へ投稿します。

mp3 ファイルは manifest と同じディレクトリの audio_file から自動解決します。
投稿は 2 段構成です: ①親メッセージ（mp3 + 初期コメント）、②スレッド返信（要約 + コーナー）。

Bot トークンは共通設定の slack.bot_token_env で指定した環境変数から取得します。
環境変数 VOX_RADIO_SLACK_API_URL を設定すると、Slack API の接続先 URL を上書きできます（テスト・検証用）。

実行進捗は状態ファイルに記録されます。タイムアウト後に再実行すると、音声の二重投稿なしに
未完了の返信投稿から再開します。状態ファイルの既定パスは manifest と同じディレクトリです。

例:
  vox-radio slackpost --manifest output/manifest.json --spec config/slack-spec.yaml
  vox-radio slackpost --manifest output/manifest.json --spec config/slack-spec.yaml --dry-run
  vox-radio slackpost --manifest output/manifest.json --spec config/slack-spec.yaml --state /tmp/state.json

```
vox-radio slackpost [flags]
```

### Options

```
      --dry-run           API 非呼び出しで出力内容を確認する
  -h, --help              help for slackpost
      --manifest string   manifest.json ファイルのパス（必須）
      --spec string       slack-spec.yaml ファイルのパス（必須）
      --state string      状態ファイルのパス（省略時は manifest と同じディレクトリ）
```

### Options inherited from parent commands

```
      --config string    共通設定 YAML ファイル（vox-radio.yaml）のパス (default "vox-radio.yaml")
      --log-dir string   ログ出力ディレクトリのパス (default ".vox-radio/logs")
```

### SEE ALSO

* [vox-radio](vox-radio.md)	 - AI を使ったポッドキャスト制作ツール
* [vox-radio slackpost check](vox-radio_slackpost_check.md)	 - slack-spec.yaml を strict モードでフル検証する

