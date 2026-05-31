## vox-radio manifest

Generate a content manifest JSON alongside an episode

### Synopsis

Build a manifest.json that describes the episode content: title, description,
datetime, audio filename, and corners with their articles.

The manifest is intended for use by a separate publishing service to generate
RSS feeds without re-running the full pipeline.

Example:
  vox-radio manifest --profile profiles/tech/profile.yaml --audio output/episode.mp3 --out output/manifest.json
  vox-radio manifest --profile profiles/tech/profile.yaml --articles output/intermediate/articles.json --audio output/episode.mp3 --out output/manifest.json

```
vox-radio manifest [flags]
```

### Options

```
      --articles string   articles.json path (optional; corners get empty articles when omitted)
      --audio string      audio file path; basename is stored in manifest (required)
  -h, --help              help for manifest
      --out string        output manifest.json path (required)
      --profile string    profile YAML file path (default "profiles/test/profile.yaml")
```

### SEE ALSO

* [vox-radio](vox-radio.md)	 - AI-powered podcast production tool

