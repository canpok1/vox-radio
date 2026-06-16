# language: ja
機能: 設定ファイルの検証コマンド
  各設定ファイルを strict モードで検証し、typo や必須フィールド欠落を非ゼロ終了で検知できること。

  背景:
    前提 "vox-radio init" を実行する

  シナリオ: config check はテンプレート設定を通す
    もし "vox-radio config check" を実行する
    ならば 終了コードは 0 である
    かつ 標準出力に "OK: vox-radio.yaml" を含む

  シナリオ: config check は未知キーをエラーにする
    前提 ファイル "broken-config.yaml" を以下の内容で作成する:
      """
      llm:
        provider: openai
        openai:
          base_url: https://example.com
          api_key_env: TEST_KEY
          model: test
      unknown_key: true
      """
    もし "vox-radio --config broken-config.yaml config check" を実行する
    ならば 終了コードは 0 以外である
    かつ 標準エラーに "unknown_key" を含む

  シナリオ: episodegen check はテンプレート仕様を通す
    もし "vox-radio episodegen check episode-spec.yaml" を実行する
    ならば 終了コードは 0 である
    かつ 標準出力に "OK: episode-spec.yaml" を含む

  シナリオ: episodegen check は program.id 欠落をエラーにする
    前提 ファイル "broken-spec.yaml" を以下の内容で作成する:
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
    もし "vox-radio episodegen check broken-spec.yaml" を実行する
    ならば 終了コードは 0 以外である
    かつ 標準エラーに "program.id is required" を含む

  シナリオ: assets check はテンプレート設定を通す
    もし "vox-radio assets check assets/assets.yaml" を実行する
    ならば 終了コードは 0 である
    かつ 標準出力に "OK: assets/assets.yaml" を含む

  シナリオ: feedgen check はテンプレート設定を通す
    もし "vox-radio feedgen check feed-spec.yaml" を実行する
    ならば 終了コードは 0 である
    かつ 標準出力に "OK: feed-spec.yaml" を含む

  シナリオ: feedgen check はプレースホルダ欠落をエラーにする
    前提 ファイル "broken-feed-spec.yaml" を以下の内容で作成する:
      """
      feed:
        language: "ja"
        email: "test@example.com"
        category: "Technology"
        explicit: false
        cover_image_url: ""
        site_url: "https://example.com"
        audio_url_template: "https://example.com/episodes/fixed.mp3"
      output:
        public: "public"
      """
    もし "vox-radio feedgen check broken-feed-spec.yaml" を実行する
    ならば 終了コードは 0 以外である
    かつ 標準エラーに "audio_url_template" を含む

  シナリオ: slackpost check はテンプレート設定を通す
    もし "vox-radio slackpost check slack-spec.yaml" を実行する
    ならば 終了コードは 0 である
    かつ 標準出力に "OK: slack-spec.yaml" を含む

  シナリオ: slackpost check はチャンネル欠落をエラーにする
    前提 ファイル "broken-slack-spec.yaml" を以下の内容で作成する:
      """
      slack:
        channel_env: ""
      """
    もし "vox-radio slackpost check broken-slack-spec.yaml" を実行する
    ならば 終了コードは 0 以外である
    かつ 標準エラーに "channel" を含む
