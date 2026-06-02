## vox-radio feedgen

キャッシュから RSS フィード（feed.xml）を生成する

### Synopsis

cache ファイルと distribution.yaml から RSS 2.0 + iTunes フィード（feed.xml）を生成します。

cache はエピソード状態の正データです。manifest や mp3 は必要ありません。
生成された feed.xml は distribution.yaml の output.public ディレクトリに書き出されます。

例:
  vox-radio feedgen --cache .vox-radio/cache/zundamon-tech-radio.jsonl --config config/distribution.yaml

```
vox-radio feedgen [flags]
```

### Options

```
      --cache string    キャッシュ JSONL ファイルのパス（必須）
      --config string   distribution.yaml ファイルのパス（必須）
  -h, --help            help for feedgen
```

### SEE ALSO

* [vox-radio](vox-radio.md)	 - AI を使ったポッドキャスト制作ツール

