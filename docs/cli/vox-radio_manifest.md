## vox-radio manifest

エピソードのコンテンツマニフェスト JSON を生成する

### Synopsis

エピソードの内容を記述する manifest.json を生成します。
タイトル・説明・要約・日時・音声ファイル名・各コーナーの記事情報を含みます。

マニフェストは別の配信サービスが RSS フィードを生成する際に使用することを想定しており、
フルパイプラインを再実行せずに済みます。

--script を指定すると、vox-radio.yaml の LLM 設定を使って
LLM が生成した要約をマニフェストに追加します。

例:
  vox-radio manifest --profile sample-profiles/tech_profile.yaml --audio output/episode.mp3 --out output/manifest.json
  vox-radio manifest --profile sample-profiles/tech_profile.yaml --articles output/intermediate/articles.json --audio output/episode.mp3 --out output/manifest.json
  vox-radio manifest --profile sample-profiles/tech_profile.yaml --script output/intermediate/script.json --audio output/episode.mp3 --out output/manifest.json

```
vox-radio manifest [flags]
```

### Options

```
      --articles string   articles.json のパス（任意）省略するとコーナーの記事は空になる
      --audio string      音声ファイルのパス。ファイル名のみマニフェストに記録される（必須）
  -h, --help              help for manifest
      --out string        manifest.json の出力先パス（必須）
      --profile string    プロファイル YAML ファイルのパス（必須）
      --prompts string    プロンプトテンプレートを含むディレクトリ（--script 指定時に使用） (default "prompts")
      --script string     script.json のパス（任意）指定すると LLM が台本から要約を生成する
```

### SEE ALSO

* [vox-radio](vox-radio.md)	 - AI を使ったポッドキャスト制作ツール

