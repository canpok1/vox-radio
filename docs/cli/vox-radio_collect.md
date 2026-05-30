## vox-radio collect

Collect articles from RSS feeds and URLs

### Synopsis

Collect articles from RSS feeds and web URLs defined in the profile,
extract their body text, and write the result to articles.json.

Example:
  vox-radio collect --out work/articles.json
  vox-radio collect --out work/articles.json --profile profiles/tech/profile.yaml

```
vox-radio collect [flags]
```

### Options

```
  -h, --help             help for collect
      --out string       output articles.json path (required)
      --profile string   profile YAML file path (default "profiles/test/profile.yaml")
```

### SEE ALSO

* [vox-radio](vox-radio.md)	 - AI-powered podcast production tool

