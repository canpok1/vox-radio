# 0035. Slack へエピソードを投稿する slackpost サブコマンドを追加する（ADR-0005 の方針転換）

- ステータス: 採用
- 日付: 2026-06-04
- 補足: 配信を RSS に一本化し Slack 専用 API を実装しないとした [ADR-0005](0005-podcast-rss-only-distribution.md) を方針転換する。RSS 配信を廃止するものではなく、Slack 向けの配信経路を追加する。

## コンテキスト

ADR-0005 は配信を Podcast(RSS) に一本化し、Slack は標準の RSS アプリで購読（`/feed subscribe`）する前提で Slack 専用 API を実装しないと決めた。だが運用してみると、RSS 経由では Slack 上で音声を**直接再生できず、いったんダウンロードする手間**がかかることが分かった。毎日 1 回・約 5 分の番組を「できるだけ手間なく聴ける」ことを優先したい。

ADR-0026 が ADR-0012 の分離方針を運用コストを踏まえて部分修正した前例にならい、配信判断を実態に合わせて見直す。

## 決定

- Slack へエピソードを投稿する**新サブコマンド `vox-radio slackpost`** を追加する。feedgen（RSS 配信版）に対する **Slack 配信版**と位置づける。
- 入力は **`manifest.json`（番組内容）+ mp3 ファイル + `slack-spec.yaml`（Slack 専用設定）**。feedgen が cache を正とするのに対し、slackpost は**単一エピソードの投稿**なので per-episode 成果物である manifest を入力にする。
- mp3 は**ファイル本体を Slack にアップロード**し（`files.getUploadURLExternal` → `files.completeUploadExternal`）、Slack 上で直接再生できるようにする。これが本 ADR の主目的（再生の手間削減）を満たす。
- 投稿文は **`slack-spec.yaml` のテンプレート（プレースホルダ置換）で生成**し、LLM は使わない（決定的・低コスト・既存の専用設定駆動方針と一貫）。本文は **Block Kit** で整形し、番組要約・各コーナー・参照記事リンク（manifest の `corners[].articles`）を載せる。
- 認証は **Bot Token を共通設定 `vox-radio.yaml` の `slack.bot_token_env`（環境変数名）で指定**する（LLM と同じ `*_env` パターン。Bot は番組横断で 1 つのため共通設定に置く）。**投稿先チャンネルと表示名・アイコン（`username`/`icon_emoji`/`icon_url`）は番組ごとに `slack-spec.yaml`** で指定する（同一 Bot を番組ごとのチャンネル・表示で使い分ける）。必要スコープは `chat:write` / `chat:write.customize` / `files:write`。
- 表示名・アイコンの上書きは `chat.postMessage`（親の告知メッセージ）にのみ効く。**スレッドに添付する mp3 のファイル共有メッセージは Slack App 既定の名前・アイコンで表示される**（API 制約。許容する）。
- Slack API は**標準 `net/http` で直接呼び出す**（VOICEVOX・OpenAI 互換クライアントと同じ手法）。Slack SDK 依存は追加しない。

## 結果

- Slack 利用者はダウンロードなしでタイムライン上から再生できる。RSS 配信（feedgen）は併存し、Podcast アプリ購読も従来どおり維持される。
- 配信経路が RSS の 1 つから「RSS + Slack」の 2 つになり、その分の実装・運用（Bot Token 管理・チャンネル設定）が増える。ADR-0005 の「配信経路を 1 つに保つ」判断はこの利便性と引き換えに見直す。
- slackpost は feedgen と独立し、cache ではなく manifest を入力にするため、両者で状態源が異なる。単発投稿には manifest が自然なため許容する。
- 投稿の重複防止（同一回の二重投稿ガード）は持たず、呼び出し側（CI が回ごとに 1 回実行）の責務とする。

## 検討した代替案

- **公開 URL へのリンクのみ投稿**: 実装は軽いが Slack 上で直接再生できず、本 ADR の主目的（再生の手間削減）を満たさないため却下。
- **投稿文を LLM 生成**: 表現は豊かになるがコスト・非決定性・プロンプト管理が増え、定型の配信告知には過剰なため却下。
- **Slack SDK（slack-go 等）の導入**: 依存が増える。既存の HTTP 直叩き方針で必要 2 エンドポイントを賄えるため却下。
- **cache を入力にして feedgen と統一**: 複数回をまとめる feed と違い投稿は単発で、per-episode の manifest が自然なため却下。
