# language: ja
機能: feedgen による RSS フィード生成
  キャッシュ（JSONL）と feed-spec.yaml から RSS 2.0 + iTunes フィードを生成できること。

  シナリオ: キャッシュから feed.xml を生成する
    前提 fixture "cache-2entries.jsonl" をファイル "cache.jsonl" として配置する
    かつ fixture "feed-spec.yaml" をファイル "feed-spec.yaml" として配置する
    もし "vox-radio feedgen --cache cache.jsonl --spec feed-spec.yaml" を実行する
    ならば 終了コードは 0 である
    かつ 標準出力に "2 items" を含む
    かつ ファイル "public/feed.xml" が存在する
    かつ ファイル "public/feed.xml" に "E2Eテストラジオ" を含む
    かつ ファイル "public/feed.xml" に "https://example.com/episodes/1/episode1.mp3" を含む
    かつ ファイル "public/feed.xml" に "https://example.com/episodes/2/episode2.mp3" を含む
