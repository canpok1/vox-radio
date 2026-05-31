## vox-radio prune

Remove old episodes, keeping only the most recent N

### Synopsis

Delete audio files and episode entries beyond the configured max_items limit,
then update episodes.json and regenerate feed.xml.

Example:
  vox-radio prune --out-dir public
  vox-radio prune --out-dir public --profile sample-profiles/tech/profile.yaml

```
vox-radio prune [flags]
```

### Options

```
      --base-url string   base URL for audio/feed URLs (default: site_url from profile)
  -h, --help              help for prune
      --out-dir string    output directory for local hosting (required)
      --profile string    profile YAML file path (required)
```

### SEE ALSO

* [vox-radio](vox-radio.md)	 - AI-powered podcast production tool

