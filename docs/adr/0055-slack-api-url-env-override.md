# 0055. Slack API URL の環境変数オーバーライド

- ステータス: 採用
- 日付: 2026-06-10

## コンテキスト

e2e テスト（ADR-0054）で slackpost の実投稿フロー（3段階アップロード → スレッド ts 解決 → スレッド返信 → 状態ファイルによる冪等な再開）をモックサーバーに対して検証したい。しかし Slack クライアントは `slackgo.New(token)` 固定で API の接続先 URL を差し替える手段がなく、実投稿フローは `slackpost check` と `--dry-run` までしかテストできなかった。

## 決定

環境変数 `VOX_RADIO_SLACK_API_URL` で Slack API のベース URL を上書きできるようにする。VOICEVOX URL の環境変数オーバーライド（ADR-0042 の `VOX_RADIO_VOICEVOX_URL`）と同じパターンに揃える。`config.SlackConfig.EffectiveAPIURL()` が環境変数を読んで返し（slack-go はベース URL とメソッド名を単純連結するため末尾スラッシュを補正する）、`slack.NewPoster(token, apiURL)` が非空のとき `slackgo.OptionAPIURL` を適用する。未設定時は空文字を返し slack-go のデフォルト URL を使う。

## 結果

e2e テストで実投稿フロー（アップロード3段階・files.info ポーリング・chat.postMessage・状態ファイル再開時の二重投稿なし）をモックサーバーで検証できるようになった。設定ファイルにフィールドを増やさず、通常利用時の挙動は変わらない。環境変数の設定ミスで本番投稿先が変わるリスクはあるが、`VOX_RADIO_` プレフィックスで意図的な設定に限られ、slackpost の Long 説明にテスト・検証用と明記した。

## 検討した代替案

- **`vox-radio.yaml` の `slack` セクションに `api_url` フィールドを追加**: 通常運用で使わないテスト用設定が恒久的な設定ファイルスキーマに混入する。テスト時のみ有効にしたい一時的な上書きには環境変数が適合するため却下。
- **Poster をテストから直接注入**: ユニットテストでは既に `slack.Run(opts, poster)` で注入できるが、e2e はバイナリを別プロセスで起動するため注入できない。プロセス境界を越えられる環境変数が必要。
