## vox-radio slackpost check

slack-spec.yaml を strict モードでフル検証する

### Synopsis

指定した slack-spec.yaml を strict モードでパースし、以下を検証します:

  (a) strict パース: 未知キー（typo）をエラー化
  (b) 必須フィールド: slack.channel の存在チェック

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
      --config string   共通設定 YAML ファイル（vox-radio.yaml）のパス (default "vox-radio.yaml")
```

### SEE ALSO

* [vox-radio slackpost](vox-radio_slackpost.md)	 - manifest を入力に mp3 を Slack へ投稿する

