## vox-radio episodegen rundown

収集記事から番組設計図（rundown）を生成する

### Synopsis

LLM を使って収集記事を選別し、コーナーごとの話の流れと要約を含む
02_rundown.json を生成します。

vox-radio.yaml はカレントディレクトリから自動読み込みされます。
コーナー定義はエピソード仕様から取得します。

例:
  vox-radio episodegen rundown --in work/intermediate/01_articles.json --out work/intermediate/02_rundown.json --spec examples/tech.yaml

```
vox-radio episodegen rundown [flags]
```

### Options

```
  -h, --help          help for rundown
      --in string     01_articles.json の入力パス（必須）
      --out string    02_rundown.json の出力先パス（必須）
      --spec string   エピソード仕様 YAML ファイルのパス（必須）
```

### SEE ALSO

* [vox-radio episodegen](vox-radio_episodegen.md)	 - ポッドキャスト制作パイプラインをすべて実行する

