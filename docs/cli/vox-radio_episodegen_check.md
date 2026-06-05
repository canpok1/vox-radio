## vox-radio episodegen check

エピソード仕様ファイルを strict モードでフル検証する

### Synopsis

指定したエピソード仕様ファイルを strict モードでパースし、以下を検証します:

  (a) strict パース: 未知キー（typo）をエラー化
  (b) アセット参照: corners[].start_jingle / end_jingle / bgm が assets に存在するか
  (c) corners[].cast のキーが casts に宣言済みであるか
  (d) casts のキャラ ID が共通設定ファイルの characters に存在するか、type/condition が正しいか

共通設定ファイルのパスは --config フラグで指定します（省略時は vox-radio.yaml）。

成功時は標準出力に OK メッセージを出力し、ゼロで終了します。
失敗時は非ゼロで終了します（CI での自動検知に使用できます）。

```
vox-radio episodegen check <path> [flags]
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

* [vox-radio episodegen](vox-radio_episodegen.md)	 - ポッドキャスト制作パイプラインをすべて実行する

