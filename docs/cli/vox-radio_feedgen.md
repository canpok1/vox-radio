## vox-radio feedgen

キャッシュから RSS フィード（feed.xml）を生成する

### Synopsis

cache ファイルと feed-spec.yaml から RSS 2.0 + iTunes フィード（feed.xml）を生成します。

cache はエピソード状態の正データです。manifest や mp3 は必要ありません。
生成された feed.xml は feed-spec.yaml の output.public ディレクトリに書き出されます。

例:
  vox-radio feedgen --cache .vox-radio/cache/zundamon-tech-radio.jsonl --spec config/feed-spec.yaml

```
vox-radio feedgen [flags]
```

### Options

```
      --cache string   キャッシュ JSONL ファイルのパス（必須）
  -h, --help           help for feedgen
      --spec string    feed-spec.yaml ファイルのパス（必須）
```

### SEE ALSO

* [vox-radio](vox-radio.md)	 - AI を使ったポッドキャスト制作ツール

