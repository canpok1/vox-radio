## vox-radio feedgen check

feed-spec.yaml を strict モードでフル検証する

### Synopsis

指定した feed-spec.yaml を strict モードでパースし、以下を検証します:

  (a) strict パース: 未知キー（typo）をエラー化
  (b) 必須フィールド: feed.language / feed.author / feed.email /
      feed.site_url / feed.audio_url_template の存在チェック
  (c) URL / email 形式: 各フィールドの値が正しい形式かチェック
  (d) プレースホルダ: audio_url_template に {episode_number} と {audio_file} が含まれるかチェック

意味検証エラー (b)(c)(d) は全件収集してまとめて報告します。

成功時は標準出力に OK メッセージを出力し、ゼロで終了します。
失敗時は非ゼロで終了します（CI での自動検知に使用できます）。

```
vox-radio feedgen check <path> [flags]
```

### Options

```
  -h, --help   help for check
```

### Options inherited from parent commands

```
      --config string    共通設定 YAML ファイル（vox-radio.yaml）のパス (default "vox-radio.yaml")
      --log-dir string   ログ出力ディレクトリのパス (default ".vox-radio/logs")
```

### SEE ALSO

* [vox-radio feedgen](vox-radio_feedgen.md)	 - キャッシュから RSS フィード（feed.xml）を生成する

