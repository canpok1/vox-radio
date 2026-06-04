## vox-radio episodegen manifest

エピソードのコンテンツマニフェスト JSON を生成する

### Synopsis

エピソードの内容を記述する manifest.json を生成します。
タイトル・説明・要約・日時・音声ファイル名・各コーナーの記事情報・会話メモを含みます。

マニフェストは別の配信サービスが RSS フィードを生成する際に使用することを想定しており、
フルパイプラインを再実行せずに済みます。

--lines を指定すると、共通設定ファイルの LLM 設定を使って
LLM が 03_lines.json（元表記のセリフ）から番組要約・会話メモ・コーナー単位要約を生成してマニフェストに追加します。
共通設定ファイルのパスは --config フラグで指定します（省略時は vox-radio.yaml）。
会話メモはキャラの近況・掛け合い・感想・ハプニング・継続ネタなど
rundown（記事の事実）に含まれない会話情報を幅広く記録します。

例:
  vox-radio episodegen manifest --spec examples/tech.yaml --audio output/episode.mp3 --out output/manifest.json
  vox-radio episodegen manifest --spec examples/tech.yaml --rundown output/intermediate/02_rundown.json --audio output/episode.mp3 --out output/manifest.json
  vox-radio episodegen manifest --spec examples/tech.yaml --lines output/intermediate/03_lines.json --audio output/episode.mp3 --out output/manifest.json

```
vox-radio episodegen manifest [flags]
```

### Options

```
      --audio string     音声ファイルのパス。ファイル名のみマニフェストに記録される（必須）
  -h, --help             help for manifest
      --lines string     03_lines.json のパス（任意）。指定すると LLM が元表記のセリフから番組要約・会話メモ・コーナー単位要約を生成する
      --out string       manifest.json の出力先パス（必須）
      --rundown string   02_rundown.json のパス（任意）。省略するとコーナーの記事は空になる
      --spec string      エピソード仕様 YAML ファイルのパス（必須）
```

### Options inherited from parent commands

```
      --config string   共通設定 YAML ファイル（vox-radio.yaml）のパス (default "vox-radio.yaml")
```

### SEE ALSO

* [vox-radio episodegen](vox-radio_episodegen.md)	 - ポッドキャスト制作パイプラインをすべて実行する

