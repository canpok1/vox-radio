## vox-radio episodegen synth

台本から音声クリップを合成する

### Synopsis

script.json を読み込み、VOICEVOX を呼び出して各台詞を WAV クリップに合成します。
出力ディレクトリには台詞ごとの WAV ファイルと clips.json マニフェストが格納されます。

共通設定ファイルのパスは --config フラグで指定します（省略時は vox-radio.yaml）。
voicevox.url フィールドで VOICEVOX エンジンの URL を指定します（デフォルト: http://localhost:50021）。
環境変数 VOX_RADIO_VOICEVOX_URL を設定すると、設定ファイルの値より優先して URL を上書きできます。
話者 ID は共通設定ファイルのキャラクターカタログから解決されます。

例:
  vox-radio episodegen synth --in work/script.json --out-dir work/clips

```
vox-radio episodegen synth [flags]
```

### Options

```
  -h, --help             help for synth
      --in string        script.json の入力パス（必須）
      --out-dir string   WAV クリップの出力ディレクトリ（必須）
```

### Options inherited from parent commands

```
      --config string     共通設定 YAML ファイル（vox-radio.yaml）のパス (default "vox-radio.yaml")
      --env-file string   環境変数を読み込む env ファイルのパス（未指定時は .env を自動読込、不在は無視） (default ".env")
      --log-dir string    ログ出力ディレクトリのパス (default ".vox-radio/logs")
```

### SEE ALSO

* [vox-radio episodegen](vox-radio_episodegen.md)	 - ポッドキャスト制作パイプラインをすべて実行する

