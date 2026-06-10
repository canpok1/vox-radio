# language: ja
機能: slackpost による Slack 配信
  manifest と mp3 を Slack へ投稿し、状態ファイルによる冪等な再実行ができること。
  Slack API はモックサーバー（環境変数 VOX_RADIO_SLACK_API_URL で注入）に向ける。

  背景:
    前提 テスト用設定一式を配置する
    かつ fixture "manifest.json" をファイル "manifest.json" として配置する
    かつ ファイル "episode.mp3" を以下の内容で作成する:
      """
      dummy-mp3-content
      """

  シナリオ: --dry-run は API を呼ばずに投稿内容を表示する
    もし "vox-radio slackpost --manifest manifest.json --spec slack-spec.yaml --dry-run" を実行する
    ならば 終了コードは 0 である
    かつ 標準出力に "audio:" を含む
    かつ 標準出力に "header:" を含む
    かつ ファイル "manifest.slackpost-state.json" が存在しない

  シナリオ: モックサーバーへ投稿し状態ファイルが完了を記録する
    前提 モックSlackサーバーが起動している
    もし "vox-radio slackpost --manifest manifest.json --spec slack-spec.yaml" を実行する
    ならば 終了コードは 0 である
    かつ 標準出力に "channel: C0123456789" を含む
    かつ モックSlackサーバーは "files.getUploadURLExternal" を受信した
    かつ モックSlackサーバーは "files.completeUploadExternal" を受信した
    かつ モックSlackサーバーは "chat.postMessage" を受信した
    かつ JSONファイル "manifest.slackpost-state.json" のキー "replied" は真である
    かつ JSONファイル "manifest.slackpost-state.json" のキー "file_id" は文字列 "F-E2E-001" である

  シナリオ: アップロード済み状態からの再実行は音声を二重投稿しない
    前提 モックSlackサーバーが起動している
    かつ ファイル "manifest.slackpost-state.json" を以下の内容で作成する:
      """
      {
        "audio_file": "episode.mp3",
        "episode_number": 1,
        "channel": "C0123456789",
        "file_id": "F-RESUME-01",
        "thread_ts": "",
        "replied": false
      }
      """
    もし "vox-radio slackpost --manifest manifest.json --spec slack-spec.yaml" を実行する
    ならば 終了コードは 0 である
    かつ モックSlackサーバーは "files.getUploadURLExternal" を受信していない
    かつ モックSlackサーバーは "chat.postMessage" を受信した
    かつ JSONファイル "manifest.slackpost-state.json" のキー "replied" は真である

  シナリオ: 投稿完了済みの再実行は API を一切呼ばない
    前提 モックSlackサーバーが起動している
    かつ ファイル "manifest.slackpost-state.json" を以下の内容で作成する:
      """
      {
        "audio_file": "episode.mp3",
        "episode_number": 1,
        "channel": "C0123456789",
        "file_id": "F-DONE-01",
        "thread_ts": "1700000000.000300",
        "replied": true
      }
      """
    もし "vox-radio slackpost --manifest manifest.json --spec slack-spec.yaml" を実行する
    ならば 終了コードは 0 である
    かつ 標準出力に "file_id: F-DONE-01" を含む
    かつ モックSlackサーバーは "files.getUploadURLExternal" を受信していない
    かつ モックSlackサーバーは "chat.postMessage" を受信していない
