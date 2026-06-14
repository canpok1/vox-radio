# slack-spec.yaml（Slack 投稿設定）リファレンス

> **検証の正**: 設定が正しいかは下記の検証コマンドの結果で判断してください。本ドキュメントと実際の挙動が食い違う場合は、スキルとバイナリの版ずれが原因のことがあります。SKILL.md の「バージョン整合チェック」に従ってスキル / バイナリを揃えてください。

> **検証コマンド**: `vox-radio slackpost check`

`slack-spec.yaml` は `vox-radio slackpost` / `vox-radio slackpost check` が使用する Slack 投稿設定ファイルです。`vox-radio init` で生成されるテンプレートを元に編集してください。

## 必要な Slack スコープ

| スコープ | 用途 |
|---------|------|
| `chat:write` | スレッド返信の投稿 |
| `files:write` | mp3 ファイルのアップロード |
| `files:read` | アップロード完了確認・親メッセージ ts の取得（`files.info`） |

> **スコープを追加した場合は、ワークスペースへのアプリ再インストールが必要です。**
> Slack App の管理画面（OAuth & Permissions）でスコープを更新した後、「Install to Workspace」を再実行してください。

## Bot トークンの設定

Bot トークン（`xoxb-...`）は共通設定 `vox-radio.yaml` の `slack.bot_token_env` で指定した環境変数名から読み込まれます。

```yaml
# vox-radio.yaml
slack:
  bot_token_env: SLACK_BOT_TOKEN   # 環境変数名を指定
```

```bash
# 環境変数を設定してから実行
export SLACK_BOT_TOKEN=xoxb-your-token-here
vox-radio slackpost --manifest output/manifest.json --spec config/slack-spec.yaml
```

## slack-spec.yaml フィールド一覧

| フィールド | 型 | 必須/任意 | 説明 |
|---|---|---|---|
| `slack.channel` | string | 必須 | 投稿先チャンネル ID（`C` で始まる Slack のチャンネル ID） |
| `slack.message.header` | string | 任意 | 親メッセージ（mp3 アップロード時の初期コメント）のテンプレート |
| `slack.message.fallback` | string | 任意 | スレッド返信の通知用プレーンテキストのテンプレート |
| `slack.message.summary` | string | 任意 | スレッド返信の要約 Section テンプレート（空のとき省略） |
| `slack.message.corner` | string | 任意 | コーナー Section テンプレート |
| `slack.message.article` | string | 任意 | 記事 1 件のテンプレート |

`message.*` を省略した場合はコード側のデフォルトテンプレートが適用されます。

## 利用可能なプレースホルダ

| スコープ | プレースホルダ | manifest フィールド |
|---------|----------------|---------------------|
| 全体（header/summary/fallback） | `{title}` `{episode_number}` `{episode_title}` `{description}` `{summary}` `{datetime}` `{audio_file}` | `Manifest` の各 json タグ |
| 全体（header/summary/fallback） | `{credit}` | `Manifest.Credits` を改行結合した文字列。credits が空のとき空文字列に置換される |
| コーナー（`corner`） | `{corner_title}` `{corner_summary}` `{articles}` | `ManifestCorner.Title` / `.Summary` / 記事展開 |
| 記事（`article`） | `{title}` `{url}` | `ArticleRef.Title` / `.URL` |

## 投稿フロー

1. `vox-radio.yaml` から Bot トークンを環境変数で取得
2. `manifest.json` を読み込み、mp3 パスを自動解決（manifest と同じディレクトリ + `audio_file`）
3. `slack-spec.yaml` を読み込み・検証
4. mp3 を Slack へアップロード（親メッセージ = mp3 + 初期コメント）
5. 要約・コーナーをスレッドに返信（要約・コーナーが両方空の場合はスレッド投稿をスキップ）
