## vox-radio render

manifest を text/template でレンダリングして出力する

### Synopsis

manifest.json と text/template を入力に、レンダリング結果を標準出力（または --output ファイル）へ書き出します。

テンプレートはファイル（--template）またはインライン文字列（--template-string）で指定します。両方の同時指定は不可。

よく使うトップレベルフィールド:
  .Title          — 番組タイトル
  .EpisodeNumber  — 回番号（int）
  .EpisodeTitle   — サブタイトル
  .AudioFile      — 音声ファイル名
  .Summary        — 全体要約
  .Datetime       — 配信日時
  .Author         — 著者

テンプレート関数:
  corner "<id>"     — 指定 ID のコーナーを返す（見つからない場合は nil）
  hasLinks <corner> — コーナーに URL 付き記事が 1 件以上あれば true

全フィールド・コーナー・関数の一覧:
  https://github.com/canpok1/vox-radio/blob/main/internal/cli/skills/vox-radio/references/manifest.md

例（ファイル指定）:
  vox-radio render --manifest output/manifest.json --template release-note.tmpl

例（インライン指定・CI での値抽出）:
  vox-radio render --manifest output/manifest.json --template-string '{{.EpisodeNumber}}'
  vox-radio render --manifest output/manifest.json --template-string '第{{.EpisodeNumber}}回 {{.EpisodeTitle}}'

```
vox-radio render [flags]
```

### Options

```
  -h, --help                     help for render
      --manifest string          manifest.json ファイルのパス（必須）
      --output string            出力先ファイルのパス（省略時は標準出力）
      --template string          text/template ファイルのパス（--template-string と排他）
      --template-string string   テンプレート文字列（--template と排他、CI での値抽出に便利）
```

### Options inherited from parent commands

```
      --config string     共通設定 YAML ファイル（vox-radio.yaml）のパス (default "vox-radio.yaml")
      --env-file string   環境変数を読み込む env ファイルのパス（未指定時は .env を自動読込、不在は無視） (default ".env")
      --log-dir string    ログ出力ディレクトリのパス (default ".vox-radio/logs")
```

### SEE ALSO

* [vox-radio](vox-radio.md)	 - AI を使ったポッドキャスト制作ツール

