## vox-radio render

manifest を text/template でレンダリングして出力する

### Synopsis

manifest.json と text/template ファイルを入力に、レンダリング結果を標準出力（または --output ファイル）へ書き出します。

テンプレートのデータ文脈は manifest 全体です。以下のテンプレート関数が使えます:
  corner "<id>"  — 指定 ID のコーナーを返す（見つからない場合は nil）
  hasLinks <corner> — コーナーに URL 付き記事が 1 件以上あれば true

URL なし記事のスキップは {{if .URL}} でテンプレ側に表現できます。

例:
  vox-radio render --manifest output/manifest.json --template release-note.tmpl
  vox-radio render --manifest output/manifest.json --template release-note.tmpl --output RELEASE_NOTES.md

```
vox-radio render [flags]
```

### Options

```
  -h, --help              help for render
      --manifest string   manifest.json ファイルのパス（必須）
      --output string     出力先ファイルのパス（省略時は標準出力）
      --template string   text/template ファイルのパス（必須）
```

### Options inherited from parent commands

```
      --config string    共通設定 YAML ファイル（vox-radio.yaml）のパス (default "vox-radio.yaml")
      --log-dir string   ログ出力ディレクトリのパス (default ".vox-radio/logs")
```

### SEE ALSO

* [vox-radio](vox-radio.md)	 - AI を使ったポッドキャスト制作ツール

