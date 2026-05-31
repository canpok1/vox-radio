## vox-radio manifest

Generate a content manifest JSON alongside an episode

### Synopsis

Build a manifest.json that describes the episode content: title, description,
summary, datetime, audio filename, and corners with their articles.

The manifest is intended for use by a separate publishing service to generate
RSS feeds without re-running the full pipeline.

When --script is provided, an LLM-generated summary is added to the manifest
using vox-radio.yaml for LLM configuration.

Example:
  vox-radio manifest --profile sample-profiles/tech_profile.yaml --audio output/episode.mp3 --out output/manifest.json
  vox-radio manifest --profile sample-profiles/tech_profile.yaml --articles output/intermediate/articles.json --audio output/episode.mp3 --out output/manifest.json
  vox-radio manifest --profile sample-profiles/tech_profile.yaml --script output/intermediate/script.json --audio output/episode.mp3 --out output/manifest.json

```
vox-radio manifest [flags]
```

### Options

```
      --articles string   articles.json path (optional; corners get empty articles when omitted)
      --audio string      audio file path; basename is stored in manifest (required)
  -h, --help              help for manifest
      --out string        output manifest.json path (required)
      --profile string    profile YAML file path (required)
      --prompts string    directory containing prompt templates (used when --script is provided) (default "prompts")
      --script string     script.json path (optional; when provided, LLM generates a summary from the script)
```

### SEE ALSO

* [vox-radio](vox-radio.md)	 - AI-powered podcast production tool

