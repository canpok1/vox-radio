## vox-radio synth

Synthesize voice clips from a script

### Synopsis

Read script.json and call VOICEVOX to synthesize each line into WAV clips.
The output directory will contain per-line WAV files and a clips.json manifest.

vox-radio.yaml is automatically loaded from the current directory.
The voicevox.url field specifies the VOICEVOX engine URL (default: http://localhost:50021).

Example:
  vox-radio synth --in work/script.json --out-dir work/clips

```
vox-radio synth [flags]
```

### Options

```
  -h, --help             help for synth
      --in string        input script.json path (required)
      --out-dir string   output directory for WAV clips (required)
```

### SEE ALSO

* [vox-radio](vox-radio.md)	 - AI-powered podcast production tool

