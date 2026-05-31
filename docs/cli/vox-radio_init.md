## vox-radio init

Generate template config files in the current directory

### Synopsis

Generate vox-radio.yaml (common settings) and profile.yaml (program profile)
in the current directory.

Existing files are skipped individually to prevent accidental overwrites.
If both files already exist, nothing is generated.

After generation, edit the files to configure your LLM API key, program
content, and audio asset paths, then run the pipeline:

  vox-radio run --profile profile.yaml

```
vox-radio init [flags]
```

### Options

```
  -h, --help   help for init
```

### SEE ALSO

* [vox-radio](vox-radio.md)	 - AI-powered podcast production tool

