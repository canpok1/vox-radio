# vox-radio

**設定ファイルを元に、ラジオ番組（ポッドキャスト）の音声を自動生成する CLI ツールです。**

記事の収集から台本生成・音声合成・配信用ファイル出力までを一括で行います。

## インストール

最新リリースのインストールスクリプトを実行します。

```bash
curl -fsSL https://github.com/canpok1/vox-radio/releases/latest/download/install.sh | bash
```

設置先の変更・特定バージョンの指定などは[インストールガイド](docs/installation.md)を参照してください。

## クイックスタート

すぐ動くサンプル設定で番組を 1 本作る最短手順です（サンプルの詳細は[設定方法](#設定方法)）。

```bash
# サンプル設定一式を sample/ に生成
vox-radio init --sample

# サンプルでそのまま番組生成
vox-radio --config sample/vox-radio.yaml episodegen --spec sample/episode-spec.yaml
```

**前提条件:**

- **`GEMINI_API_KEY`** — [Google AI Studio](https://aistudio.google.com/) で取得し環境変数に設定（`export GEMINI_API_KEY=<your-key>`）
- **VOICEVOX Engine** — Docker: `docker run -d -p 50021:50021 voicevox/voicevox_engine:cpu-latest`、または [公式アプリ](https://voicevox.hiroshiba.jp/)に同梱（既定 `http://localhost:50021`）

出力先は `output/<YYYYMMDDHHMMSS>/` です。

## 設定方法

設定は 2 種類です。

| 種別 | ファイル | 内容 |
|------|---------|------|
| 共通設定 (config) | `vox-radio.yaml`（既定はカレントディレクトリ。`--config` で別パス指定可） | LLM / VOICEVOX URL / キャラカタログ |
| エピソード仕様 (spec) | `episode-spec.yaml`（`--sample` 生成の `sample/episode-spec.yaml` も同形式） | program / corners（source でデータソース指定）/ アセット参照 |

`vox-radio init --sample` で生成されるサンプル設定は、ずんだもん・めたんが MC を務めるお天気番組（気象庁の防災情報XMLを利用）の記入済み一式です。音声アセットは同梱しないため、効果音・BGM はコメントアウト済みの記入例になっています。コピー・編集して使えます。

### コーディングエージェントで設定する（おすすめ）

Claude Code などのエージェントを使うなら、設定作成をエージェントに任せられます。

```bash
vox-radio install --skills
```

エージェントスキル（`SKILL.md` ＋ フィールド定義 `references/*.md`）が `.claude/skills/vox-radio/` に入ります。あとは「ラジオ番組の設定を作って」と依頼すれば、`init` →リファレンス参照で編集→ `check` 検証まで自動で仕上げます。

### 手動で設定する

`vox-radio init` でテンプレートを生成し（既存ファイルは上書きしません）、各ファイルを編集します。フィールド定義は「設定ファイルリファレンス」を参照してください。

```bash
vox-radio init
```

## 使い方

### パイプラインの実行

```
collect → rundown → script → synth → assemble → manifest
```

| 段 | 概要 |
|----|------|
| collect | コーナーごとにフィード・URL から記事を収集 |
| rundown | LLM が記事を選別し番組設計図を生成 |
| script | 番組設計図から台本を生成（write → direct の多段） |
| synth | VOICEVOX で音声クリップを合成 |
| assemble | クリップとイントロ・アウトロを ffmpeg で結合し MP3 化 |
| manifest | 配信用の番組情報（タイトル・要約・記事など）を JSON 出力 |

`episodegen` で全段を一括実行します。

```bash
# 一括実行
vox-radio episodegen --spec episode-spec.yaml

# 出力とログを1ツリーに集約する場合
vox-radio episodegen --spec episode-spec.yaml --out-dir output --log-dir output/logs
```

各段は個別にも実行できます。

```bash
vox-radio episodegen collect  --out work/01_articles.json --spec sample/episode-spec.yaml
vox-radio episodegen rundown  --in work/01_articles.json --out work/02_rundown.json --spec sample/episode-spec.yaml
vox-radio episodegen script   --in work/02_rundown.json --out work/04_script.json --spec sample/episode-spec.yaml
vox-radio episodegen synth    --in work/04_script.json --out-dir work/clips
vox-radio episodegen assemble --in work/04_script.json --clips work/clips --out work/episode.mp3 --spec sample/episode-spec.yaml
```

ログは既定で `.vox-radio/logs/` に出力されます。

### アセット（音声演出）

ジングル（イントロ/アウトロ）・効果音（SE）・BGM を番組に組み込めます（`assemble` で合成）。

1. `assets/` に音声ファイルを配置
2. `assets/assets.yaml` で `jingle:` / `se:` / `bgm:` を定義（音量・フェード・ダッキング比などを指定）
3. `episode-spec.yaml` の `assets_files` で参照し、コーナー単位で `start_audio` / `end_audio` / `bgm` を割り当て
4. `assets check` で検証、`assets preview` で素材単体を確認

```bash
vox-radio assets check assets/assets.yaml
vox-radio assets preview assets/assets.yaml --id jingle:opening --out preview.mp3
```

各フィールドの詳細は[アセット設定リファレンス](internal/cli/skills/vox-radio/references/assets.md)を参照。

### 過去回の記憶（キャッシュ）と `program.id`

過去回の履歴キャッシュは常に有効です（`vox-radio.yaml` の `cache` で保持件数・日数を調整可、無効化は不可）。実体は `program.id` をキーにした JSONL（`.vox-radio/cache/<program.id>.jsonl`）です。

このため **`program.id` は必須**です。未設定だと `episodegen` / `episodegen check` でエラーになります。回番号・過去回参照・出演回数などが `program.id` 単位で記録されます。

### キャラカタログとスタイル選択

`vox-radio.yaml` の `characters` でキャラごとに複数の音声スタイルを定義できます。`default_style` は `style` 未指定時のフォールバックです。

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

`script` 生成時に LLM が感情に応じてスタイルを選び、`synth` が各行の `style` の `speaker_id` で合成します（未指定・不正時は `default_style`）。

### 配信（feedgen / slackpost）

- **RSS フィード** — `feedgen` がキャッシュ（`.jsonl`）と `feed-spec.yaml` から RSS 2.0 + iTunes フィード（`feed.xml`）を生成（manifest・mp3 不要、状態は cache が正）
- **Slack 配信** — `slackpost` が `manifest.json` と `slack-spec.yaml` で mp3 を Slack へ投稿。親メッセージ＋スレッド返信の 2 段構成で、タイムアウト後も二重投稿なしに再開

各コマンドのフラグは[コマンド一覧](#コマンド一覧)を参照。

## コマンド一覧

コマンド・サブコマンド・フラグの詳細は自動生成ドキュメントを参照してください。

- [docs/cli/vox-radio.md](docs/cli/vox-radio.md) — コマンド一覧
- 各サブコマンド: `vox-radio <command> --help`

## 設定ファイルリファレンス

各設定ファイルのフィールド定義は `internal/cli/skills/vox-radio/references/` にあります（定義の正は `internal/config/config.go`、feed は `internal/model/feed_spec.go`）。

| 設定ファイル | リファレンス | 検証コマンド |
|---|---|---|
| `vox-radio.yaml` | [references/vox-radio.md](internal/cli/skills/vox-radio/references/vox-radio.md) | `vox-radio config check --config <パス>` |
| `episode-spec.yaml` | [references/episode-spec.md](internal/cli/skills/vox-radio/references/episode-spec.md) | `vox-radio episodegen check <パス>` |
| アセット設定 YAML | [references/assets.md](internal/cli/skills/vox-radio/references/assets.md) | `vox-radio assets check <パス>` |
| `feed-spec.yaml` | [references/feed-spec.md](internal/cli/skills/vox-radio/references/feed-spec.md) | `vox-radio feedgen check` |
| `slack-spec.yaml` | [references/slack-spec.md](internal/cli/skills/vox-radio/references/slack-spec.md) | `vox-radio slackpost check` |

> **エージェント向け**: `vox-radio install --skills` でスキルファイルを `.claude/skills/vox-radio/` にインストールでき、以後はローカルの `references/*.md` を参照できます。

## 開発

開発環境・ビルド・テスト・プロンプト評価・リリース検証・アーキテクチャなど、開発者向けの情報は [docs/development/](docs/development/) を参照してください。
