# vox-radio

**設定ファイルを元に、ラジオ番組（ポッドキャスト）の音声を自動生成する CLI ツールです。**

原稿の生成に生成AI（Gemini）、音声の合成に VOICEVOX を利用します。

## インストール

```bash
curl -fsSL https://github.com/canpok1/vox-radio/releases/latest/download/install.sh | bash
```

設置先の変更・特定バージョンの指定などは[インストールガイド](docs/installation.md)を参照してください。

## クイックスタート

すぐ動くサンプル設定で番組を 1 本作る最短手順です（サンプルの詳細は[設定方法](#設定方法)）。

### 1. 前提条件を準備する

**`GEMINI_API_KEY`** を [Google AI Studio](https://aistudio.google.com/) で取得し、環境変数に設定します。

```bash
export GEMINI_API_KEY=<your-key>
```

**VOICEVOX Engine** をいずれかの方法でインストールして起動します（既定 `http://localhost:50021`）。

- [VOICEVOX 公式アプリ](https://voicevox.hiroshiba.jp/)をインストールして起動する
- Docker で起動する: `docker run -d -p 50021:50021 voicevox/voicevox_engine:cpu-latest`

### 2. サンプルで番組を生成する

```bash
# サンプル設定一式を sample/ に生成
vox-radio init --sample

# サンプルでそのまま番組生成
vox-radio --config sample/vox-radio.yaml episodegen --spec sample/episode-spec.yaml
```

出力先は `output/<YYYYMMDDHHMMSS>/` です。

## 使い方

vox-radio は次の 6 ステップで番組を生成します。各ステップは前のステップの出力を受け取って次へ渡します。

```
collect → rundown → script → synth → assemble → manifest
```

| ステップ | 概要 |
|----|------|
| collect | コーナーごとにフィード・URL から記事を収集 |
| rundown | LLM が記事を選別し番組設計図を生成 |
| script | 番組設計図から台本を生成（多段の LLM パイプライン） |
| synth | VOICEVOX で音声クリップを合成 |
| assemble | クリップとイントロ・アウトロを ffmpeg で結合し MP3 化 |
| manifest | 配信用の番組情報（タイトル・要約・記事など）を JSON 出力 |

`episodegen` で全ステップを一括実行します。

```bash
# 一括実行
vox-radio episodegen --spec episode-spec.yaml

# 出力とログを1ツリーに集約する場合
vox-radio episodegen --spec episode-spec.yaml --out-dir output --log-dir output/logs
```

各ステップは個別にも実行できます。

```bash
vox-radio episodegen collect  --out work/01_articles.json --spec sample/episode-spec.yaml
vox-radio episodegen rundown  --in work/01_articles.json --out work/02_rundown.json --spec sample/episode-spec.yaml
vox-radio episodegen script   --in work/02_rundown.json --out work/04_script.json --spec sample/episode-spec.yaml
vox-radio episodegen synth    --in work/04_script.json --out-dir work/clips
vox-radio episodegen assemble --in work/04_script.json --clips work/clips --out work/episode.mp3 --spec sample/episode-spec.yaml
```

ログは既定で `.vox-radio/logs/` に出力されます。

## 設定方法

`vox-radio init` で設定ファイルのテンプレートを生成し（既存ファイルは上書きしません）、次のファイルを編集します。各フィールドの定義は右列のリファレンスを参照してください。

| ファイル | 内容 | リファレンス |
|---|---|---|
| `vox-radio.yaml` | 共通設定（LLM / VOICEVOX URL / キャラクター） | [vox-radio.md](internal/cli/skills/vox-radio/references/vox-radio.md) |
| `episode-spec.yaml` | エピソード仕様（番組情報・コーナー・アセット参照） | [episode-spec.md](internal/cli/skills/vox-radio/references/episode-spec.md) |
| `assets/assets.yaml` | アセット設定（ジングル・効果音・BGM） | [assets.md](internal/cli/skills/vox-radio/references/assets.md) |
| `feed-spec.yaml` | RSS フィード生成設定（`feedgen` で使用） | [feed-spec.md](internal/cli/skills/vox-radio/references/feed-spec.md) |
| `slack-spec.yaml` | Slack 投稿設定（`slackpost` で使用） | [slack-spec.md](internal/cli/skills/vox-radio/references/slack-spec.md) |

番組生成に必要なのは `vox-radio.yaml` と `episode-spec.yaml` で、残りはアセット演出・配信を使う場合に編集します。

```bash
vox-radio init
```

記入済みのサンプルは `vox-radio init --sample` で生成できます（ずんだもん・めたん MC のお天気番組。音声アセットは同梱しないため、効果音・BGM はコメントアウト済みの記入例です）。コーディングエージェントに任せる方法は「[応用的な設定方法](#応用的な設定方法)」を参照してください。

### アセット設定

ジングル（イントロ/アウトロ）・効果音（SE）・BGM を番組に組み込めます（`assemble` で合成）。

1. 使う音声ファイルを `assets/` に置く
2. 各素材を登録する（`assets.yaml`）。音量やフェードのほか、BGM はセリフ中に音量を下げる度合い（ダッキング）なども設定できる
3. `assets check` で設定を検証し、`assets preview` で素材ごとの鳴り方を確認する
4. 各コーナーで「いつ何を鳴らすか」を割り当てる（`episode-spec.yaml`）。コーナーの開始・終了に鳴らすジングルや効果音、コーナー中に流す BGM を指定する

```bash
vox-radio assets check assets/assets.yaml
vox-radio assets preview assets/assets.yaml --id jingle:opening --out preview.mp3
```

### キャラクター設定

キャラカタログは、番組に出演させるキャラクターの一覧です。`vox-radio.yaml` の `characters` に、キャラごとの名前・一人称・口調・性格と、使える音声スタイル（VOICEVOX の声色）を登録します。台本生成と音声合成はこのカタログを参照します。

```yaml
characters:
  zundamon:
    name: ずんだもん
    pronoun: ボク
    speech_suffix: ["〜のだ", "〜なのだ"]
    personality: ["元気", "明るい"]
    default_style: ノーマル
    styles:
      ノーマル: 3    # スタイル名 → VOICEVOX の話者ID
      あまあま: 1
      なみだめ: 76
```

台本生成ではセリフの感情に応じてスタイルが選ばれ、音声合成はそのスタイルの声色で読み上げます。指定がない・不正なときは `default_style` が使われます。

### 配信設定

生成した番組は次の方法で配信できます。

- **RSS フィード** — `feedgen` が履歴キャッシュと `feed-spec.yaml` から RSS 2.0 + iTunes フィード（`feed.xml`）を生成します（manifest・mp3 は不要）。
- **Slack 配信** — `slackpost` が `manifest.json` と `slack-spec.yaml` をもとに mp3 を Slack へ投稿します。親メッセージ＋スレッド返信の 2 段構成で、タイムアウト後の再実行でも二重投稿なしに再開できます。

各コマンドのフラグは[コマンド一覧](#コマンド一覧)を参照。

### キャッシュ（過去回の記憶）

vox-radio は、過去に放送した番組の情報（扱った話題や放送回など）をキャッシュに記録します。これにより、過去回で触れた内容を新しい回の会話に織り込んだり、放送回数を管理したりできます。

キャッシュは番組ごとに `program.id` をキーとして保存されます（`.vox-radio/cache/<program.id>.jsonl`）。**このため `episode-spec.yaml` の `program.id` は必須**で、未設定だと `episodegen`（番組生成）や `episodegen check` でエラーになります。放送回数・過去回の参照・出演回数などはすべて `program.id` 単位で記録されます。

## 応用的な設定方法

Claude Code などのコーディングエージェントを使うなら、設定方法で説明した編集をエージェントに任せられます。

```bash
vox-radio install --skills
```

エージェントスキル（`SKILL.md` ＋ フィールド定義 `references/*.md`）が `.claude/skills/vox-radio/` に入ります。あとは「ラジオ番組の設定を作って」と依頼すれば、`init` →リファレンス参照で編集→ `check` 検証まで自動で仕上げます。

## コマンド一覧

コマンド・サブコマンド・フラグの詳細は自動生成ドキュメントを参照してください。

- [docs/cli/vox-radio.md](docs/cli/vox-radio.md) — コマンド一覧
- 各サブコマンド: `vox-radio <command> --help`

## 開発

開発環境・ビルド・テスト・プロンプト評価・リリース検証・アーキテクチャなど、開発者向けの情報は [docs/development/](docs/development/) を参照してください。
