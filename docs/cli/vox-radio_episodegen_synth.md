## vox-radio episodegen synth

台本から音声クリップを合成する

### Synopsis

script.json を読み込み、VOICEVOX を呼び出して各台詞を WAV クリップに合成します。
出力ディレクトリには台詞ごとの WAV ファイルと clips.json マニフェストが格納されます。

vox-radio.yaml はカレントディレクトリから自動読み込みされます。
voicevox.url フィールドで VOICEVOX エンジンの URL を指定します（デフォルト: http://localhost:50021）。
話者 ID は vox-radio.yaml のキャラクターカタログから解決されます。

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

### SEE ALSO

* [vox-radio episodegen](vox-radio_episodegen.md)	 - ポッドキャスト制作パイプラインをすべて実行する

