## vox-radio install

vox-radio のエージェントスキルやリソースをインストールする

### Synopsis

vox-radio のエージェントスキルやリソースを現在のプロジェクトへインストールします。

--skills フラグを指定すると、LLM エージェント向けのスキルファイル一式を
.claude/skills/vox-radio/ 配下にインストールします。

```
vox-radio install [flags]
```

### Options

```
      --force    既存ファイルを上書きする
  -h, --help     help for install
      --skills   エージェントスキルを .claude/skills/vox-radio/ にインストールする
```

### Options inherited from parent commands

```
      --config string    共通設定 YAML ファイル（vox-radio.yaml）のパス (default "vox-radio.yaml")
      --log-dir string   ログ出力ディレクトリのパス (default ".vox-radio/logs")
```

### SEE ALSO

* [vox-radio](vox-radio.md)	 - AI を使ったポッドキャスト制作ツール

