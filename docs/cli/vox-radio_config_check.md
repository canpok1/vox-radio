## vox-radio config check

設定ファイル（vox-radio.yaml）を strict モードで検証する

### Synopsis

共通設定ファイルを strict モードでパースし、
未知のキー（typo）や設定値の不整合をエラーとして報告します。
設定ファイルのパスは --config フラグで指定します（省略時は vox-radio.yaml）。

成功時は標準出力に OK メッセージを出力し、ゼロで終了します。
失敗時は非ゼロで終了します（CI での自動検知に使用できます）。

```
vox-radio config check [flags]
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

* [vox-radio config](vox-radio_config.md)	 - 設定ファイル（vox-radio.yaml）を操作するサブコマンド群

