## vox-radio install

vox-radio のエージェントスキルやリソースをインストールする

### Synopsis

vox-radio のエージェントスキルやリソースを現在のプロジェクトへインストールします。

--skills フラグを指定すると、LLM エージェント向けのスキルファイル一式を
<skills-dir>/vox-radio/ 配下にインストールします（既定: .claude/skills/vox-radio/）。

```
vox-radio install [flags]
```

### Options

```
      --force               既存ファイルを上書きする
  -h, --help                help for install
      --skills              エージェントスキルを <skills-dir>/vox-radio/ にインストールする
      --skills-dir string   スキルのインストール先ディレクトリ（このディレクトリ下に vox-radio/ を作成する） (default ".claude/skills")
```

### Options inherited from parent commands

```
      --config string    共通設定 YAML ファイル（vox-radio.yaml）のパス (default "vox-radio.yaml")
      --log-dir string   ログ出力ディレクトリのパス (default ".vox-radio/logs")
```

### SEE ALSO

* [vox-radio](vox-radio.md)	 - AI を使ったポッドキャスト制作ツール

