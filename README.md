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

バージョンを埋め込む場合は `VERSION` を指定します。

```bash
make build VERSION=v0.1.0
```

ビルドしたバイナリのバージョンを確認するには `--version` フラグを使います。

```bash
vox-radio --version
```

### パイプライン概要

vox-radio は以下のパイプラインでポッドキャストを自動生成します。

```
collect → rundown → script → synth → assemble → manifest
```

| コマンド | 概要 |
|----------|------|
| `init` | カレントディレクトリに `vox-radio.yaml` と `profile.yaml` のテンプレートを生成する（初回セットアップ用） |
| `collect` | `corners[].source` に定義したフィード・URL からコーナーごとに記事を収集し `01_articles.json` を生成する |
| `rundown` | LLM が収集記事を選別し、コーナーごとの話の流れと要約を含む `02_rundown.json` を生成する（番組設計図） |
| `script` | rundown を LLM に渡して台本 `04_script.json` を生成する（write → direct の多段パイプライン） |
| `synth` | `04_script.json` をもとに VOICEVOX で音声クリップを合成する |
| `assemble` | 音声クリップとイントロ・アウトロを ffmpeg で結合し MP3 エピソードを生成する |
| `manifest` | 番組内容（タイトル・概要・要約・コーナー・コーナー会話要約・記事・会話メモ）を記した `manifest.json` を MP3 と並べて出力する。コーナー記事は `02_rundown.json`（選別済み）から取得する。`--script` で番組全体要約と会話メモ（`conversation_notes`）、`--lines` でコーナー単位の会話要約を LLM で生成して付加する |
| `run` | collect → rundown → script → synth → assemble → manifest の全パイプラインを一括実行する |
| `config check` | `vox-radio.yaml`（共通設定）を strict モードでパースし、未知キー（typo）や設定値の不整合をエラーとして報告する |
| `profile check` | プロファイル YAML を strict モードでパースし、アセット参照・キャラ参照（cwd の `vox-radio.yaml` を使用）の整合性を検証する |

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
vox-radio collect --out work/intermediate/01_articles.json --profile sample-profiles/tech_profile.yaml

# 番組設計図（rundown）を生成
vox-radio rundown --in work/intermediate/01_articles.json --out work/intermediate/02_rundown.json \
    --profile sample-profiles/tech_profile.yaml

# 台本を生成
vox-radio script --in work/intermediate/02_rundown.json --out work/intermediate/04_script.json \
    --profile sample-profiles/tech_profile.yaml

# 音声合成（設定不要）
vox-radio synth --in work/intermediate/04_script.json --out-dir work/clips

# 音声結合
vox-radio assemble --in work/intermediate/04_script.json --clips work/clips --out work/episode.mp3 \
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
| `provider` | string | 任意 | LLM プロバイダ。`openai`（デフォルト）または `dify-chat` |
| `temperature` | float64 | 任意 | 生成のランダム性（0.0〜1.0）。デフォルト: 0（Go ゼロ値） |
| `max_retries` | int | 任意 | APIリトライ回数。デフォルト: 0（Go ゼロ値） |
| `min_request_interval_ms` | *int | 任意 | リクエスト間隔（ミリ秒）。省略時は 4500ms |
| `steps` | map[string]LLMStepConfig | 任意 | ステップごとの設定（キー: ステップ名） |
| `openai` | OpenAIConfig | `provider: openai` 時必須 | OpenAI 互換プロバイダの接続設定 |
| `dify-chat` | DifyChatConfig | `provider: dify-chat` 時必須 | Dify chat-messages の接続設定 |

##### `llm.openai` サブフィールド（`provider: openai` 時）

| フィールド | 型 | 必須/任意 | 説明 |
|---|---|---|---|
| `base_url` | string | 必須 | LLM API のベースURL（OpenAI 互換エンドポイント） |
| `api_key_env` | string | 必須 | APIキーを格納する環境変数名 |
| `model` | string | 必須 | 使用するモデル名 |

##### `llm.dify-chat` サブフィールド（`provider: dify-chat` 時）

| フィールド | 型 | 必須/任意 | 説明 |
|---|---|---|---|
| `base_url` | string | 必須 | Dify API サーバーURL（例: `https://api.dify.ai/v1`） |
| `api_key_env` | string | 必須 | Dify API キーを格納する環境変数名 |
| `user` | string | 任意 | 利用者識別子。省略時は `vox-radio` |
| `inputs` | map[string]string | 任意 | Dify アプリに渡す変数。値に `${temperature}` プレースホルダーを書ける |

`inputs` の `${temperature}` プレースホルダーについて:
- 値が `"${temperature}"` だけの場合（完全一致）→ そのステップの temperature を **JSON 数値**で送信
- 値に `${temperature}` が含まれる場合（部分一致）→ 文字列として補間
- プレースホルダーを書かない場合 → temperature を inputs に含めない

> **注意**: inputs に temperature を載せても、Dify アプリ側でその変数をモデルパラメータにバインドしない限り効果はありません。

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
| `length_sec` | int | 任意 | 番組全体の目標収録時間（秒）。デフォルト: 0（Go ゼロ値） |

#### `corners` セクション

`corners` はコーナー定義のリストです。番組を構成するセグメントを順番に記述します。

| フィールド | 型 | 必須/任意 | 説明 |
|---|---|---|---|
| `title` | string | 必須 | コーナータイトル |
| `content` | string | 任意 | コーナーの内容説明（台本生成 LLM への指示に使用） |
| `direction` | string | 任意 | コーナーの演出説明（演出生成 LLM への指示に使用。SE の挿入タイミングなど）。台本生成 LLM へは渡されない |
| `cast` | map[string]string | 任意 | キャラID → 役割説明のマップ（キーは `vox-radio.yaml` の `characters` のキーと一致させること） |
| `length_sec` | int | 任意 | このコーナーの目標収録時間（秒）。台本生成時に文字数（≈7文字/秒）へ換算される |
| `source` | SourceConfig | 任意 | データソース（省略するとこのコーナーの収集はスキップ） |
| `start_jingle` | string | 任意 | コーナー開始ジングルのキー名（`assets.jingle` のキーと一致させること）。コーナー本編の前に確定的に挿入される |
| `end_jingle` | string | 任意 | コーナー終了ジングルのキー名（`assets.jingle` のキーと一致させること）。コーナー本編の後に確定的に挿入される |
| `bgm` | string | 任意 | コーナー中 BGM のキー名（`assets.bgm` のキーと一致させること）。コーナー本編を開始/停止セグメントで挟む |

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

音声アセットは `script.json` のセグメント型として統一的に表現されます。各セグメントは `type` フィールドで種別を指定し、`asset_name` フィールドで対応するマップのキーを参照します。

| セグメント種別 | `type` 値 | 再生方式 | 説明 |
|---|---|---|---|
| 音声（ナレーション） | `speech` | serial | メイン音声。複数同時不可 |
| 効果音 | `se` | overlay | 音声に重ねて再生。複数同時可 |
| BGM | `bgm` | overlay | 音声の裏で再生。排他（停止→切替）。`asset_name` 空 = 停止 |
| ジングル | `jingle` | serial | 単独再生（音声・BGMと重ならない）。前後に pause が入る |

ジングルはラン境界として機能します: 台本がジングルで区切られ、各ラン内の SE/BGM はそのランにのみ適用されます（ジングルをまたいで継続しません）。

##### `assets.jingle` マップ値

| フィールド | 型 | 必須/任意 | 説明 |
|---|---|---|---|
| `file` | string | 必須 | 音声ファイルパス |
| `fade_in` | float64 | 任意 | フェードイン時間（秒）。デフォルト: 0 |
| `fade_out` | float64 | 任意 | フェードアウト時間（秒）。デフォルト: 0 |
| `description` | string | 任意 | アセットの説明（「何の音か・いつ使うか」）。LLM が挿入タイミングを判断する際の手がかりになる |

ジングルはコーナー毎に `corners[].start_jingle` / `corners[].end_jingle` で設定します。script 生成ステップでコードがコーナー本編の前後へ確定的に挿入するため、生成された `04_script.json` にジングルセグメントが含まれます。BGM も `corners[].bgm` で同様にコーナー単位で管理します。

##### `assets.se` マップ値

| フィールド | 型 | 必須/任意 | 説明 |
|---|---|---|---|
| `file` | string | 必須 | 音声ファイルパス |
| `volume` | float64 | 任意 | 音量倍率。デフォルト: 0（Go ゼロ値） |
| `description` | string | 任意 | アセットの説明（「何の音か・いつ使うか」）。LLM が挿入タイミングを判断する際の手がかりになる |

##### `assets.bgm` マップ値

| フィールド | 型 | 必須/任意 | 説明 |
|---|---|---|---|
| `file` | string | 必須 | 音声ファイルパス |
| `volume` | float64 | 任意 | 音量倍率。デフォルト: 0（Go ゼロ値） |
| `duck_ratio` | float64 | 任意 | セリフ再生中の音量低減比率（サイドチェインコンプ）。デフォルト: 0 |
| `loop` | bool | 任意 | ループ再生するかどうか。デフォルト: false |
| `description` | string | 任意 | アセットの説明（「何の音か・いつ使うか」）。LLM が挿入タイミングを判断する際の手がかりになる |

BGM の開始・停止は台本の `bgm` セグメントで制御します。`asset_name` にキー名を指定するとその BGM を開始し、空文字列を指定すると停止します。
