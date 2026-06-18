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

## Bot トークンとチャンネル ID の設定

Bot トークン（`xoxb-...`）は共通設定 `vox-radio.yaml` の `slack.bot_token_env` で指定した環境変数名から読み込まれます。
投稿先チャンネル ID は `slack-spec.yaml` の `slack.channel_env` で指定した環境変数名から読み込まれます。

```yaml
# vox-radio.yaml
slack:
  bot_token_env: SLACK_BOT_TOKEN   # 環境変数名を指定
```

```yaml
# slack-spec.yaml
slack:
  channel_env: SLACK_CHANNEL_ID    # 環境変数名を指定
```

```bash
# 環境変数を設定してから実行
export SLACK_BOT_TOKEN=xoxb-your-token-here
export SLACK_CHANNEL_ID=C0123456789
vox-radio slackpost --manifest output/manifest.json --spec config/slack-spec.yaml
```

## slack-spec.yaml フィールド一覧

| フィールド | 型 | 必須/任意 | 説明 |
|---|---|---|---|
| `slack.channel_env` | string | 必須 | 投稿先チャンネル ID を保持する環境変数名（実行時に `os.Getenv` で解決） |
| `slack.message.parent` | string | 任意 | 親メッセージ（mp3 アップロード初期コメント）のテンプレートファイルパス |
| `slack.message.thread` | string | 任意 | スレッド返信本文のテンプレートファイルパス（3000 文字超は自動分割） |
| `slack.message.fallback` | string | 任意 | スレッド通知用プレーンテキストのテンプレートファイルパス |

`message.*` を省略した場合は組み込みのデフォルトテンプレートが使われます。

## テンプレートファイルのパス指定

- **相対パス**: `slack-spec.yaml` のあるディレクトリ基準で解決されます
- **絶対パス**: そのまま使用されます

```yaml
slack:
  channel_env: SLACK_CHANNEL_ID
  message:
    parent: "slack-parent.tmpl"          # 相対パス（slack-spec.yaml と同じディレクトリ）
    thread: "/abs/path/slack-thread.tmpl" # 絶対パス
```

`vox-radio init` を実行すると `slack-spec.yaml`・`slack-parent.tmpl`・`slack-thread.tmpl` が同時に生成されます。

## テンプレートの書き方（Go text/template）

テンプレートは Go 標準の [`text/template`](https://pkg.go.dev/text/template) 記法を使います。

### データ文脈・テンプレート関数

利用できるフィールドと関数の一覧は [manifest.md](manifest.md) を参照してください。

### レシピ集

#### 回番号・サブタイトルの条件付き表示

```
{{- /* EpisodeNumber が 0 のとき「第N回」を省略、EpisodeTitle が空のとき「」を省略 */}}
🎙️ {{.Title}}{{if .EpisodeNumber}} 第{{.EpisodeNumber}}回{{end}}{{if .EpisodeTitle}}「{{.EpisodeTitle}}」{{end}}
```

#### URL なし記事のスキップ

```
{{range .Corners}}
*{{.Title}}*
{{.Summary}}
{{range .Articles}}{{if .URL}} • <{{.URL}}|{{.Title}}>
{{end}}{{end}}{{end}}
```

#### 特定コーナーを ID で取り出す

```
{{with corner "news"}}
*{{.Title}}*
{{.Summary}}
{{end}}
```

#### 特定コーナーだけ別表記（`eq`/`ne`）

```
{{range .Corners}}
{{if eq .ID "oheri"}}お便り: {{.Summary}}
{{else}}*{{.Title}}*
{{.Summary}}
{{range .Articles}}{{if .URL}} • <{{.URL}}|{{.Title}}>{{end}}{{end}}
{{end}}
{{end}}
```

#### URL 付き記事があるコーナーだけ見出しを出す（`hasLinks`）

```
{{range .Corners}}{{if hasLinks .}}
*{{.Title}}*
{{range .Articles}}{{if .URL}} • <{{.URL}}|{{.Title}}>
{{end}}{{end}}
{{end}}{{end}}
```

#### クレジット表示

```
{{range .Credits}} • {{.}}
{{end}}
```

### スレッドの 3000 文字自動分割

`thread` テンプレートのレンダリング結果が 3000 文字（Rune）を超える場合、`vox-radio` は改行境界で自動的に複数の Section ブロックに分割します。テンプレート側での分割指定は不要です。

## 投稿フロー

1. `vox-radio.yaml` から Bot トークンを環境変数で取得
2. `manifest.json` を読み込み、mp3 パスを自動解決（manifest と同じディレクトリ + `audio_file`）
3. `slack-spec.yaml` を読み込み・検証（テンプレートファイルの存在・構文も検証）
4. `slack-spec.yaml` の `slack.channel_env` が指す環境変数からチャンネル ID を取得
5. mp3 を Slack へアップロード（親メッセージ = mp3 + 初期コメント）
6. `thread` テンプレートをレンダリングし、3000 文字以下の Section ブロックに分割してスレッドに返信
   （`thread` 結果が空のとき、スレッド投稿はスキップ）

## 検証コマンド（slackpost check）

```bash
vox-radio slackpost check slack-spec.yaml
```

以下を検証します:

- strict パース: 未知キー（typo）をエラー化
- `slack.channel_env` の存在チェック（env の実値は検証しない）
- `slack.message.{parent,thread,fallback}` 指定時: ファイル存在・読み込み・`template.Parse` 構文検証
