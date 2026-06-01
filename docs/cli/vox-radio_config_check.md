## vox-radio config check

設定ファイル（vox-radio.yaml）を strict モードで検証する

### Synopsis

指定した設定ファイル（デフォルト: vox-radio.yaml）を strict モードでパースし、
未知のキー（typo）や設定値の不整合をエラーとして報告します。

成功時は標準出力に OK メッセージを出力し、ゼロで終了します。
失敗時は非ゼロで終了します（CI での自動検知に使用できます）。

```
vox-radio config check [path] [flags]
```

### Options

```
  -h, --help   help for check
```

### SEE ALSO

* [vox-radio config](vox-radio_config.md)	 - 設定ファイル（vox-radio.yaml）を操作するサブコマンド群

