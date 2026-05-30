## vox-radio collect

Collect articles from RSS feeds and URLs

### Synopsis

Collect articles from RSS feeds and web URLs defined in feeds.yaml,
extract their body text, and write the result to articles.json.

Example:
  vox-radio collect --out work/articles.json
  vox-radio collect --out work/articles.json --config config

```
vox-radio collect [flags]
```

### Options

```
      --config string   config directory containing feeds.yaml (default "config")
  -h, --help            help for collect
      --out string      output articles.json path (required)
```

### SEE ALSO

* [vox-radio](vox-radio.md)	 - AI-powered podcast production tool

