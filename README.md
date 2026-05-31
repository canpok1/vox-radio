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

## CLIの使い方

### ビルド

```bash
make build
```

### パイプライン概要

vox-radio は以下のパイプラインでポッドキャストを自動生成します。

```
collect → script → synth → assemble → manifest
```

| コマンド | 概要 |
|----------|------|
| `init` | カレントディレクトリに `vox-radio.yaml` と `profile.yaml` のテンプレートを生成する（初回セットアップ用） |
| `collect` | `corners[].source` に定義したフィード・URL からコーナーごとに記事を収集し `articles.json` を生成する |
| `script` | 記事を LLM に渡して台本 `script.json` を生成する（summarize → write → direct の多段パイプライン） |
| `synth` | `script.json` をもとに VOICEVOX で音声クリップを合成する |
| `assemble` | 音声クリップとイントロ・アウトロを ffmpeg で結合し MP3 エピソードを生成する |
| `manifest` | 番組内容（タイトル・概要・要約・コーナー・記事）を記した `manifest.json` を MP3 と並べて出力する。`--script` を指定すると LLM で台本ベースの要約を生成する |
| `run` | collect → script → synth → assemble → manifest の全パイプラインを一括実行する |

### 設定ファイルの作成

`vox-radio init` を実行すると、カレントディレクトリに `vox-radio.yaml`（共通設定）と `profile.yaml`（プロファイル）のテンプレートが生成されます。

```bash
# テンプレートを生成
vox-radio init

# 生成されたファイルを編集（LLM APIキー・番組設定を記入）
# その後、パイプラインを実行
vox-radio run --profile profile.yaml
```

既存ファイルは上書きされません（ファイルごとに独立してスキップ判定します）。

### 設定ファイル

設定は2種類に分かれています。

| 種別 | ファイル | 内容 |
|------|---------|------|
| 共通設定 (config) | `vox-radio.yaml`（カレントディレクトリ、自動読込） | LLM / VOICEVOX URL / キャラカタログ |
| ジャンル別設定 (profile) | `profile.yaml` または `sample-profiles/<genre>_profile.yaml` | program / corners（各コーナーの source でデータソース指定） / assets |

`vox-radio.yaml` はカレントディレクトリから自動的に読み込まれます（`--config` フラグは不要）。

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

プロファイルのサンプルは `sample-profiles/` ディレクトリに用意しています。`sample-profiles/tech_profile.yaml`（技術ニュース用）を、共通アセット（`sample-profiles/assets/`）とあわせて配置しています。これらをコピー・編集して利用してください。詳細は [sample-profiles/README.md](sample-profiles/README.md) を参照してください。

### 実行例

```bash
# 記事を収集（--profile は必須）
vox-radio collect --out work/articles.json --profile sample-profiles/tech_profile.yaml

# 台本を生成
vox-radio script --in work/articles.json --out work/script.json \
    --profile sample-profiles/tech_profile.yaml

# 音声合成（設定不要）
vox-radio synth --in work/script.json --out-dir work/clips

# 音声結合
vox-radio assemble --in work/script.json --clips work/clips --out work/episode.mp3 \
    --profile sample-profiles/tech_profile.yaml
```

### 詳細リファレンス

各コマンドのフラグ一覧は自動生成ドキュメントを参照してください。

- [docs/cli/vox-radio.md](docs/cli/vox-radio.md) — コマンド一覧
- 各サブコマンドの詳細: `vox-radio <command> --help`

## 設定ファイルリファレンス

### vox-radio.yaml（共通設定）

`vox-radio.yaml` はカレントディレクトリから自動読み込みされます。フィールド定義は `internal/config/config.go` の構造体が正です。

#### `llm` セクション

| フィールド | 型 | 必須/任意 | 説明 |
|---|---|---|---|
| `base_url` | string | 必須 | LLM API のベースURL（OpenAI 互換エンドポイント） |
| `api_key_env` | string | 必須 | APIキーを格納する環境変数名 |
| `model` | string | 必須 | 使用するモデル名 |
| `temperature` | float64 | 任意 | 生成のランダム性（0.0〜1.0）。デフォルト: 0（Go ゼロ値） |
| `max_retries` | int | 任意 | APIリトライ回数。デフォルト: 0（Go ゼロ値） |
| `steps` | map[string]LLMStepConfig | 任意 | ステップごとの設定（キー: ステップ名） |

##### `llm.steps.<step>` サブフィールド

| フィールド | 型 | 必須/任意 | 説明 |
|---|---|---|---|
| `temperature` | *float64 | 任意 | このステップの温度（省略時は `llm.temperature` を使用） |

組み込みステップ名: `summarize`（記事要約）、`plan`（台本設計）、`write`（セリフ執筆）、`direct`（ダイレクト生成）。

#### `voicevox` セクション

| フィールド | 型 | 必須/任意 | 説明 |
|---|---|---|---|
| `url` | string | 必須 | VOICEVOX Engine のURL |

#### `characters` セクション

`characters` はキャラID（文字列キー）をキーにしたマップです。プロファイルの `corners[].cast` で使用するIDを定義します。

| フィールド | 型 | 必須/任意 | 説明 |
|---|---|---|---|
| `name` | string | 必須 | キャラクターの表示名 |
| `pronoun` | string | 任意 | 一人称代名詞（台本生成時に LLM へ渡す） |
| `speech_suffix` | []string | 任意 | 語尾パターン（台本生成時に LLM へ渡す） |
| `personality` | []string | 任意 | 性格特徴（台本生成時に LLM へ渡す） |
| `default_style` | string | 任意 | デフォルトの音声スタイル名（`styles` のキーと一致させること） |
| `styles` | map[string]int | 任意 | スタイル名 → VOICEVOX 話者ID のマップ |

`default_style` を指定した場合、その値は `styles` のキーとして存在しなければなりません（起動時検証あり）。

---

### profile.yaml（プロファイル）

`--profile` フラグで指定するジャンル別設定ファイルです。`vox-radio init` で生成されるテンプレートは `profile.yaml` という名前です。詳細は [sample-profiles/README.md](sample-profiles/README.md) も参照してください。

#### `program` セクション

| フィールド | 型 | 必須/任意 | 説明 |
|---|---|---|---|
| `title` | string | 任意 | 番組タイトル |
| `description` | string | 任意 | 番組の説明（LLM への指示に使用） |
| `segment_pause_sec` | float64 | 任意 | コーナー間の無音時間（秒）。デフォルト: 0（Go ゼロ値） |
| `target_duration_sec` | int | 任意 | 番組全体の目標収録時間（秒）。デフォルト: 0（Go ゼロ値） |

#### `corners` セクション

`corners` はコーナー定義のリストです。番組を構成するセグメントを順番に記述します。

| フィールド | 型 | 必須/任意 | 説明 |
|---|---|---|---|
| `title` | string | 必須 | コーナータイトル |
| `content` | string | 任意 | コーナーの内容説明（LLM への指示に使用） |
| `cast` | map[string]string | 任意 | キャラID → 役割説明のマップ（キーは `vox-radio.yaml` の `characters` のキーと一致させること） |
| `target_duration_sec` | int | 任意 | このコーナーの目標収録時間（秒）。台本生成時に文字数（≈7文字/秒）へ換算される |
| `source` | SourceConfig | 任意 | データソース（省略するとこのコーナーの収集はスキップ） |

##### `corners[].source` サブフィールド

| フィールド | 型 | 必須/任意 | 説明 |
|---|---|---|---|
| `feeds` | []FeedEntry | 任意 | RSS/Atom フィードのリスト |
| `articles` | []string | 任意 | 個別記事 URL のリスト |

##### `corners[].source.feeds[]` サブフィールド

| フィールド | 型 | 必須/任意 | 説明 |
|---|---|---|---|
| `url` | string | 必須 | フィードの URL |
| `max_items` | int | 任意 | 取得する最大記事数。デフォルト: 0（Go ゼロ値、実質無制限） |

#### `assets` セクション

`assets` はジングル・SE・BGM の音声素材を設定します。バイナリ素材は別途用意してください。ファイルパスはプロファイルファイルのディレクトリからの相対パスで解決されます。

##### `assets.jingle` マップ値

| フィールド | 型 | 必須/任意 | 説明 |
|---|---|---|---|
| `file` | string | 必須 | 音声ファイルパス |
| `fade_in` | float64 | 任意 | フェードイン時間（秒）。デフォルト: 0 |
| `fade_out` | float64 | 任意 | フェードアウト時間（秒）。デフォルト: 0 |

##### `assets.se` マップ値

| フィールド | 型 | 必須/任意 | 説明 |
|---|---|---|---|
| `file` | string | 必須 | 音声ファイルパス |
| `volume` | float64 | 任意 | 音量倍率。デフォルト: 0（Go ゼロ値） |

##### `assets.bgm` マップ値

| フィールド | 型 | 必須/任意 | 説明 |
|---|---|---|---|
| `file` | string | 必須 | 音声ファイルパス |
| `volume` | float64 | 任意 | 音量倍率。デフォルト: 0（Go ゼロ値） |
| `duck_ratio` | float64 | 任意 | セリフ再生中の音量低減比率。デフォルト: 0 |
| `loop` | bool | 任意 | ループ再生するかどうか。デフォルト: false |
