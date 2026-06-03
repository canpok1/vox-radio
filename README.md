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

`sample-profiles/tech_profile.yaml` を使ってパイプライン全体を試すには `make run-sample` を実行します。

```bash
make run-sample
```

出力先は `output/<YYYYMMDDHHMMSS>/` ディレクトリになります（例: `output/20260601053357/episode.mp3`）。

プロファイルや出力先を変更する場合は `PROFILE` / `OUT_DIR` 変数で上書きできます。

```bash
# 別のプロファイルを使う
make run-sample PROFILE=sample-profiles/other_profile.yaml

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
| `init` | カレントディレクトリに `vox-radio.yaml`・`profile.yaml`・`feedgen.yaml` のテンプレートを生成する（初回セットアップ用） |
| `episodegen` | collect → rundown → script → synth → assemble → manifest の全パイプラインを一括実行し 1 本のエピソードを生成する |
| `episodegen collect` | `corners[].source` に定義したフィード・URL からコーナーごとに記事を収集し `01_articles.json` を生成する |
| `episodegen rundown` | LLM が収集記事を選別し、コーナーごとの話の流れと要約を含む `02_rundown.json` を生成する（番組設計図） |
| `episodegen script` | rundown を LLM に渡して台本 `04_script.json` を生成する（write → direct の多段パイプライン） |
| `episodegen synth` | `04_script.json` をもとに VOICEVOX で音声クリップを合成する |
| `episodegen assemble` | 音声クリップとイントロ・アウトロを ffmpeg で結合し MP3 エピソードを生成する |
| `episodegen manifest` | 番組内容（タイトル・概要・要約・コーナー・コーナー会話要約・記事・会話メモ）を記した `manifest.json` を MP3 と並べて出力する。コーナー記事は `02_rundown.json`（選別済み）から取得する。`--lines` で番組全体要約・会話メモ（`conversation_notes`）・コーナー単位の会話要約を LLM で生成して付加する（`03_lines.json`（元表記）を入力とするため manifest の文字列は英字・漢字のまま出力される）|
| `feedgen` | キャッシュ（`.jsonl`）と `feedgen.yaml` から RSS 2.0 + iTunes フィード（`feed.xml`）を生成する。manifest・mp3 は不要。エピソード状態は cache を正とする |
| `config check` | `vox-radio.yaml`（共通設定）を strict モードでパースし、未知キー（typo）や設定値の不整合をエラーとして報告する |
| `profile check` | プロファイル YAML を strict モードでパースし、アセット参照・キャラ参照（cwd の `vox-radio.yaml` を使用）の整合性を検証する |

### 設定ファイルの作成

`vox-radio init` を実行すると、カレントディレクトリに `vox-radio.yaml`（共通設定）・`profile.yaml`（プロファイル）・`feedgen.yaml`（フィード生成設定）のテンプレートが生成されます。

```bash
# テンプレートを生成
vox-radio init

# 生成されたファイルを編集（LLM APIキー・番組設定を記入）
# その後、パイプラインを実行
vox-radio episodegen --profile profile.yaml
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
vox-radio episodegen collect --out work/intermediate/01_articles.json --profile sample-profiles/tech_profile.yaml

# 番組設計図（rundown）を生成
vox-radio episodegen rundown --in work/intermediate/01_articles.json --out work/intermediate/02_rundown.json \
    --profile sample-profiles/tech_profile.yaml

# 台本を生成
vox-radio episodegen script --in work/intermediate/02_rundown.json --out work/intermediate/04_script.json \
    --profile sample-profiles/tech_profile.yaml

# 音声合成（設定不要）
vox-radio episodegen synth --in work/intermediate/04_script.json --out-dir work/clips

# 音声結合
vox-radio episodegen assemble --in work/intermediate/04_script.json --clips work/clips --out work/episode.mp3 \
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
| `presets` | *VoicevoxPresets | 任意 | 抑揚・音高・話速プリセット定義。省略時はコード組込みのデフォルトプリセットが適用される |

##### `voicevox.presets` サブフィールド

| フィールド | 型 | 必須/任意 | 説明 |
|---|---|---|---|
| `intonation` | map[string]float64 | 任意 | 抑揚プリセット（intonationScale, 0.0〜2.0）。省略時はデフォルト7段階が適用される |
| `pitch` | map[string]float64 | 任意 | 音高プリセット（pitchScale, -0.15〜0.15）。省略時はデフォルト7段階が適用される |
| `speed` | map[string]float64 | 任意 | 話速プリセット（speedScale, 0.5〜2.0）。省略時はデフォルト7段階が適用される |

#### `cache` セクション

| フィールド | 型 | 必須/任意 | 説明 |
|---|---|---|---|
| `enabled` | bool | 任意 | キャッシュ機能の有効/無効。デフォルト: false。`program.id` 未設定時は常に無効 |
| `max_entries` | int | 任意 | JSONL に保持する最大エピソード数（超過分は古い行から削除）。デフォルト: 100 |
| `retention_days` | int | 任意 | 保持日数（超過した古い行は削除）。デフォルト: 90 |
| `llm_context_entries` | int | 任意 | LLM へ渡す直近エピソード件数。デフォルト: 10 |

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
| `summary_length` | int | 任意 | 番組全体サマリーの目安文字数。未指定時はデフォルト 200 文字 |

#### `corners` セクション

`corners` はコーナー定義のリストです。番組を構成するセグメントを順番に記述します。

| フィールド | 型 | 必須/任意 | 説明 |
|---|---|---|---|
| `title` | string | 必須 | コーナータイトル |
| `content` | string | 任意 | コーナーの内容説明（台本生成 LLM への指示に使用） |
| `direction` | string | 任意 | コーナーの演出説明（演出生成 LLM への指示に使用。SE の挿入タイミングなど）。台本生成 LLM へは渡されない |
| `cast` | map[string]string | 任意 | キャラID → 役割説明のマップ（キーは `vox-radio.yaml` の `characters` のキーと一致させること） |
| `length_sec` | int | 任意 | このコーナーの目標収録時間（秒）。台本生成時に文字数（≈7文字/秒）へ換算される |
| `summary_length` | int | 任意 | コーナーサマリーの目安文字数。未指定時はデフォルト 100 文字 |
| `source` | SourceConfig | 任意 | データソース（省略するとこのコーナーの収集はスキップ） |
| `start_jingle` | string | 任意 | コーナー開始ジングルのキー名（`assets.jingle` のキーと一致させること）。コーナー本編の前に確定的に挿入される |
| `end_jingle` | string | 任意 | コーナー終了ジングルのキー名（`assets.jingle` のキーと一致させること）。コーナー本編の後に確定的に挿入される |
| `bgm` | string | 任意 | コーナー中 BGM のキー名（`assets.bgm` のキーと一致させること）。コーナー本編を開始/停止セグメントで挟む |
| `start_pause_sec` | float64 | 任意 | コーナー先頭（`start_jingle` より前）に挿入する無音時間（秒）。0 または省略時は挿入しない |
| `end_pause_sec` | float64 | 任意 | コーナー末尾（`end_jingle` より後）に挿入する無音時間（秒）。0 または省略時は挿入しない |

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

#### `assets_files` フィールド

`assets_files` はアセット設定ファイル（ジングル・SE・BGM を定義した YAML）のパスリストです。バイナリ素材は別途用意してください。

- `assets_files` の各パスは**プロファイルファイルのディレクトリ**を基準に解決されます
- アセット設定ファイル内の `file:` 相対パスは**そのアセット設定ファイルのディレクトリ**を基準に解決されます（アセット設定ファイルと音声素材をひとまとめに配布できます）
- 複数ファイルを指定した場合は後勝ちでマージされます（共通アセット集＋番組固有アセットの組み合わせが可能）
- `assets_files` を省略した場合はアセットが空となります（アセット不要なプロファイルで許容）

```yaml
# プロファイルファイルからの参照
assets_files:
  - assets/assets.yaml          # 共通アセット集
  - assets/my-assets.yaml       # 番組固有アセット（後勝ちでマージ）
```

アセット設定ファイルのトップレベルには `jingle:` / `se:` / `bgm:` を記述します。

音声アセットは `script.json` のセグメント型として統一的に表現されます。各セグメントは `type` フィールドで種別を指定し、`asset_name` フィールドで対応するマップのキーを参照します。

| セグメント種別 | `type` 値 | 再生方式 | 説明 |
|---|---|---|---|
| 音声（ナレーション） | `speech` | serial | メイン音声。複数同時不可 |
| 効果音 | `se` | serial（既定）/ overlay（`overlay: true` 指定時） | 既定は SE が鳴り終わってから次のセリフを再生（順次）。`overlay: true` を設定すると音声に重ねて再生 |
| BGM | `bgm` | overlay | 音声の裏で再生。排他（停止→切替）。`asset_name` 空 = 停止 |
| ジングル | `jingle` | serial | 単独再生（音声・BGMと重ならない）。前後に pause が入る |

ジングルはラン境界として機能します: 台本がジングルで区切られ、各ラン内の SE/BGM はそのランにのみ適用されます（ジングルをまたいで継続しません）。

##### アセット設定ファイル: `jingle` マップ値

| フィールド | 型 | 必須/任意 | 説明 |
|---|---|---|---|
| `file` | string | 必須 | 音声ファイルパス |
| `fade_in` | float64 | 任意 | フェードイン時間（秒）。デフォルト: 0 |
| `fade_out` | float64 | 任意 | フェードアウト時間（秒）。デフォルト: 0 |
| `description` | string | 任意 | アセットの説明（「何の音か・いつ使うか」）。LLM が挿入タイミングを判断する際の手がかりになる |

ジングルはコーナー毎に `corners[].start_jingle` / `corners[].end_jingle` で設定します。script 生成ステップでコードがコーナー本編の前後へ確定的に挿入するため、生成された `04_script.json` にジングルセグメントが含まれます。BGM も `corners[].bgm` で同様にコーナー単位で管理します。

##### アセット設定ファイル: `se` マップ値

| フィールド | 型 | 必須/任意 | 説明 |
|---|---|---|---|
| `file` | string | 必須 | 音声ファイルパス |
| `volume` | float64 | 任意 | 音量倍率。デフォルト: 0（Go ゼロ値） |
| `trim_silence` | bool | 任意 | 前後の無音を自動除去するかどうか。デフォルト: true |
| `overlay` | bool | 任意 | `true` = 音声に重ねて再生（従来の overlay 動作）。`false` または省略 = SE が鳴り終わってから次のセリフを再生（順次）。デフォルト: false |
| `description` | string | 任意 | アセットの説明（「何の音か・いつ使うか」）。LLM が挿入タイミングを判断する際の手がかりになる |

##### アセット設定ファイル: `bgm` マップ値

| フィールド | 型 | 必須/任意 | 説明 |
|---|---|---|---|
| `file` | string | 必須 | 音声ファイルパス |
| `volume` | float64 | 任意 | 音量倍率。デフォルト: 0（Go ゼロ値） |
| `duck_ratio` | float64 | 任意 | セリフ再生中の音量低減比率（サイドチェインコンプ）。デフォルト: 0 |
| `loop` | bool | 任意 | ループ再生するかどうか。デフォルト: false |
| `fade_in` | float64 | 任意 | BGM 開始時のフェードイン秒数。省略時は 1.0 秒。`0` を指定するとフェードなし |
| `fade_out` | float64 | 任意 | BGM 終了時のフェードアウト秒数。省略時は 1.0 秒。`0` を指定するとフェードなし |
| `description` | string | 任意 | アセットの説明（「何の音か・いつ使うか」）。LLM が挿入タイミングを判断する際の手がかりになる |

BGM の開始・停止は台本の `bgm` セグメントで制御します。`asset_name` にキー名を指定するとその BGM を開始し、空文字列を指定すると停止します。

同一ラン内で BGM が別の BGM に切り替わる場合、前の BGM がフェードアウトしつつ次の BGM がフェードインするクロスフェードが自動で適用されます（重なり幅 = `min(prevFadeOut, nextFadeIn)`）。ジングル境界または BGM 明示停止時も `fade_out` 秒でフェードアウトします。
