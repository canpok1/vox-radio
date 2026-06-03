## vox-radio episodegen

ポッドキャスト制作パイプラインをすべて実行する

### Synopsis

collect → rundown → script → synth → assemble → manifest を一括実行します。

中間ファイルは <out-dir>/intermediate/ に書き出され、
最終的な episode.mp3 は <out-dir>/ 直下に配置されます。

vox-radio.yaml はカレントディレクトリから自動読み込みされます。

例:
  vox-radio episodegen
  vox-radio episodegen --out-dir output --profile sample-profiles/tech_profile.yaml

```
vox-radio episodegen [flags]
```

### Options

```
  -h, --help             help for episodegen
      --out-dir string   出力ディレクトリ（episode.mp3 をここに配置し、中間ファイルは <out-dir>/intermediate/ に配置） (default "output")
      --profile string   プロファイル YAML ファイルのパス（必須）
```

### SEE ALSO

* [vox-radio](vox-radio.md)	 - AI を使ったポッドキャスト制作ツール
* [vox-radio episodegen assemble](vox-radio_episodegen_assemble.md)	 - WAV クリップを MP3 エピソードに組み立てる
* [vox-radio episodegen collect](vox-radio_episodegen_collect.md)	 - コーナーごとに RSS/Atom フィードと URL から記事を収集する
* [vox-radio episodegen manifest](vox-radio_episodegen_manifest.md)	 - エピソードのコンテンツマニフェスト JSON を生成する
* [vox-radio episodegen rundown](vox-radio_episodegen_rundown.md)	 - 収集記事から番組設計図（rundown）を生成する
* [vox-radio episodegen script](vox-radio_episodegen_script.md)	 - LLM を使って rundown から台本を生成する
* [vox-radio episodegen synth](vox-radio_episodegen_synth.md)	 - 台本から音声クリップを合成する

