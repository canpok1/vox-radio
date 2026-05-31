## vox-radio collect

コーナーごとに RSS/Atom フィードと URL から記事を収集する

### Synopsis

corners[].source に定義された RSS/Atom フィードや Web URL から記事を収集し、
本文テキストを抽出して articles.json に書き出します。

source フィールドのないコーナーはスキップされます。

例:
  vox-radio collect --out work/articles.json
  vox-radio collect --out work/articles.json --profile sample-profiles/tech_profile.yaml

```
vox-radio collect [flags]
```

### Options

```
  -h, --help             help for collect
      --out string       articles.json の出力先パス（必須）
      --profile string   プロファイル YAML ファイルのパス（必須）
```

### SEE ALSO

* [vox-radio](vox-radio.md)	 - AI を使ったポッドキャスト制作ツール

