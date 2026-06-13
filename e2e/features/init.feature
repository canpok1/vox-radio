# language: ja
機能: init コマンドによる設定ファイルの生成
  初回セットアップとして、カレントディレクトリに設定ファイル一式を生成できること。

  シナリオ: init で設定ファイル一式が生成される
    もし "vox-radio init" を実行する
    ならば 終了コードは 0 である
    かつ ファイル "vox-radio.yaml" が存在する
    かつ ファイル "episode-spec.yaml" が存在する
    かつ ファイル "feed-spec.yaml" が存在する
    かつ ファイル "slack-spec.yaml" が存在する
    かつ ファイル "assets/assets.yaml" が存在する

  シナリオ: init の再実行は既存ファイルを上書きしない
    前提 "vox-radio init" を実行する
    かつ ファイル "vox-radio.yaml" を以下の内容で作成する:
      """
      # user-edited-marker
      """
    もし "vox-radio init" を実行する
    ならば 終了コードは 0 である
    かつ 標準出力に "skip: vox-radio.yaml already exists" を含む
    かつ ファイル "vox-radio.yaml" に "user-edited-marker" を含む

  シナリオ: init --sample でサンプル設定一式がカレントディレクトリに生成される
    もし "vox-radio init --sample" を実行する
    ならば 終了コードは 0 である
    かつ ファイル "vox-radio.yaml" が存在する
    かつ ファイル "episode-spec.yaml" が存在する
    かつ ファイル "feed-spec.yaml" が存在する
    かつ ファイル "slack-spec.yaml" が存在する
    かつ ファイル "assets/assets.yaml" が存在する

  シナリオ: init --sample --output-dir sample でサンプル設定一式が sample/ に生成される
    もし "vox-radio init --sample --output-dir sample" を実行する
    ならば 終了コードは 0 である
    かつ ファイル "sample/vox-radio.yaml" が存在する
    かつ ファイル "sample/episode-spec.yaml" が存在する
    かつ ファイル "sample/feed-spec.yaml" が存在する
    かつ ファイル "sample/slack-spec.yaml" が存在する
    かつ ファイル "sample/assets/assets.yaml" が存在する
