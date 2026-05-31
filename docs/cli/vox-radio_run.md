## vox-radio run

ポッドキャスト制作パイプラインをすべて実行する

### Synopsis

collect → script → synth → assemble → manifest を一括実行します。

中間ファイルは <out-dir>/intermediate/ に書き出され、
最終的な episode.mp3 は <out-dir>/ 直下に配置されます。

vox-radio.yaml はカレントディレクトリから自動読み込みされます。

例:
  vox-radio run
  vox-radio run --out-dir output --profile sample-profiles/tech_profile.yaml

```
vox-radio run [flags]
```

### Options

```
  -h, --help             help for run
      --out-dir string   出力ディレクトリ（episode.mp3 をここに配置し、中間ファイルは <out-dir>/intermediate/ に配置） (default "output")
      --profile string   プロファイル YAML ファイルのパス（必須）
      --prompts string   プロンプトテンプレートを含むディレクトリ (default "prompts")
```

### SEE ALSO

* [vox-radio](vox-radio.md)	 - AI を使ったポッドキャスト制作ツール

