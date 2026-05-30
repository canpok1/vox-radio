## vox-radio prune

Remove old episodes, keeping only the most recent N

### Synopsis

Delete audio files and episode entries beyond the configured max_items limit,
then update episodes.json and regenerate feed.xml.

Example:
  vox-radio prune --out-dir public
  vox-radio prune --out-dir public --config config

```
vox-radio prune [flags]
```

### Options

```
      --base-url string   base URL for audio/feed URLs (default: site_url from podcast.yaml)
      --config string     config directory containing podcast.yaml (default "config")
  -h, --help              help for prune
      --out-dir string    output directory for local hosting (required)
```

### SEE ALSO

* [vox-radio](vox-radio.md)	 - AI-powered podcast production tool

