## vox-radio assets

アセット設定ファイルを管理するコマンド群

### Synopsis

アセット設定ファイル（assets.yaml）の管理操作を提供します。

サブコマンド:
  check    アセット設定ファイルを strict モードで検証する
  preview  素材ID単体にパラメータを適用した音声をプレビュー生成する

### Options

```
  -h, --help   help for assets
```

### Options inherited from parent commands

```
      --config string     共通設定 YAML ファイル（vox-radio.yaml）のパス (default "vox-radio.yaml")
      --env-file string   環境変数を読み込む env ファイルのパス（未指定時は .env を自動読込、不在は無視） (default ".env")
      --log-dir string    ログ出力ディレクトリのパス (default ".vox-radio/logs")
```

### SEE ALSO

* [vox-radio](vox-radio.md)	 - AI を使ったポッドキャスト制作ツール
* [vox-radio assets check](vox-radio_assets_check.md)	 - アセット設定ファイルを strict モードでフル検証する
* [vox-radio assets preview](vox-radio_assets_preview.md)	 - 素材ID単体にパラメータを適用した音声をプレビュー生成する

