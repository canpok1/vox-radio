## vox-radio publish

Publish an episode to the local hosting directory

### Synopsis

Copy the MP3 file into the hosting directory, update episodes.json,
and regenerate feed.xml for RSS distribution.

Example:
  vox-radio publish --in work/episode.mp3 --out-dir public
  vox-radio publish --in work/episode.mp3 --out-dir public --date 2026-01-01 --title "Episode title"
  vox-radio publish --in work/episode.mp3 --out-dir public --profile profiles/tech/profile.yaml

```
vox-radio publish [flags]
```

### Options

```
      --base-url string      base URL for audio/feed URLs (default: site_url from profile)
      --date string          episode date YYYY-MM-DD (default: today)
      --description string   episode description
  -h, --help                 help for publish
      --in string            input mp3 path (required)
      --out-dir string       output directory for local hosting (required)
      --profile string       profile YAML file path (default "profiles/test/profile.yaml")
      --title string         episode title (default: <date> <podcast.title>)
```

### SEE ALSO

* [vox-radio](vox-radio.md)	 - AI-powered podcast production tool

