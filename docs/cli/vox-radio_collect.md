## vox-radio collect

Collect articles from RSS feeds and URLs per corner

### Synopsis

Collect articles from RSS feeds and web URLs defined in corners[].source,
extract their body text, and write the result to articles.json.

Corners without a source field are skipped.

Example:
  vox-radio collect --out work/articles.json
  vox-radio collect --out work/articles.json --profile sample-profiles/tech_profile.yaml

```
vox-radio collect [flags]
```

### Options

```
  -h, --help             help for collect
      --out string       output articles.json path (required)
      --profile string   profile YAML file path (required)
```

### SEE ALSO

* [vox-radio](vox-radio.md)	 - AI-powered podcast production tool

