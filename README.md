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
collect → script → synth → assemble → publish
                                              └─ prune（古いエピソードを削除）
```

| コマンド | 概要 |
|----------|------|
| `collect` | `corners[].source` に定義したフィード・URL からコーナーごとに記事を収集し `articles.json` を生成する |
| `script` | 記事を LLM に渡して台本 `script.json` を生成する（summarize → write → direct の多段パイプライン） |
| `synth` | `script.json` をもとに VOICEVOX で音声クリップを合成する |
| `assemble` | 音声クリップとイントロ・アウトロを ffmpeg で結合し MP3 エピソードを生成する |
| `publish` | MP3 をホスティングディレクトリへコピーし、`episodes.json` と `feed.xml` を更新する |
| `prune` | 直近 N 件を残して古いエピソードを削除し、`episodes.json` と `feed.xml` を更新する |
| `run` | collect → script → synth → assemble → publish の全パイプラインを一括実行する |

### 設定ファイル

設定は2種類に分かれています。

| 種別 | ファイル | 内容 |
|------|---------|------|
| 共通設定 (config) | `vox-radio.yaml`（カレントディレクトリ、自動読込） | LLM / VOICEVOX URL / キャラカタログ |
| ジャンル別設定 (profile) | `sample-profiles/<genre>_profile.yaml` | program / corners（各コーナーの source でデータソース指定） / assets |

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

# 公開
vox-radio publish --in work/episode.mp3 --out-dir public \
    --profile sample-profiles/tech_profile.yaml

# 古いエピソードを削除
vox-radio prune --out-dir public --profile sample-profiles/tech_profile.yaml
```

### 詳細リファレンス

各コマンドのフラグ一覧は自動生成ドキュメントを参照してください。

- [docs/cli/vox-radio.md](docs/cli/vox-radio.md) — コマンド一覧
- 各サブコマンドの詳細: `vox-radio <command> --help`
