## vox-radio init

カレントディレクトリにテンプレート設定ファイルを生成する

### Synopsis

vox-radio.yaml（共通設定）・episode-spec.yaml（エピソード仕様）・feed-spec.yaml（フィード生成設定）・slack-spec.yaml（Slack 投稿設定）・assets/assets.yaml（アセット設定）を
カレントディレクトリに生成します。

--sample を指定すると、ずんだもん・めたんが MC を務めるお天気番組（気象庁の防災情報XMLを利用）の
「すぐ動くサンプル設定一式」を sample/ ディレクトリに生成します。生成されるのは
sample/vox-radio.yaml・sample/episode-spec.yaml・sample/feed-spec.yaml・
sample/slack-spec.yaml・sample/assets/assets.yaml の 5 ファイルです。生成後は次のコマンドで
番組生成を試せます:

  vox-radio --config sample/vox-radio.yaml episodegen --spec sample/episode-spec.yaml

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
  -h, --help     help for init
      --sample   ずんだもん・めたんMCのお天気番組サンプル一式を sample/ に生成する
```

### Options inherited from parent commands

```
      --config string   共通設定 YAML ファイル（vox-radio.yaml）のパス (default "vox-radio.yaml")
```

### SEE ALSO

* [vox-radio](vox-radio.md)	 - AI を使ったポッドキャスト制作ツール

