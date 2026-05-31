## vox-radio script

Generate a script from collected articles using LLM

### Synopsis

Run the multi-stage LLM pipeline (summarize → plan → write → direct) to
produce script.json from articles.json.

vox-radio.yaml is automatically loaded from the current directory.

Use --step to run a single stage independently:
  summarize  Summarize each article (writes summaries.json)
  plan       Plan corners from summaries (writes rundown.json)
  write      Write lines per corner (writes lines.json)
  direct     Assign SE/speakers to lines (writes script.json)

Example:
  vox-radio script --in work/articles.json --out work/script.json
  vox-radio script --out work/script.json --step plan
  vox-radio script --in work/articles.json --out work/script.json --profile profiles/tech/profile.yaml

```
vox-radio script [flags]
```

### Options

```
  -h, --help             help for script
      --in string        input articles.json path (required for full pipeline or summarize step)
      --out string       output script.json path (required)
      --profile string   profile YAML file path (default "profiles/test/profile.yaml")
      --prompts string   directory containing prompt templates (default "prompts")
      --step string      run a single step: summarize|plan|write|direct
```

### SEE ALSO

* [vox-radio](vox-radio.md)	 - AI-powered podcast production tool

