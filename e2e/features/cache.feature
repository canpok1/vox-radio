# language: ja
機能: 過去回の記憶（キャッシュ）
  エピソード履歴は program.id をキーに JSONL キャッシュへ記録され、
  回番号のインクリメントと program.id 必須バリデーションが機能すること。

  @ffmpeg
  シナリオ: 2回実行すると回番号がインクリメントされる
    前提 モックLLMサーバーが起動している
    かつ モックVOICEVOXサーバーが起動している
    かつ モックフィードサーバーが起動している
    かつ テスト用設定一式を配置する
    かつ "vox-radio episodegen --spec episode-spec.yaml --out-dir out1" を実行する
    かつ 終了コードは 0 である
    もし "vox-radio episodegen --spec episode-spec.yaml --out-dir out2" を実行する
    ならば 終了コードは 0 である
    かつ ファイル ".vox-radio/cache/e2e-radio.jsonl" の行数は 2 である
    かつ JSONファイル "out1/e2e-radio_ep001_manifest.json" のキー "episode_number" は数値 1 である
    かつ JSONファイル "out2/e2e-radio_ep002_manifest.json" のキー "episode_number" は数値 2 である

  シナリオ: program.id のない仕様はエラーになる
    前提 モックLLMサーバーが起動している
    かつ モックフィードサーバーが起動している
    かつ テスト用設定一式を配置する
    かつ ファイル "no-id-spec.yaml" を以下の内容で作成する:
      """
      program:
        title: "IDなし番組"
        description: "program.id がない"
      corners:
        - id: "opening"
          title: "オープニング"
          content: "挨拶"
          length_sec: 30
      """
    もし "vox-radio episodegen --spec no-id-spec.yaml --out-dir out" を実行する
    ならば 終了コードは 0 以外である
    かつ 標準エラーに "program.id is required" を含む
