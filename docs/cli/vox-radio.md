## vox-radio

AI を使ったポッドキャスト制作ツール

### Synopsis

vox-radio は AI を活用したポッドキャストエピソード制作 CLI ツールです。

記事収集・LLM による台本生成・音声合成・音声組み立て・コンテンツマニフェスト出力まで、
フルパイプラインをカバーします。

### Options

```
      --config string    共通設定 YAML ファイル（vox-radio.yaml）のパス (default "vox-radio.yaml")
  -h, --help             help for vox-radio
      --log-dir string   ログ出力ディレクトリのパス (default ".vox-radio/logs")
```

### SEE ALSO

* [vox-radio assets](vox-radio_assets.md)	 - アセット設定ファイルを管理するコマンド群
* [vox-radio config](vox-radio_config.md)	 - 設定ファイル（vox-radio.yaml）を操作するサブコマンド群
* [vox-radio episodegen](vox-radio_episodegen.md)	 - ポッドキャスト制作パイプラインをすべて実行する
* [vox-radio feedgen](vox-radio_feedgen.md)	 - キャッシュから RSS フィード（feed.xml）を生成する
* [vox-radio init](vox-radio_init.md)	 - テンプレート設定ファイルを生成する
* [vox-radio install](vox-radio_install.md)	 - vox-radio のエージェントスキルやリソースをインストールする
* [vox-radio render](vox-radio_render.md)	 - manifest を text/template でレンダリングして出力する
* [vox-radio slackpost](vox-radio_slackpost.md)	 - manifest を入力に mp3 を Slack へ投稿する

