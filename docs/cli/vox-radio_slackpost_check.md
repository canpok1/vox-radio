## vox-radio slackpost check

slack-spec.yaml を strict モードでフル検証する

### Synopsis

指定した slack-spec.yaml を strict モードでパースし、以下を検証します:

  (a) strict パース: 未知キー（typo）をエラー化
  (b) 必須フィールド: slack.channel_env の存在チェック
  (c) テンプレートファイル: slack.message.{parent,thread,fallback} 指定時にファイルの存在・読み込み・構文を検証

成功時は標準出力に OK メッセージを出力し、ゼロで終了します。
失敗時は非ゼロで終了します（CI での自動検知に使用できます）。

```
vox-radio slackpost check <path> [flags]
```

### Options

```
  -h, --help   help for check
```

### Options inherited from parent commands

```
      --config string     共通設定 YAML ファイル（vox-radio.yaml）のパス (default "vox-radio.yaml")
      --env-file string   環境変数を読み込む env ファイルのパス（未指定時は .env を自動読込、不在は無視） (default ".env")
      --log-dir string    ログ出力ディレクトリのパス (default ".vox-radio/logs")
```

### SEE ALSO

* [vox-radio slackpost](vox-radio_slackpost.md)	 - manifest を入力に mp3 を Slack へ投稿する

