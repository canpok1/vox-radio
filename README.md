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
| `collect` | RSS フィードや URL から記事を収集し `articles.json` を生成する |
| `script` | 記事を LLM に渡して台本 `script.json` を生成する（summarize → plan → write → direct の多段パイプライン） |
| `synth` | `script.json` をもとに VOICEVOX で音声クリップを合成する |
| `assemble` | 音声クリップとイントロ・アウトロを ffmpeg で結合し MP3 エピソードを生成する |
| `publish` | MP3 をホスティングディレクトリへコピーし、`episodes.json` と `feed.xml` を更新する |
| `prune` | 直近 N 件を残して古いエピソードを削除し、`episodes.json` と `feed.xml` を更新する |

### 詳細リファレンス

各コマンドのフラグ一覧は自動生成ドキュメントを参照してください。

- [docs/cli/vox-radio.md](docs/cli/vox-radio.md) — コマンド一覧
- 各サブコマンドの詳細: `vox-radio <command> --help`
