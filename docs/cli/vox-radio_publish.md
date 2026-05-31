## vox-radio publish

Publish an episode to the hosting directory

### Synopsis

Copy the MP3 file into the hosting directory, update episodes.json,
and regenerate feed.xml for RSS distribution.

Hosting types:
  local    Write files to a local directory (default).
  ghpages  Write files to a local git working tree and push to gh-pages as an orphan commit.

Example:
  vox-radio publish --in work/episode.mp3 --out-dir public
  vox-radio publish --in work/episode.mp3 --out-dir public --date 2026-01-01 --title "Episode title"
  vox-radio publish --in work/episode.mp3 --out-dir public --profile profiles/tech/profile.yaml
  vox-radio publish --in work/episode.mp3 --out-dir public --hosting ghpages

```
vox-radio publish [flags]
```

### Options

```
      --base-url string      base URL for audio/feed URLs (default: site_url from profile)
      --date string          episode date YYYY-MM-DD (default: today)
      --description string   episode description
  -h, --help                 help for publish
      --hosting string       hosting type: local or ghpages (default "local")
      --in string            input mp3 path (required)
      --out-dir string       output directory for hosting (required)
      --profile string       profile YAML file path (default "profiles/test/profile.yaml")
      --title string         episode title (default: <date> <program.title>)
```

### SEE ALSO

* [vox-radio](vox-radio.md)	 - AI-powered podcast production tool

