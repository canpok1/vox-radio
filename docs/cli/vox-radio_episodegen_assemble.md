## vox-radio episodegen assemble

WAV クリップを MP3 エピソードに組み立てる

### Synopsis

script.json と synth が生成したクリップディレクトリを読み込み、
ffmpeg を使ってイントロ・アウトロ・SE をミックスし、最終的な MP3 エピソードを生成します。

実行には ffmpeg および ffprobe が必要です。インストール手順は vox-radio の README を参照してください:
https://github.com/canpok1/vox-radio#readme

例:
  vox-radio episodegen assemble --in work/script.json --clips work/clips --out work/episode.mp3
  vox-radio episodegen assemble --in work/script.json --clips work/clips --out work/episode.mp3 --spec episode-spec.yaml

```
vox-radio episodegen assemble [flags]
```

### Options

```
      --clips string   clips.json と WAV ファイルを含むディレクトリ（必須）
  -h, --help           help for assemble
      --in string      script.json の入力パス（必須）
      --out string     MP3 の出力先パス（必須）
      --spec string    アセット設定を含むエピソード仕様 YAML ファイルのパス（任意）
```

### Options inherited from parent commands

```
      --config string    共通設定 YAML ファイル（vox-radio.yaml）のパス (default "vox-radio.yaml")
      --log-dir string   ログ出力ディレクトリのパス (default ".vox-radio/logs")
```

### SEE ALSO

* [vox-radio episodegen](vox-radio_episodegen.md)	 - ポッドキャスト制作パイプラインをすべて実行する

