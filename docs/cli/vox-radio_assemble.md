## vox-radio assemble

WAV クリップを MP3 エピソードに組み立てる

### Synopsis

script.json と synth が生成したクリップディレクトリを読み込み、
ffmpeg を使ってイントロ・アウトロ・SE をミックスし、最終的な MP3 エピソードを生成します。

例:
  vox-radio assemble --in work/script.json --clips work/clips --out work/episode.mp3
  vox-radio assemble --in work/script.json --clips work/clips --out work/episode.mp3 --profile sample-profiles/tech_profile.yaml

```
vox-radio assemble [flags]
```

### Options

```
      --clips string     clips.json と WAV ファイルを含むディレクトリ（必須）
  -h, --help             help for assemble
      --in string        script.json の入力パス（必須）
      --out string       MP3 の出力先パス（必須）
      --profile string   アセット設定を含むプロファイル YAML ファイルのパス（任意）
```

### SEE ALSO

* [vox-radio](vox-radio.md)	 - AI を使ったポッドキャスト制作ツール

