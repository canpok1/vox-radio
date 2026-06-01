## vox-radio script

LLM を使って rundown から台本を生成する

### Synopsis

多段階 LLM パイプライン（write → direct）を実行し、
02_rundown.json から 04_script.json を生成します。

vox-radio.yaml はカレントディレクトリから自動読み込みされます。
コーナー定義はプロファイルから取得します。

--step で単一ステージのみ実行できます:
  write      コーナーごとに台詞を書きます（03_lines.json を出力）
  direct     台詞に SE・話者を割り当てます（04_script.json を出力）

例:
  vox-radio script --in work/intermediate/02_rundown.json --out work/intermediate/04_script.json
  vox-radio script --out work/intermediate/04_script.json --step write
  vox-radio script --in work/intermediate/02_rundown.json --out work/intermediate/04_script.json --profile sample-profiles/tech_profile.yaml

```
vox-radio script [flags]
```

### Options

```
  -h, --help             help for script
      --in string        02_rundown.json の入力パス（フルパイプラインまたは write ステップで必須）
      --out string       04_script.json の出力先パス（必須）
      --profile string   プロファイル YAML ファイルのパス（必須）
      --step string      単一ステップを実行する: write|direct
```

### SEE ALSO

* [vox-radio](vox-radio.md)	 - AI を使ったポッドキャスト制作ツール

