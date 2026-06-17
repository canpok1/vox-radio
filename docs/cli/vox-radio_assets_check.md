## vox-radio assets check

アセット設定ファイルを strict モードでフル検証する

### Synopsis

指定したアセット設定ファイルを strict モードでパースし、以下を検証します:

  (a) strict パース: 未知キー（typo）をエラー化
  (b) 参照ファイルの実在確認: jingle/se/bgm の file フィールドが示すファイルの存在確認
  (c) 値の範囲検証: volume/fade_in/fade_out/duck_ratio の正当性確認

成功時は標準出力に OK メッセージを出力し、ゼロで終了します。
失敗時は非ゼロで終了します（CI での自動検知に使用できます）。

```
vox-radio assets check <path> [flags]
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

* [vox-radio assets](vox-radio_assets.md)	 - アセット設定ファイルを管理するコマンド群

