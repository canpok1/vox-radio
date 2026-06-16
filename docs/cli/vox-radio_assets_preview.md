## vox-radio assets preview

素材ID単体にパラメータを適用した音声をプレビュー生成する

### Synopsis

指定した assets.yaml から素材IDを検索し、パラメータを適用したプレビュー音声を MP3 で生成します。

loudnorm/alimiter は適用されないため、各パラメータの素の効果を確認できます。

デフォルトでは末尾の打ち切りを行わず素材の全長を出力します（30秒を超える BGM 全体を確認できます）。
--max-length-sec に正の秒数を指定したときのみ、その長さで末尾を打ち切ります。
loop=true の BGM は、--max-length-sec 未指定時はループせず素材を1回分出力します。

例:
  vox-radio assets preview assets.yaml --id jingle:opening --out preview.mp3
  vox-radio assets preview assets.yaml --id bgm:talk --out preview.mp3 --max-length-sec 15

```
vox-radio assets preview <path> [flags]
```

### Options

```
  -h, --help                   help for preview
      --id string              {type}:{key} 形式の素材ID（type: jingle/se/bgm）（必須）
      --max-length-sec float   プレビュー出力の最大長（秒）。未指定（0以下）なら打ち切らず全長を出力し、正の値を指定したときのみその長さで末尾を打ち切る
      --out string             MP3 出力先パス（必須）
```

### Options inherited from parent commands

```
      --config string    共通設定 YAML ファイル（vox-radio.yaml）のパス (default "vox-radio.yaml")
      --log-dir string   ログ出力ディレクトリのパス (default ".vox-radio/logs")
```

### SEE ALSO

* [vox-radio assets](vox-radio_assets.md)	 - アセット設定ファイルを管理するコマンド群

