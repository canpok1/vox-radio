# vox-radio

**設定ファイルを元に、ラジオ番組（ポッドキャスト）の音声を自動生成する CLI ツールです。**

記事の収集から台本生成・音声合成・配信用ファイルの出力までを、設定ファイルに沿って一括で行います。

## インストール

最新リリースのインストールスクリプトを実行します。

```bash
# 最新版をインストール
curl -fsSL https://github.com/canpok1/vox-radio/releases/latest/download/install.sh | bash

# 設置先を変更する場合は環境変数 INSTALL_DIR を指定
curl -fsSL https://github.com/canpok1/vox-radio/releases/latest/download/install.sh | INSTALL_DIR=$HOME/.local/bin bash
```

特定のバージョンを入れたい場合は、URL の `latest/download` をリリースタグに置き換えます（例: `releases/download/v0.0.16/install.sh`）。利用可能なバージョンは [GitHub Releases](https://github.com/canpok1/vox-radio/releases) で確認できます。

デフォルトの設置先は `/usr/local/bin` です。書き込み権限がない場合は自動で `sudo` にフォールバックします。

## クイックスタート

すぐ動くサンプル設定一式を使って、番組を 1 本作る最短手順です。`vox-radio init --sample` で、ずんだもん・めたんが MC を務めるお天気番組（出典明記のうえで翻案・再配信が可能な気象庁の防災情報XMLフィードを利用）の記入済み設定が `sample/` に生成されます。

```bash
# すぐ動くサンプル一式を sample/ に生成
vox-radio init --sample

# サンプルでそのまま番組生成を試す
vox-radio --config sample/vox-radio.yaml episodegen --spec sample/episode-spec.yaml
```

- **前提条件:** `GEMINI_API_KEY` 環境変数と VOICEVOX Engine が必要です。
- 出力先は `output/<YYYYMMDDHHMMSS>/` ディレクトリになります（例: `output/20260601053357/episode.mp3`）。
- 音声アセットは同梱しないため、サンプルの効果音・BGM 設定はコメントアウトした記入例として入っています。

## 設定方法

設定は 2 種類に分かれています。

| 種別 | ファイル | 内容 |
|------|---------|------|
| 共通設定 (config) | `vox-radio.yaml`（デフォルト。`--config` フラグで別パス指定可） | LLM / VOICEVOX URL / キャラカタログ |
| エピソード仕様 (spec) | `episode-spec.yaml`（`init --sample` で生成される `sample/episode-spec.yaml` も同形式） | program / corners（各コーナーの source でデータソース指定） / アセット参照 |

`vox-radio.yaml` はデフォルトでカレントディレクトリから読み込まれます。別ディレクトリの設定ファイルを使う場合は `--config` フラグでパスを指定します。

VOICEVOX エンジンの URL は環境変数 `VOX_RADIO_VOICEVOX_URL` で上書きできます（優先順位は 環境変数 > `voicevox.url` > デフォルト `http://localhost:50021`）。devcontainer では VOICEVOX が別サービスとして起動するため、この環境変数が自動設定されており、サンプル設定そのままで音声生成できます。

設定の作り方には 2 通りあります。

### コーディングエージェントで設定する（おすすめ）

Claude Code などのコーディングエージェントを使っているなら、設定ファイルの作成・修正をエージェントに任せられます。`install --skills` で vox-radio のエージェントスキル（`SKILL.md` ＋ 各設定ファイルのフィールド定義 `references/*.md`）を `.claude/skills/vox-radio/` にインストールします。

```bash
vox-radio install --skills
```

あとはエージェントに「ラジオ番組の設定を作って」のように依頼すると、スキルが `vox-radio init` でテンプレートを生成し、同梱リファレンスを参照しながら YAML を編集し、`check` コマンドで検証が通るまで仕上げます。フィールド定義を自分で読み込む必要がないぶん手軽です。

### 手動で設定する

```bash
# テンプレートを生成
vox-radio init

# 生成されたファイルを編集（LLM APIキー・番組設定を記入）
# 各ファイルのフィールド定義は下記「設定ファイルリファレンス」を参照
# その後、パイプラインを実行
vox-radio episodegen --spec episode-spec.yaml
```

`vox-radio init` を実行すると、カレントディレクトリに `vox-radio.yaml`（共通設定）・`episode-spec.yaml`（エピソード仕様）・`feed-spec.yaml`（フィード生成設定）・`slack-spec.yaml`（Slack 投稿設定）・`assets/assets.yaml`（アセット設定）のテンプレートが生成されます。既存ファイルは上書きされません（ファイルごとに独立してスキップ判定します）。

## 使い方

### パイプラインの実行

vox-radio は以下のパイプラインでポッドキャストを自動生成します。

```
collect → rundown → script → synth → assemble → manifest
```

| 段 | 概要 |
|----|------|
| collect | コーナーごとにフィード・URL から記事を収集する |
| rundown | LLM が記事を選別し番組設計図（rundown）を生成する |
| script | rundown から台本を生成する（write → direct の多段パイプライン） |
| synth | VOICEVOX で音声クリップを合成する |
| assemble | 音声クリップとイントロ・アウトロを ffmpeg で結合し MP3 を生成する |
| manifest | 配信用の番組情報（タイトル・要約・コーナー・記事など）を JSON で出力する |

`episodegen` で全パイプラインを一括実行できます。

```bash
# カレントディレクトリの vox-radio.yaml を使う（デフォルト）
vox-radio episodegen --spec episode-spec.yaml

# 別パスの設定ファイルを指定する
vox-radio --config /path/to/my-station/vox-radio.yaml episodegen --spec episode-spec.yaml

# 成果物とログを同一ツリーへ集約する
vox-radio episodegen --out-dir output --log-dir output/logs --spec episode-spec.yaml
```

ログはデフォルトで `.vox-radio/logs/` に出力されます（`--log-dir` フラグで変更可能）。

各ステップを個別に実行することもできます。

```bash
# 記事を収集（--spec は必須）
vox-radio episodegen collect --out work/intermediate/01_articles.json --spec sample/episode-spec.yaml

# 番組設計図（rundown）を生成
vox-radio episodegen rundown --in work/intermediate/01_articles.json --out work/intermediate/02_rundown.json \
    --spec sample/episode-spec.yaml

# 台本を生成
vox-radio episodegen script --in work/intermediate/02_rundown.json --out work/intermediate/04_script.json \
    --spec sample/episode-spec.yaml

# 音声合成（設定不要）
vox-radio episodegen synth --in work/intermediate/04_script.json --out-dir work/clips

# 音声結合
vox-radio episodegen assemble --in work/intermediate/04_script.json --clips work/clips --out work/episode.mp3 \
    --spec sample/episode-spec.yaml
```

### アセット（音声演出）

ジングル（イントロ/アウトロ）・効果音（SE）・BGM といった音声アセットを番組に組み込めます。アセットは `assemble` ステップで音声に合成されます。

おおまかな流れは次のとおりです。

1. `assets/` ディレクトリに音声ファイルを配置する
2. `assets/assets.yaml` に `jingle:` / `se:` / `bgm:` ごとの素材を定義する（音量・フェード・ダッキング比などのパラメータを指定）
3. `episode-spec.yaml` の `assets_files` でアセット設定ファイルを参照し、コーナー単位で `start_audio` / `end_audio` / `bgm` を割り当てる
4. `assets check` で設定を検証し、`assets preview` で素材単体の鳴り方を確認する

```bash
# アセット設定を検証する
vox-radio assets check assets/assets.yaml

# 素材IDを指定してプレビュー音声を生成する（type は jingle/se/bgm）
vox-radio assets preview assets/assets.yaml --id jingle:opening --out preview.mp3
```

各フィールドの詳細は[アセット設定リファレンス](internal/cli/skills/vox-radio/references/assets.md)を参照してください。

### 過去回の記憶（キャッシュ）と `program.id`

過去回の記憶（エピソード履歴キャッシュ）は常に有効です。`vox-radio.yaml` の `cache` セクションでは保持件数・保持日数などを調整できますが、有効/無効の切り替えはできません。キャッシュの実体は `episode-spec.yaml` の `program.id` をキーにした JSONL ファイル（`.vox-radio/cache/<program.id>.jsonl`）です。

このため **`program.id` は必須** です。未設定の場合は `episodegen check` および `episodegen`（番組生成）でバリデーションエラーになります。回番号・過去回参照・出演回数などは `program.id` をキーに記録されます。

### キャラカタログとスタイル選択

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

### 配信（feedgen / slackpost）

生成した番組は次の 2 通りで配信できます。

- **RSS フィード**: `feedgen` がキャッシュ（`.jsonl`）と `feed-spec.yaml` から RSS 2.0 + iTunes フィード（`feed.xml`）を生成します。manifest・mp3 は不要で、エピソード状態は cache を正とします。
- **Slack 配信**: `slackpost` が `manifest.json` と `slack-spec.yaml` を入力に、mp3 を Slack へアップロードします。親メッセージ（mp3 + 初期コメント）とスレッド返信（要約 + コーナー）の 2 段構成で、タイムアウト後の再実行でも音声の二重投稿なしに返信のみ再開できます。

各コマンドのフラグは[コマンド一覧](#コマンド一覧)から参照してください。

## コマンド一覧

各コマンド・サブコマンドの一覧とフラグの詳細は、自動生成ドキュメントを参照してください。

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

## 開発

開発環境のセットアップ・ビルド・テスト・プロンプト品質評価・リリース設定の検証・アーキテクチャルールなど、開発・コントリビュート向けの情報は [docs/development/](docs/development/) にまとめています。
