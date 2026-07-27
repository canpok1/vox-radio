## vox-radio retro

過去回の分析から課題と施策を抽出し try ファイルを更新する（自動改善ループ）

### Synopsis

蓄積された過去回の分析（analyze の出力）から、反復して現れる課題を見つけ、
次に試す施策と組にして .vox-radio/programs/{program.id}/try.yaml へ記録します。

try ファイルの内容は episodegen 実行時に write（台本生成）プロンプトへ自動的に注入されます。
retro を実行しなくても、既存の try ファイルは注入され続けます。適用を止めたいときは
try ファイルを削除してください。

retro は毎回 try ファイルを全置換します（個別項目の承認は行いません）。恒久的に固定したい
方針は episode-spec.yaml の script_note に書いてください。

共通設定ファイルのパスは --config フラグで指定します（省略時は vox-radio.yaml）。

例:
  vox-radio retro --spec episode-spec.yaml
  vox-radio retro --spec episode-spec.yaml --dry-run

```
vox-radio retro [flags]
```

### Options

```
      --dry-run       try ファイルを書き込まず標準出力に出力する
  -h, --help          help for retro
      --spec string   エピソード仕様 YAML ファイルのパス（必須）
```

### Options inherited from parent commands

```
      --config string     共通設定 YAML ファイル（vox-radio.yaml）のパス (default "vox-radio.yaml")
      --env-file string   環境変数を読み込む env ファイルのパス（未指定時は .env を自動読込、不在は無視） (default ".env")
      --log-dir string    ログ出力ディレクトリのパス (default ".vox-radio/logs")
```

### SEE ALSO

* [vox-radio](vox-radio.md)	 - AI を使ったポッドキャスト制作ツール

