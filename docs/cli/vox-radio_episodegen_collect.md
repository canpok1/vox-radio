## vox-radio episodegen collect

コーナーごとに RSS/Atom フィードと URL から記事を収集する

### Synopsis

corners[].source に定義された RSS/Atom フィードや Web URL から記事を収集し、
本文テキストを抽出して articles.json に書き出します。

source フィールドのないコーナーはスキップされます。

例:
  vox-radio episodegen collect --out work/articles.json
  vox-radio episodegen collect --out work/articles.json --spec examples/tech.yaml

```
vox-radio episodegen collect [flags]
```

### Options

```
  -h, --help          help for collect
      --out string    articles.json の出力先パス（必須）
      --spec string   エピソード仕様 YAML ファイルのパス（必須）
```

### Options inherited from parent commands

```
      --config string   共通設定 YAML ファイル（vox-radio.yaml）のパス (default "vox-radio.yaml")
```

### SEE ALSO

* [vox-radio episodegen](vox-radio_episodegen.md)	 - ポッドキャスト制作パイプラインをすべて実行する

