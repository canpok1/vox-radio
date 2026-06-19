# vox-radio

**設定ファイルを元に、ラジオ番組（ポッドキャスト）の音声を自動生成する CLI ツールです。**

原稿の生成に生成AI（Gemini を推奨）、音声の合成に VOICEVOX を利用します。

[デモ番組を聴く](https://github.com/canpok1/vox-radio/releases/tag/demo)

## クイックスタート

すぐ動くサンプル設定で番組を 1 本作る最短手順です（サンプルの詳細は[設定方法](#設定方法)）。

### 1. vox-radio を導入する

```bash
brew install --cask canpok1/homebrew-tap/vox-radio
```

ffmpeg も同時に導入されます。Homebrew を使わない場合は「[インストール](#インストール)」章を参照してください。

### 2. 実行環境を整える

- **生成AIの API キー**: サンプルは Gemini を使う構成です。[Google AI Studio](https://aistudio.google.com/) でキーを取得しておきます。
- **VOICEVOX Engine**: インストールして起動します。導入方法は「[インストール](#インストール)」章を参照してください。

### 3. 設定ファイルを用意する

**BGM・効果音なし**と**BGM・効果音あり**のどちらか一方を選んでください。

**BGM・効果音なし**

```bash
# サンプル設定一式をカレントディレクトリに生成
vox-radio init --sample
```

**BGM・効果音あり** — サンプル音源パックを展開し、各コーナーに音を割り当て済みの設定を生成します。

```bash
# サンプル音源パックを取得して assets/ に展開
curl -LO "https://github.com/canpok1/vox-radio/releases/download/v$(vox-radio --version | awk '{print $NF}')/vox-radio-sample-assets.zip"
unzip vox-radio-sample-assets.zip -d assets

# 音源パックを使う設定一式を生成（assets/assets.yaml はパックのものを使う）
vox-radio init --sample-with-assets
```

`init` で生成された `.env` の `GEMINI_API_KEY` 欄に、手順 2 で取得した API キーを記入します（実行時に自動で読み込まれます）。

```
GEMINI_API_KEY=<your-key>
```

サンプルはこのまま番組を生成できます。番組内容やキャラクターを変えたい場合は各設定ファイルを編集します（詳細は[設定方法](#設定方法)）。

### 4. 番組を生成する

```bash
vox-radio episodegen --spec episode-spec.yaml
```

番組は `output/{program.id}_ep{NNN}.mp3` に生成されます（マニフェスト・中間ファイルも `output/` 配下に出力）。

## インストール

### vox-radio の導入

1. **Homebrew（推奨・macOS / Linux）**

   ```bash
   brew install --cask canpok1/homebrew-tap/vox-radio
   ```

   ffmpeg も同時に導入されます。

2. **install.sh（macOS / Linux）**

   ```bash
   curl -fsSL https://github.com/canpok1/vox-radio/releases/latest/download/install.sh | bash
   ```

3. **バイナリ手動ダウンロード（全OS・Windows 含む）**

   [GitHub Releases](https://github.com/canpok1/vox-radio/releases/latest) から `tar.gz` / `zip` を取得して PATH に配置します。ffmpeg も別途導入が必要です。

### 依存ソフトの導入

- **VOICEVOX Engine**: いずれかの方法でインストールして起動します（既定 `http://localhost:50021`）。vox-radio は起動完了まで自動的に待機します（デフォルト最大 60 秒）。
    - [VOICEVOX 公式アプリ](https://voicevox.hiroshiba.jp/)をインストールして起動する
    - Docker で起動する: `docker run -d -p 50021:50021 voicevox/voicevox_engine:cpu-latest`
- **ffmpeg**（`ffprobe` を含む）: Homebrew でインストールした場合は自動で導入されます。手動導入の場合:
    - macOS: `brew install ffmpeg`
    - Ubuntu / Debian: `sudo apt-get install ffmpeg`
    - その他は [ffmpeg 公式サイト](https://ffmpeg.org/download.html)

### 詳細

設置先の変更・特定バージョンの指定・対応環境などは[インストールガイド](docs/installation.md)を参照してください。

## 使い方

### 番組生成

記事の収集から音声合成までを自動で行い、1 本のエピソード（mp3）を生成します。処理は次の 6 ステップで進みます。各ステップは前のステップの出力を受け取って次へ渡します。

```
gather → rundown → script → synth → mix → manifest
```

| ステップ | 概要 |
|----|------|
| gather | コーナーごとにフィード・URL から記事を収集 |
| rundown | LLM が記事を選別し番組設計図を生成 |
| script | 番組設計図から台本を生成（多段の LLM パイプライン） |
| synth | VOICEVOX で音声クリップを合成 |
| mix | クリップにイントロ・アウトロ・SE を ffmpeg で合成し MP3 化 |
| manifest | 配信用の番組情報（タイトル・要約・記事など）を JSON 出力 |

`episodegen` で全ステップを一括実行します。

```bash
# 一括実行
vox-radio episodegen --spec episode-spec.yaml

# 出力とログを1ツリーに集約する場合
vox-radio episodegen --spec episode-spec.yaml --out-dir output --log-dir output/logs

# 既存の MP3 を上書きして再実行する場合
vox-radio episodegen --spec episode-spec.yaml --force
```

各ステップは個別にも実行できます。

```bash
vox-radio episodegen gather   --out work/01_articles.json --spec episode-spec.yaml
vox-radio episodegen rundown  --in work/01_articles.json --out work/02_rundown.json --spec episode-spec.yaml
vox-radio episodegen script   --in work/02_rundown.json --out work/04_script.json --spec episode-spec.yaml
vox-radio episodegen synth    --in work/04_script.json --out-dir work/clips
vox-radio episodegen mix      --in work/04_script.json --clips work/clips --out work/episode.mp3 --spec episode-spec.yaml
vox-radio episodegen manifest --spec episode-spec.yaml --rundown work/02_rundown.json --audio work/episode.mp3 --out work/manifest.json
```

ログは既定で `.vox-radio/logs/` に出力されます。

### フィード生成

配信用の RSS フィード（`feed.xml`）を生成します。`feedgen` が番組の履歴キャッシュと `feed-spec.yaml` から出力します（manifest・mp3 は不要）。既定では `public/feed.xml` に書き出されます（出力先は `feed-spec.yaml` で変更可）。

```bash
vox-radio feedgen --cache .vox-radio/cache/<program.id>.jsonl --spec feed-spec.yaml
```

### Slack投稿

生成した番組を Slack へ投稿します。`slackpost` がマニフェスト（`{program.id}_ep{NNN}_manifest.json`）と `slack-spec.yaml` をもとに mp3 をアップロードします。投稿は親メッセージ（mp3 ＋ 初期コメント）とスレッド返信（要約＋コーナー）の 2 段構成です。投稿の進捗は状態ファイルに記録されるため、途中で失敗して再実行しても、mp3 を二重に投稿せず続きから再開します。

実行前に、以下の環境変数を設定しておきます（`init` 生成の `.env` に記入欄があります）。

- **Bot トークン**: `vox-radio.yaml` の `slack.bot_token_env` で指定した環境変数
- **投稿先チャンネル ID**: `slack-spec.yaml` の `slack.channel_env` で指定した環境変数

```
SLACK_BOT_TOKEN=xoxb-...
SLACK_CHANNEL_ID=C0123456789
```

```bash
vox-radio slackpost --manifest output/{program.id}_ep{NNN}_manifest.json --spec slack-spec.yaml
```

### テンプレートレンダリング

`render` は manifest.json を Go 標準の [text/template](https://pkg.go.dev/text/template) 記法でレンダリングします。テンプレートはファイル（`--template`）またはインライン文字列（`--template-string`）のどちらか一方で指定します。CI での値抽出や配信メモの生成に使います。

```bash
# 回番号を取り出す
vox-radio render --manifest output/manifest.json --template-string '{{.EpisodeNumber}}'

# リリースタイトルを組み立てる
vox-radio render --manifest output/manifest.json --template-string '第{{.EpisodeNumber}}回 {{.EpisodeTitle}}'

# テンプレートファイルから配信メモを生成
vox-radio render --manifest output/manifest.json --template release-note.tmpl --output RELEASE_NOTES.md
```

テンプレートで参照できるフィールドと関数の一覧: [manifest.md](internal/cli/skills/vox-radio/references/manifest.md)

## 設定方法

設定ファイルは `vox-radio init` で生成します。

```bash
# 設定テンプレートを生成
vox-radio init

# 記入済みサンプルを生成（ずんだもん・めたんMCのお天気番組）
vox-radio init --sample
```

`init` は次のファイルのテンプレートを生成します（既存ファイルは上書きしません）。各フィールドの定義は右列のリファレンスを参照してください。

| ファイル | 内容 | リファレンス |
|---|---|---|
| `vox-radio.yaml` | 共通設定（LLM / VOICEVOX URL / キャラクター） | [vox-radio.md](internal/cli/skills/vox-radio/references/vox-radio.md) |
| `episode-spec.yaml` | エピソード設定（番組情報・コーナー・アセット参照） | [episode-spec.md](internal/cli/skills/vox-radio/references/episode-spec.md) |
| `assets/assets.yaml` | アセット設定（ジングル・効果音・BGM） | [assets.md](internal/cli/skills/vox-radio/references/assets.md) |
| `feed-spec.yaml` | RSS フィード生成設定（`feedgen` で使用） | [feed-spec.md](internal/cli/skills/vox-radio/references/feed-spec.md) |
| `slack-spec.yaml` | Slack 投稿設定（`slackpost` で使用） | [slack-spec.md](internal/cli/skills/vox-radio/references/slack-spec.md) |
| `.env` | 生成AI・VOICEVOX・Slack の環境変数テンプレート（記入欄を埋めて使用） | — |

`slackpost` の投稿文テンプレート（`slack-parent.tmpl`・`slack-thread.tmpl`）は `template/` ディレクトリ配下に生成されます。

番組生成に必要なのは `vox-radio.yaml` と `episode-spec.yaml` で、残りはアセット演出・配信を使う場合に編集します。サンプル（`init --sample`）には音声アセットを同梱しないため、効果音・BGM はコメントアウト済みの記入例になっています。コーディングエージェントに任せる方法は「[エージェントでの番組制作](#エージェントでの番組制作)」を参照してください。

### 共通設定

`vox-radio.yaml` には番組全体で共通する設定を記載します。原稿生成に使う LLM（OpenAI 互換 API。Gemini を推奨。ほかに Dify にも対応）と VOICEVOX の接続先、出演キャラクター（キャラカタログ）、過去回キャッシュの設定を含みます。

**キャラクター（キャラカタログ）** — 番組に出演させるキャラクターの一覧です。`characters` に、キャラごとの名前・一人称・口調・性格と、使える音声スタイル（VOICEVOX の声色）を登録します。台本生成と音声合成はこのカタログを参照します。

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

**キャッシュ（過去回の記憶）** — vox-radio は過去に放送した番組の情報（扱った話題や放送回など）をキャッシュに記録し、過去回で触れた内容を新しい回の会話に織り込んだり、放送回数を管理したりします。キャッシュは番組ごとに `episode-spec.yaml` の `program.id` をキーとして保存されます（`.vox-radio/cache/<program.id>.jsonl`）。**このため `program.id` は必須**で、未設定だと `episodegen`（番組生成）や `episodegen check` でエラーになります。

### エピソード設定

`episode-spec.yaml` は 1 回分の番組内容を定義します。番組タイトルなどの基本情報、コーナー（話題ブロック）とそのデータソース、使用するアセットの参照、キャッシュのキーになる `program.id`（必須）を記載します。

### アセット設定

`assets/assets.yaml` でジングル（イントロ/アウトロ）・効果音（SE）・BGM を定義し、番組に組み込めます（`mix` で合成）。次の手順で設定を固めるのがおすすめです。

1. 使う音声ファイルを `assets/` に置く
2. 各素材を登録する（`assets.yaml`）。音量やフェードのほか、BGM はセリフ中に音量を下げる度合い（ダッキング）なども設定できる
3. `assets check` で設定を検証し、`assets preview` で素材ごとの鳴り方を確認する（`--id` には手順 2 で登録済みの素材を指定します）

   ```bash
   vox-radio assets check assets/assets.yaml
   vox-radio assets preview assets/assets.yaml --id jingle:theme --out preview.mp3
   ```

4. 各コーナーで「いつ何を鳴らすか」を割り当てる（`episode-spec.yaml`）。コーナーの開始・終了に鳴らすジングルや効果音、コーナー中に流す BGM を指定する

> **サンプル音源パックを使う場合**: 自分で音源を用意しなくても、ジングル・効果音・BGM 入りのサンプルパックを使えば上記の手順 1・2 を省けます。使い方は[クイックスタート](#クイックスタート)の「BGM・効果音あり」を参照してください。
>
> パック展開後に `vox-radio init --sample-with-assets` を実行すると、各コーナーへの割り当て（手順 4）まで済んだサンプル設定（`episode-spec.yaml` 等）が生成され、そのまま番組生成できます（クイックスタートの「BGM・効果音あり」と同じ）。

### RSS フィード生成設定

`feed-spec.yaml` には配信フィードの情報（言語・配信者名・連絡先・番組サイト URL・各エピソード音声の URL テンプレートなど）を設定します。生成は「[フィード生成](#フィード生成)」を参照してください。

### Slack 投稿設定

`slack-spec.yaml` には投稿先チャンネルや各メッセージのテンプレートを設定します。Bot トークンは `vox-radio.yaml` の `slack.bot_token_env` で指定した環境変数から読み込まれます。投稿は「[Slack投稿](#slack投稿)」を参照してください。

## 応用的な使い方

### エージェントでの番組制作

設定方法で説明した編集は、コーディングエージェントに任せることもできます。`vox-radio install --skills` は、エージェントスキル（`SKILL.md` ＋ フィールド定義 `references/*.md`）をインストールします。

```bash
# 既定: .claude/skills/vox-radio/
vox-radio install --skills

# 別のエージェントのスキルディレクトリへ展開する場合
vox-radio install --skills --skills-dir <スキルディレクトリ>
```

あとは「ラジオ番組を作って」と依頼すれば、エージェントがどんな番組にしたいか（テーマ・出演キャラ・コーナーなど）を質問し、その回答をもとに `init` →リファレンス参照で設定編集→ `check` 検証まで仕上げます。VOICEVOX・API キーなどの実行環境が整っていれば、続けて番組（mp3）の生成まで行えます。設定の一部だけ直したい・使い方を相談したいといった依頼にも、同じスキルで対応できます。

### お便りフォームの作成

Google フォームでリスナーからお便りを募り、その回答を RSS フィードとして公開すれば、vox-radio のコーナーのデータソースとして取り込めます（Google フォーム側の設定は vox-radio の責務の範囲外です）。設定方法は[お便りフォームの作成](docs/guides/listener-form.md)を参照してください。

### GitHub Actions で定期投稿

GitHub Actions のスケジュール実行で、番組生成から Slack 投稿までを定期的に自動化できます。最小構成のサンプルは[GitHub Actions で定期的に Slack へ投稿する](docs/guides/github-actions-slack.md)を参照してください。

## コマンド一覧

コマンド・サブコマンド・フラグの詳細は自動生成ドキュメントを参照してください。

- [docs/cli/vox-radio.md](docs/cli/vox-radio.md) — コマンド一覧
- 各サブコマンド: `vox-radio <command> --help`

## 開発

開発環境・ビルド・テスト・プロンプト評価・リリース検証・アーキテクチャなど、開発者向けの情報は [docs/development/](docs/development/) を参照してください。

## ライセンス・免責事項

本ツールは MIT ライセンスで提供されます（[LICENSE](LICENSE)）。

作った番組を自分で楽しむだけなら気軽に使えますが、番組を公開・配信する場合は注意が必要です。フィード・キャラクター音声（VOICEVOX）・BGM や効果音などの音声素材には**それぞれ利用規約があり、これを守る責任は利用者にあります**。たとえば、フィードの個人利用限定の指定に従うことや、合成音声・音声素材を公開する際にクレジット表記（例: `VOICEVOX:ずんだもん`）を行うことが必要です。

規約・クレジット表記の詳細と無保証事項は [DISCLAIMER.md](DISCLAIMER.md) を参照してください。
