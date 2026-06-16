# language: ja
@ffmpeg
機能: episodegen 一括実行
  collect → rundown → script → synth → assemble → manifest の全パイプラインを
  一括実行し、エピソード一式が生成されること。

  シナリオ: 全パイプラインを一括実行してエピソードを生成する
    前提 モックLLMサーバーが起動している
    かつ モックVOICEVOXサーバーが起動している
    かつ モックフィードサーバーが起動している
    かつ テスト用設定一式を配置する
    もし "vox-radio episodegen --spec episode-spec.yaml --out-dir out" を実行する
    ならば 終了コードは 0 である
    かつ 標準出力に "pipeline complete" を含む
    かつ ファイル "out/e2e-radio_ep001.mp3" のサイズは 0 より大きい
    かつ ファイル "out/intermediate/e2e-radio_ep001/01_articles.json" が存在する
    かつ ファイル "out/intermediate/e2e-radio_ep001/02_rundown.json" が存在する
    かつ ファイル "out/intermediate/e2e-radio_ep001/03_lines.json" が存在する
    かつ ファイル "out/intermediate/e2e-radio_ep001/04_script.json" が存在する
    かつ JSONファイル "out/e2e-radio_ep001_manifest.json" のキー "title" は文字列 "E2Eテストラジオ" である
    かつ JSONファイル "out/e2e-radio_ep001_manifest.json" のキー "episode_number" は数値 1 である
    かつ JSONファイル "out/e2e-radio_ep001_manifest.json" のキー "audio_file" は文字列 "e2e-radio_ep001.mp3" である
    かつ ファイル ".vox-radio/cache/e2e-radio.jsonl" の行数は 1 である
