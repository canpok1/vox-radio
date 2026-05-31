## vox-radio rundown

収集記事から番組設計図（rundown）を生成する

### Synopsis

LLM を使って収集記事を選別し、コーナーごとの話の流れと要約を含む
02_rundown.json を生成します。

vox-radio.yaml はカレントディレクトリから自動読み込みされます。
コーナー定義はプロファイルから取得します。

例:
  vox-radio rundown --in work/intermediate/01_articles.json --out work/intermediate/02_rundown.json --profile sample-profiles/tech_profile.yaml

```
vox-radio rundown [flags]
```

### Options

```
  -h, --help             help for rundown
      --in string        01_articles.json の入力パス（必須）
      --out string       02_rundown.json の出力先パス（必須）
      --profile string   プロファイル YAML ファイルのパス（必須）
      --prompts string   プロンプトテンプレートを含むディレクトリ (default "prompts")
```

### SEE ALSO

* [vox-radio](vox-radio.md)	 - AI を使ったポッドキャスト制作ツール

