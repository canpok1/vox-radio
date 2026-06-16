## vox-radio init

テンプレート設定ファイルを生成する

### Synopsis

vox-radio.yaml（共通設定）・episode-spec.yaml（エピソード仕様）・feed-spec.yaml（フィード生成設定）・slack-spec.yaml（Slack 投稿設定）・assets/assets.yaml（アセット設定）を
生成します。出力先は --output-dir で指定します（省略時はカレントディレクトリ）。

--sample を指定すると、ずんだもん・めたんが MC を務めるお天気番組（気象庁の防災情報XMLを利用）の
「すぐ動くサンプル設定一式」のテンプレートを生成します。--sample を使っても出力先は --output-dir で
決まり、省略時はカレントディレクトリです。旧来の sample/ 配下への出力は次のコマンドで再現できます:

  vox-radio init --sample --output-dir sample

--sample-with-assets を指定すると、サンプル音源パック（sample-assets）を前提に、各コーナーへ
ジングル・SE・BGM を割り当て済みの設定一式を生成します（assets/assets.yaml は生成せず、別途
パックを assets/ に展開して使います）。手順は次のとおりです:

  curl -LO "https://github.com/canpok1/vox-radio/releases/download/v$(vox-radio --version | awk '{print $NF}')/vox-radio-sample-assets.zip"
  unzip vox-radio-sample-assets.zip -d assets
  vox-radio init --sample-with-assets
  vox-radio episodegen --spec episode-spec.yaml

生成後は次のコマンドで番組生成を試せます:

  vox-radio episodegen --spec episode-spec.yaml

既存ファイルは上書きを防ぐため個別にスキップされます。
すべてのファイルがすでに存在する場合は何も生成されません。

生成後は LLM API キー・番組内容・音声アセットパスを設定ファイルに記入し、
次のコマンドでパイプラインを実行してください:

  vox-radio episodegen --spec episode-spec.yaml

```
vox-radio init [flags]
```

### Options

```
  -h, --help                 help for init
      --output-dir string    テンプレートの出力先ディレクトリ（省略時はカレントディレクトリ） (default ".")
      --sample               ずんだもん・めたんMCのお天気番組サンプル一式のテンプレートを生成する
      --sample-with-assets   サンプル音源パック（sample-assets）を前提にアセット割り当て済みのサンプル一式を生成する
```

### Options inherited from parent commands

```
      --config string    共通設定 YAML ファイル（vox-radio.yaml）のパス (default "vox-radio.yaml")
      --log-dir string   ログ出力ディレクトリのパス (default ".vox-radio/logs")
```

### SEE ALSO

* [vox-radio](vox-radio.md)	 - AI を使ったポッドキャスト制作ツール

