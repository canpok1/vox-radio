# language: ja
機能: episodegen の各ステップ単体実行
  collect → rundown → script → synth → assemble → manifest の各ステップを
  個別コマンドとして実行でき、中間ファイルが正しく受け渡されること。

  背景:
    前提 モックLLMサーバーが起動している
    かつ モックVOICEVOXサーバーが起動している
    かつ モックフィードサーバーが起動している
    かつ テスト用設定一式を配置する

  シナリオ: collect はフィードから記事を収集する
    もし "vox-radio episodegen collect --spec episode-spec.yaml --out work/01_articles.json" を実行する
    ならば 終了コードは 0 である
    かつ ファイル "work/01_articles.json" が存在する
    かつ ファイル "work/01_articles.json" に "テスト記事1" を含む
    かつ ファイル "work/01_articles.json" に "テスト記事2" を含む

  シナリオ: rundown は記事を選別して番組設計図を生成する
    前提 "vox-radio episodegen collect --spec episode-spec.yaml --out work/01_articles.json" を実行する
    もし "vox-radio episodegen rundown --spec episode-spec.yaml --in work/01_articles.json --out work/02_rundown.json" を実行する
    ならば 終了コードは 0 である
    かつ JSONファイル "work/02_rundown.json" の配列 "corners" の要素数は 2 である
    かつ JSONファイル "work/02_rundown.json" の配列 "corners.1.articles" の要素数は 2 である
    かつ JSONファイル "work/02_rundown.json" のキー "corners.0.flow" は空でない文字列である
    かつ JSONファイル "work/02_rundown.json" のキー "corners.1.articles.0.summary" は空でない文字列である

  シナリオ: script は rundown から台本を生成する
    前提 "vox-radio episodegen collect --spec episode-spec.yaml --out work/01_articles.json" を実行する
    かつ "vox-radio episodegen rundown --spec episode-spec.yaml --in work/01_articles.json --out work/02_rundown.json" を実行する
    もし "vox-radio episodegen script --spec episode-spec.yaml --in work/02_rundown.json --out work/04_script.json" を実行する
    ならば 終了コードは 0 である
    かつ ファイル "work/04_script.json" が存在する
    かつ ファイル "work/03_lines.json" が存在する
    かつ ファイル "work/04_script.json" に "speech" を含む
    かつ ファイル "work/04_script.json" に "zundamon" を含む

  @ffmpeg
  シナリオ: synth は台本から音声クリップを合成する
    前提 "vox-radio episodegen collect --spec episode-spec.yaml --out work/01_articles.json" を実行する
    かつ "vox-radio episodegen rundown --spec episode-spec.yaml --in work/01_articles.json --out work/02_rundown.json" を実行する
    かつ "vox-radio episodegen script --spec episode-spec.yaml --in work/02_rundown.json --out work/04_script.json" を実行する
    もし "vox-radio episodegen synth --in work/04_script.json --out-dir clips" を実行する
    ならば 終了コードは 0 である
    かつ ファイル "clips/clip_000.wav" のサイズは 0 より大きい
    かつ ファイル "clips/clips.json" が存在する
    かつ JSONファイル "clips/clips.json" のキー "clips.0.speaker_role" は文字列 "zundamon" である

  @ffmpeg
  シナリオ: assemble はクリップを結合して MP3 を生成する
    前提 "vox-radio episodegen collect --spec episode-spec.yaml --out work/01_articles.json" を実行する
    かつ "vox-radio episodegen rundown --spec episode-spec.yaml --in work/01_articles.json --out work/02_rundown.json" を実行する
    かつ "vox-radio episodegen script --spec episode-spec.yaml --in work/02_rundown.json --out work/04_script.json" を実行する
    かつ "vox-radio episodegen synth --in work/04_script.json --out-dir clips" を実行する
    もし "vox-radio episodegen assemble --spec episode-spec.yaml --in work/04_script.json --clips clips --out episode.mp3" を実行する
    ならば 終了コードは 0 である
    かつ ファイル "episode.mp3" のサイズは 0 より大きい

  シナリオ: manifest は --lines 指定で要約付きマニフェストを生成する
    前提 "vox-radio episodegen collect --spec episode-spec.yaml --out work/01_articles.json" を実行する
    かつ "vox-radio episodegen rundown --spec episode-spec.yaml --in work/01_articles.json --out work/02_rundown.json" を実行する
    かつ "vox-radio episodegen script --spec episode-spec.yaml --in work/02_rundown.json --out work/04_script.json" を実行する
    かつ ファイル "episode.mp3" を以下の内容で作成する:
      """
      dummy-mp3-content
      """
    もし "vox-radio episodegen manifest --spec episode-spec.yaml --rundown work/02_rundown.json --lines work/03_lines.json --audio episode.mp3 --out manifest.json" を実行する
    ならば 終了コードは 0 である
    かつ JSONファイル "manifest.json" のキー "title" は文字列 "E2Eテストラジオ" である
    かつ JSONファイル "manifest.json" のキー "summary" は空でない文字列である
    かつ JSONファイル "manifest.json" のキー "episode_title" は空でない文字列である
    かつ JSONファイル "manifest.json" の配列 "conversation_notes" の要素数は 1 以上である
    かつ JSONファイル "manifest.json" のキー "corners.0.summary" は空でない文字列である
