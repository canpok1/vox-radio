## vox-radio init

カレントディレクトリにテンプレート設定ファイルを生成する

### Synopsis

vox-radio.yaml（共通設定）・episode-spec.yaml（エピソード仕様）・feed-spec.yaml（フィード生成設定）を
カレントディレクトリに生成します。

既存ファイルは上書きを防ぐため個別にスキップされます。
すべてのファイルがすでに存在する場合は何も生成されません。

生成後は LLM API キー・番組内容・音声アセットパスを設定ファイルに記入し、
次のコマンドでパイプラインを実行してください:

  vox-radio episodegen --spec episode-spec.yaml

```
vox-radio init [flags]
```

### Options

```
  -h, --help   help for init
```

### Options inherited from parent commands

```
      --config string   共通設定 YAML ファイル（vox-radio.yaml）のパス (default "vox-radio.yaml")
```

### SEE ALSO

* [vox-radio](vox-radio.md)	 - AI を使ったポッドキャスト制作ツール

