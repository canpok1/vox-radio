## vox-radio episodegen check

エピソード仕様ファイルを strict モードでフル検証する

### Synopsis

指定したエピソード仕様ファイルを strict モードでパースし、以下を検証します:

  (a) strict パース: 未知キー（typo）をエラー化
  (b) アセット参照: corners[].start_jingle / end_jingle / bgm が assets に存在するか
  (c) キャラ参照: corners[].cast のキャラ ID がカレントディレクトリの vox-radio.yaml に存在するか

成功時は標準出力に OK メッセージを出力し、ゼロで終了します。
失敗時は非ゼロで終了します（CI での自動検知に使用できます）。

```
vox-radio episodegen check <path> [flags]
```

### Options

```
  -h, --help   help for check
```

### SEE ALSO

* [vox-radio episodegen](vox-radio_episodegen.md)	 - ポッドキャスト制作パイプラインをすべて実行する

