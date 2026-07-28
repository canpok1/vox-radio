# language: ja
機能: retro による自動改善ループ

  @ffmpeg
  シナリオ: 分析付きキャッシュから try/keep ファイルを生成する
    前提 モックLLMサーバーが起動している
    かつ モックVOICEVOXサーバーが起動している
    かつ モックフィードサーバーが起動している
    かつ テスト用設定一式を配置する
    かつ "vox-radio episodegen --spec episode-spec.yaml --out-dir out" を実行する
    もし "vox-radio retro --spec episode-spec.yaml" を実行する
    ならば 終了コードは 0 である
    かつ 標準出力に "try file written" を含む
    かつ 標準出力に "keep file written" を含む
    かつ ファイル ".vox-radio/programs/e2e-radio/try.yaml" が存在する
    かつ ファイル ".vox-radio/programs/e2e-radio/keep.yaml" が存在する
    かつ ファイル ".vox-radio/programs/e2e-radio/try.yaml" に "clear_streak" を含む

  シナリオ: キャッシュが無いときは案内を出して正常終了する
    前提 モックLLMサーバーが起動している
    かつ テスト用設定一式を配置する
    もし "vox-radio retro --spec episode-spec.yaml" を実行する
    ならば 終了コードは 0 である
    かつ 標準出力に "キャッシュが存在しません" を含む
    かつ ファイル ".vox-radio/programs/e2e-radio/try.yaml" が存在しない

  @ffmpeg
  シナリオ: --dry-run はファイルを書かない
    前提 モックLLMサーバーが起動している
    かつ モックVOICEVOXサーバーが起動している
    かつ モックフィードサーバーが起動している
    かつ テスト用設定一式を配置する
    かつ "vox-radio episodegen --spec episode-spec.yaml --out-dir out" を実行する
    もし "vox-radio retro --spec episode-spec.yaml --dry-run" を実行する
    ならば 終了コードは 0 である
    かつ 標準出力に "problems:" を含む
    かつ 標準出力に "try file written" を含まない
    かつ ファイル ".vox-radio/programs/e2e-radio/try.yaml" が存在しない
