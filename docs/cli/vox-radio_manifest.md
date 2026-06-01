## vox-radio manifest

エピソードのコンテンツマニフェスト JSON を生成する

### Synopsis

エピソードの内容を記述する manifest.json を生成します。
タイトル・説明・要約・日時・音声ファイル名・各コーナーの記事情報を含みます。

マニフェストは別の配信サービスが RSS フィードを生成する際に使用することを想定しており、
フルパイプラインを再実行せずに済みます。

--script を指定すると、vox-radio.yaml の LLM 設定を使って
LLM が生成した番組要約をマニフェストに追加します。

--lines を指定すると、vox-radio.yaml の LLM 設定を使って
LLM が各コーナーの台本からコーナー単位の要約を生成してマニフェストに追加します。

例:
  vox-radio manifest --profile sample-profiles/tech_profile.yaml --audio output/episode.mp3 --out output/manifest.json
  vox-radio manifest --profile sample-profiles/tech_profile.yaml --rundown output/intermediate/02_rundown.json --audio output/episode.mp3 --out output/manifest.json
  vox-radio manifest --profile sample-profiles/tech_profile.yaml --script output/intermediate/04_script.json --audio output/episode.mp3 --out output/manifest.json
  vox-radio manifest --profile sample-profiles/tech_profile.yaml --lines output/intermediate/03_lines.json --audio output/episode.mp3 --out output/manifest.json

```
vox-radio manifest [flags]
```

### Options

```
      --audio string     音声ファイルのパス。ファイル名のみマニフェストに記録される（必須）
  -h, --help             help for manifest
      --lines string     03_lines.json のパス（任意）。指定すると LLM がコーナー台本からコーナー単位要約を生成する
      --out string       manifest.json の出力先パス（必須）
      --profile string   プロファイル YAML ファイルのパス（必須）
      --prompts string   プロンプトテンプレートを含むディレクトリ（--script / --lines 指定時に使用） (default "prompts")
      --rundown string   02_rundown.json のパス（任意）。省略するとコーナーの記事は空になる
      --script string    04_script.json のパス（任意）。指定すると LLM が台本から番組要約を生成する
```

### SEE ALSO

* [vox-radio](vox-radio.md)	 - AI を使ったポッドキャスト制作ツール

