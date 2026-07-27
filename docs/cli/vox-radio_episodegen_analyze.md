## vox-radio episodegen analyze

台本とタイミング情報から回の分析（07_analysis.json）を生成する

### Synopsis

コーナーの目標尺・実測尺・話者別セリフ数などの機械集計指標に加え、
LLM でこの回の課題（findings）と会話の型（patterns）を抽出し、07_analysis.json を生成します。

episodegen 本体は番組生成の一部として analyze を自動実行しキャッシュへ蓄積します。
このコマンドは既存回の中間ファイルから分析を作り直したいとき（抽出観点を変えた場合など）に使います。

共通設定ファイルのパスは --config フラグで指定します（省略時は vox-radio.yaml）。

例:
  vox-radio episodegen analyze --spec episode-spec.yaml --lines output/intermediate/prog_ep001/03_lines.json --timeline output/intermediate/prog_ep001/06_timeline.json --out output/intermediate/prog_ep001/07_analysis.json

```
vox-radio episodegen analyze [flags]
```

### Options

```
  -h, --help               help for analyze
      --lines string       03_lines.json のパス（必須）
      --out string         07_analysis.json の出力先パス（必須）
      --proofread string   04_proofread.json のパス（任意。無ければ校正修正0件として扱う）
      --spec string        エピソード仕様 YAML ファイルのパス（必須）
      --timeline string    06_timeline.json のパス（任意。無ければコーナー実測尺は0として扱う）
```

### Options inherited from parent commands

```
      --config string     共通設定 YAML ファイル（vox-radio.yaml）のパス (default "vox-radio.yaml")
      --env-file string   環境変数を読み込む env ファイルのパス（未指定時は .env を自動読込、不在は無視） (default ".env")
      --log-dir string    ログ出力ディレクトリのパス (default ".vox-radio/logs")
```

### SEE ALSO

* [vox-radio episodegen](vox-radio_episodegen.md)	 - ポッドキャスト制作パイプラインをすべて実行する

