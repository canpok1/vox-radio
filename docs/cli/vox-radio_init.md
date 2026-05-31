## vox-radio init

カレントディレクトリにテンプレート設定ファイルを生成する

### Synopsis

vox-radio.yaml（共通設定）と profile.yaml（プログラムプロファイル）を
カレントディレクトリに生成します。

既存ファイルは上書きを防ぐため個別にスキップされます。
両ファイルがすでに存在する場合は何も生成されません。

生成後は LLM API キー・番組内容・音声アセットパスを設定ファイルに記入し、
次のコマンドでパイプラインを実行してください:

  vox-radio run --profile profile.yaml

```
vox-radio init [flags]
```

### Options

```
  -h, --help   help for init
```

### SEE ALSO

* [vox-radio](vox-radio.md)	 - AI を使ったポッドキャスト制作ツール

