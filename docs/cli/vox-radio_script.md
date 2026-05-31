## vox-radio script

LLM を使って収集した記事から台本を生成する

### Synopsis

多段階 LLM パイプライン（summarize → write → direct）を実行し、
articles.json から script.json を生成します。

vox-radio.yaml はカレントディレクトリから自動読み込みされます。
コーナー定義はプロファイルから取得します（plan ステップはありません）。

--step で単一ステージのみ実行できます:
  summarize  コーナーごとに各記事を要約します（summaries.json を出力）
  write      コーナーごとに台詞を書きます（lines.json を出力）
  direct     台詞に SE・話者を割り当てます（script.json を出力）

例:
  vox-radio script --in work/articles.json --out work/script.json
  vox-radio script --out work/script.json --step write
  vox-radio script --in work/articles.json --out work/script.json --profile sample-profiles/tech_profile.yaml

```
vox-radio script [flags]
```

### Options

```
  -h, --help             help for script
      --in string        articles.json の入力パス（フルパイプラインまたは summarize ステップで必須）
      --out string       script.json の出力先パス（必須）
      --profile string   プロファイル YAML ファイルのパス（必須）
      --prompts string   プロンプトテンプレートを含むディレクトリ (default "prompts")
      --step string      単一ステップを実行する: summarize|write|direct
```

### SEE ALSO

* [vox-radio](vox-radio.md)	 - AI を使ったポッドキャスト制作ツール

