# vox-radio

## 開発環境のセットアップ

### 前提

- Docker / Dev Containers が使える環境

### 手順

1. `.devcontainer/.env-template` をコピーして `.devcontainer/.env` を作成する

   ```bash
   cp .devcontainer/.env-template .devcontainer/.env
   ```

2. `.devcontainer/.env` に各自の値を設定する

   | 変数名 | 説明 |
   |--------|------|
   | `GH_TOKEN` | GitHub Personal Access Token |
   | `GEMINI_API_KEY` | Gemini API キー（[Google AI Studio](https://aistudio.google.com/) で取得） |

3. devcontainer をリビルドして起動する

> **注意:** `.devcontainer/.env` には秘密情報が含まれるため、コミットしないこと（`.gitignore` で除外済み）。

## リリース設定の検証

`.goreleaser.yaml` を編集した後は、CI を待たずにローカルで構文・設定を検証できます。

```bash
make release-check
```

`goreleaser check` を実行し、設定の構文エラーや不整合を検出します。`goreleaser` は devcontainer 起動時または `make setup` 実行時に自動インストールされます。

## CLIの使い方

### バイナリインストール（リリース版）

GitHub Releases のバイナリを `curl` のワンライナーで導入できます。

```bash
# 最新版をインストール
curl -fsSL https://raw.githubusercontent.com/canpok1/vox-radio/main/scripts/install-vox-radio.sh | bash

# バージョンを指定してインストール（引数で渡す）
curl -fsSL https://raw.githubusercontent.com/canpok1/vox-radio/main/scripts/install-vox-radio.sh | bash -s -- v0.0.1

# 設置先を変更（環境変数）
curl -fsSL https://raw.githubusercontent.com/canpok1/vox-radio/main/scripts/install-vox-radio.sh | INSTALL_DIR=$HOME/.local/bin bash -s -- v0.0.1
```

デフォルトの設置先は `/usr/local/bin` です。書き込み権限がない場合は自動で `sudo` にフォールバックします。

### ビルド

```bash
make build
```

バージョンを埋め込む場合は `VERSION` を指定します。

```bash
make build VERSION=v0.1.0
```

ビルドしたバイナリのバージョンを確認するには `--version` フラグを使います。

```bash
vox-radio --version
```

### 動作確認用サンプル実行

`examples/tech.yaml` を使ってパイプライン全体を試すには `make run-sample` を実行します。

```bash
make run-sample
```

出力先は `output/<YYYYMMDDHHMMSS>/` ディレクトリになります（例: `output/20260601053357/episode.mp3`）。

プロファイルや出力先を変更する場合は `PROFILE` / `OUT_DIR` 変数で上書きできます。

```bash
# 別のプロファイルを使う
make run-sample PROFILE=examples/other.yaml

# 出力先を指定する
make run-sample OUT_DIR=output/test
```

> **前提条件:** `GEMINI_API_KEY` 環境変数と VOICEVOX Engine が必要です。

### パイプライン概要

vox-radio は以下のパイプラインでポッドキャストを自動生成します。

```
collect → rundown → script → synth → assemble → manifest
```

| コマンド | 概要 |
|----------|------|
| `init` | カレントディレクトリに `vox-radio.yaml`・`episode-spec.yaml`・`feed-spec.yaml` のテンプレートを生成する（初回セットアップ用） |
| `install --skills` | LLM エージェント向けスキルファイル（SKILL.md + references/*.md）を `.claude/skills/vox-radio/` にインストールする |
| `episodegen` | collect → rundown → script → synth → assemble → manifest の全パイプラインを一括実行し 1 本のエピソードを生成する |
| `episodegen collect` | `corners[].source` に定義したフィード・URL からコーナーごとに記事を収集し `01_articles.json` を生成する |
| `episodegen rundown` | LLM が収集記事を選別し、コーナーごとの話の流れと要約を含む `02_rundown.json` を生成する（番組設計図） |
| `episodegen script` | rundown を LLM に渡して台本 `04_script.json` を生成する（write → direct の多段パイプライン） |
| `episodegen synth` | `04_script.json` をもとに VOICEVOX で音声クリップを合成する |
| `episodegen assemble` | 音声クリップとイントロ・アウトロを ffmpeg で結合し MP3 エピソードを生成する |
| `episodegen manifest` | 番組内容（タイトル・概要・要約・コーナー・コーナー会話要約・記事・会話メモ）を記した `manifest.json` を MP3 と並べて出力する。コーナー記事は `02_rundown.json`（選別済み）から取得する。`--lines` で番組全体要約・会話メモ（`conversation_notes`）・コーナー単位の会話要約を LLM で生成して付加する（`03_lines.json`（元表記）を入力とするため manifest の文字列は英字・漢字のまま出力される）|
| `feedgen` | キャッシュ（`.jsonl`）と `feed-spec.yaml` から RSS 2.0 + iTunes フィード（`feed.xml`）を生成する。manifest・mp3 は不要。エピソード状態は cache を正とする |
| `feedgen check` | `feed-spec.yaml` を strict モードでパースし、必須フィールド・URL/email 形式・プレースホルダを検証する。意味検証エラーは全件まとめて報告する |
| `slackpost` | `manifest.json` と `slack-spec.yaml` を入力に、mp3 を Slack へアップロードして配信する（Slack 配信版）。親メッセージ（mp3 + 初期コメント）とスレッド返信（要約 + コーナー）の 2 段構成 |
| `slackpost check` | `slack-spec.yaml` を strict モードでパースし、必須フィールド（`slack.channel`）を検証する |
| `config check` | 共通設定ファイルを strict モードでパースし、未知キー（typo）や設定値の不整合をエラーとして報告する（パスは `--config` で指定、省略時は `vox-radio.yaml`） |
| `episodegen check` | エピソード仕様 YAML を strict モードでパースし、アセット参照・キャラ参照の整合性を検証する（共通設定は `--config` で指定、省略時は `vox-radio.yaml`） |
| `assets check` | アセット設定 YAML（`assets.yaml`）を strict モードでパースし、typo・参照ファイル欠落・不正値（volume/fade/duck_ratio）をエラーとして報告する |
| `assets preview` | 素材IDを指定し、パラメータを適用したプレビュー音声を MP3 で生成する（`--id {type}:{key} --out out.mp3 [--max-length-sec 秒]`）。loudnorm/alimiter は適用されない |

### 設定ファイルの作成

`vox-radio init` を実行すると、カレントディレクトリに `vox-radio.yaml`（共通設定）・`episode-spec.yaml`（エピソード仕様）・`feed-spec.yaml`（フィード生成設定）・`slack-spec.yaml`（Slack 投稿設定）のテンプレートが生成されます。

```bash
# テンプレートを生成
vox-radio init

# 生成されたファイルを編集（LLM APIキー・番組設定を記入）
# その後、パイプラインを実行
vox-radio episodegen --spec episode-spec.yaml
```

既存ファイルは上書きされません（ファイルごとに独立してスキップ判定します）。

### 設定ファイル

設定は2種類に分かれています。

| 種別 | ファイル | 内容 |
|------|---------|------|
| 共通設定 (config) | `vox-radio.yaml`（デフォルト。`--config` フラグで別パス指定可） | LLM / VOICEVOX URL / キャラカタログ |
| エピソード仕様 (spec) | `episode-spec.yaml` または `examples/<genre>.yaml` | program / corners（各コーナーの source でデータソース指定） / assets |

`vox-radio.yaml` はデフォルトでカレントディレクトリから読み込まれます。別ディレクトリの設定ファイルを使う場合は `--config` フラグでパスを指定します。

```bash
# カレントディレクトリの vox-radio.yaml を使う（デフォルト）
vox-radio episodegen --spec episode-spec.yaml

# 別パスの設定ファイルを指定する
vox-radio --config /path/to/my-station/vox-radio.yaml episodegen --spec episode-spec.yaml
```

#### キャラカタログとスタイル選択

`vox-radio.yaml` の `characters` セクションには、キャラIDごとに複数の音声スタイルを定義できます。`default_style` は `style` 未指定時のフォールバックスタイルです。

```yaml
characters:
  zundamon:
    name: ずんだもん
    pronoun: ボク
    speech_suffix: ["〜のだ", "〜なのだ"]
    personality: ["元気", "明るい"]
    default_style: ノーマル
    styles:
      ノーマル: 3    # style名 → VOICEVOX speaker_id
      あまあま: 1
      なみだめ: 76
```

台本生成（`script` コマンド）では、LLM がセリフの感情に応じてスタイルを選択します。`synth` コマンドは行ごとの `style` フィールドを読み取り、指定されたスタイルの `speaker_id` で合成します。`style` が未指定または不正な場合は `default_style` にフォールバックします。

エピソード仕様のサンプルは `examples/` ディレクトリに用意しています。`examples/tech.yaml`（技術ニュース用）を、共通アセット（`examples/assets/`）とあわせて配置しています。これらをコピー・編集して利用してください。詳細は [examples/README.md](examples/README.md) を参照してください。

### 実行例

```bash
# 記事を収集（--spec は必須）
vox-radio episodegen collect --out work/intermediate/01_articles.json --spec examples/tech.yaml

# 番組設計図（rundown）を生成
vox-radio episodegen rundown --in work/intermediate/01_articles.json --out work/intermediate/02_rundown.json \
    --spec examples/tech.yaml

# 台本を生成
vox-radio episodegen script --in work/intermediate/02_rundown.json --out work/intermediate/04_script.json \
    --spec examples/tech.yaml

# 音声合成（設定不要）
vox-radio episodegen synth --in work/intermediate/04_script.json --out-dir work/clips

# 音声結合
vox-radio episodegen assemble --in work/intermediate/04_script.json --clips work/clips --out work/episode.mp3 \
    --spec examples/tech.yaml
```

### 詳細リファレンス

各コマンドのフラグ一覧は自動生成ドキュメントを参照してください。

- [docs/cli/vox-radio.md](docs/cli/vox-radio.md) — コマンド一覧
- 各サブコマンドの詳細: `vox-radio <command> --help`

## 設定ファイルリファレンス

各設定ファイルの詳細なフィールド定義は `internal/cli/skills/vox-radio/references/` にあります。フィールド定義の正は `internal/config/config.go`（feed は `internal/model/feed_spec.go`）のコードです。

| 設定ファイル | リファレンス | 検証コマンド |
|---|---|---|
| `vox-radio.yaml` | [references/vox-radio.md](internal/cli/skills/vox-radio/references/vox-radio.md) | `vox-radio config check --config <パス>` |
| `episode-spec.yaml` | [references/episode-spec.md](internal/cli/skills/vox-radio/references/episode-spec.md) | `vox-radio episodegen check <パス>` |
| アセット設定 YAML | [references/assets.md](internal/cli/skills/vox-radio/references/assets.md) | `vox-radio assets check <パス>` |
| `feed-spec.yaml` | [references/feed-spec.md](internal/cli/skills/vox-radio/references/feed-spec.md) | `vox-radio feedgen check` |
| `slack-spec.yaml` | [references/slack-spec.md](internal/cli/skills/vox-radio/references/slack-spec.md) | `vox-radio slackpost check` |

> **エージェント向け**: `vox-radio install --skills` を実行すると `.claude/skills/vox-radio/` にスキルファイルをインストールできます。インストール後はローカルの `references/*.md` を参照してください。
