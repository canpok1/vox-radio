## vox-radio config

設定ファイル（vox-radio.yaml）を操作するサブコマンド群

### Synopsis

vox-radio.yaml（共通設定）に関連するサブコマンドを提供します。
設定ファイルのパスは --config フラグで指定します（省略時は vox-radio.yaml）。

現在利用可能なサブコマンド:
  check  設定ファイルの内容を検証します

### Options

```
  -h, --help   help for config
```

### Options inherited from parent commands

```
      --config string   共通設定 YAML ファイル（vox-radio.yaml）のパス (default "vox-radio.yaml")
```

### SEE ALSO

* [vox-radio](vox-radio.md)	 - AI を使ったポッドキャスト制作ツール
* [vox-radio config check](vox-radio_config_check.md)	 - 設定ファイル（vox-radio.yaml）を strict モードで検証する

