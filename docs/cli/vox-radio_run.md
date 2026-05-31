## vox-radio run

Run the full podcast production pipeline

### Synopsis

Run collect → script → synth → assemble → publish → prune in one shot.

Intermediate files are written to <out-dir>/intermediate/ and the final
episode.mp3 is placed directly under <out-dir>/.

vox-radio.yaml is automatically loaded from the current directory.

Example:
  vox-radio run
  vox-radio run --out-dir output --profile sample-profiles/tech/profile.yaml
  vox-radio run --hosting ghpages --date 2026-01-01

```
vox-radio run [flags]
```

### Options

```
      --base-url string      base URL for audio/feed URLs (default: site_url from profile)
      --date string          episode date YYYY-MM-DD (default: today)
      --description string   episode description
  -h, --help                 help for run
      --hosting string       hosting type: local or ghpages (default "local")
      --out-dir string       output directory (episode.mp3 placed here, intermediate files in <out-dir>/intermediate/) (default "output")
      --profile string       profile YAML file path (required)
      --prompts string       directory containing prompt templates (default "prompts")
      --title string         episode title (default: <date> <program.title>)
```

### SEE ALSO

* [vox-radio](vox-radio.md)	 - AI-powered podcast production tool

