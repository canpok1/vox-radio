## vox-radio assemble

Assemble WAV clips into an MP3 episode

### Synopsis

Read script.json and the clips directory produced by synth, then use ffmpeg
to mix intro/outro/SE and produce a final MP3 episode file.

Example:
  vox-radio assemble --in work/script.json --clips work/clips --out work/episode.mp3
  vox-radio assemble --in work/script.json --clips work/clips --out work/episode.mp3 --config config

```
vox-radio assemble [flags]
```

### Options

```
      --clips string    directory containing clips.json and WAV files (required)
      --config string   config directory for assets (optional)
  -h, --help            help for assemble
      --in string       input script.json path (required)
      --out string      output mp3 path (required)
```

### SEE ALSO

* [vox-radio](vox-radio.md)	 - AI-powered podcast production tool

