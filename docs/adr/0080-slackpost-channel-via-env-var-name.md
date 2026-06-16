# 0080. slackpost の投稿先チャンネルを環境変数名の間接指定にする

- ステータス: 採用
- 日付: 2026-06-16

## コンテキスト

投稿先チャンネルは `slack-spec.yaml` の `slack.channel` に ID を直値で書いていた。CI の自動投稿で本番／検証など環境ごとに投稿先を切り替えたいが、直値だと spec を分けるか書き換える必要があり、設定を共有したまま env だけで切り替えられない。env 切り替えには固定上書き env（ADR-0042/0055）と、Bot トークンの間接指定（`bot_token_env`）の先例がある。

## 決定

`slack.channel`（直値）を廃止し、env 変数名を指定する `slack.channel_env` へ置き換える。`bot_token_env` と同じ間接指定に揃える。チャンネル ID は実行時に `os.Getenv(channel_env)` で解決し、トークン同様に CLI 層（`slackpost.go`）で解決して `slack.Run` へ渡す。`slackpost check` は `channel_env` の存在のみ検証し、値は実行時に検証する（未設定かつ非 dry-run でエラー）。

## 結果

CI は env の値を差し替えるだけで投稿先を切り替えられ、spec を共有・固定したまま運用できる。トークン・チャンネルとも扱いが揃う。一方 `slack.channel` を含む既存 spec は動かなくなる破壊的変更で、利用者は `channel_env` への移行が必要。

## 検討した代替案

- **固定上書き env（`VOX_RADIO_SLACK_CHANNEL`）**: ADR-0042/0055 と同型だが変数名が固定で投げ分けに弱く、env 名を spec で決める方が CI に馴染むため却下。
- **`--channel` フラグ**: CI の secret/variable と直接噛み合う env 方式が適合するため却下。
- **`channel` と `channel_env` の併用**: 後方互換だが解決順序の分岐で複雑化するため却下。
