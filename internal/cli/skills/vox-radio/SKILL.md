---
name: vox-radio
description: vox-radio の設定ファイル（vox-radio.yaml・episode-spec.yaml・アセット設定・feed-spec.yaml・slack-spec.yaml）を新規作成・修正するスキル。フィールド定義は同梱の references/*.md を参照し、check コマンドで検証完了まで繰り返す。
allowed-tools: Bash, Read, Grep, Glob, Edit, Write, AskUserQuestion
---

## 概要

vox-radio の設定ファイルを作成・修正するスキルです。

- **新規作成モード**: `vox-radio init` でテンプレートを生成し、各設定ファイルを編集して完成させる
- **不備修正モード**: 既存設定ファイルの `check` コマンドエラーを解消する

## 利用可能なリファレンス

各設定ファイルのフィールド定義は同梱の `references/` ディレクトリを参照してください。

| ファイル | リファレンス | 検証コマンド |
|---|---|---|
| `vox-radio.yaml` | `references/vox-radio.md` | `vox-radio config check --config <パス>` |
| `episode-spec.yaml` | `references/episode-spec.md` | `vox-radio episodegen check <パス>` |
| アセット設定 YAML | `references/assets.md` | `vox-radio assets check <パス>` |
| `feed-spec.yaml` | `references/feed-spec.md` | `vox-radio feedgen check` |
| `slack-spec.yaml` | `references/slack-spec.md` | `vox-radio slackpost check` |

> リファレンスの記載と check コマンドの挙動が食い違う場合は、check コマンドの検証結果を正としてください。
> 食い違いはスキルとバイナリの版ずれが原因のことがあります。下記「バージョン整合チェック」で版を揃えると解消します。

## ワークフロー

### バージョン整合チェック（ワークフローの最初に実施）

スキルファイルはインストール時のバイナリと同一バージョンで配布されます。バイナリだけを更新すると古いスキルが残り、リファレンスと実際の挙動が食い違うことがあります。作業を始める前に版ずれを確認してください。

1. スキルディレクトリの `.skill-version`（インストール時に記録されたバイナリ版）を読む
2. `vox-radio --version` で現在のバイナリ版を取得する（出力形式: `vox-radio version X.Y.Z`）
3. 両者を比較し、以下に従う
   - **いずれかが `dev`、または `.skill-version` が存在しない**: 比較不能（ローカルビルド等）。警告を出すだけで、そのまま作業を続行する
   - **一致**: そのまま作業を続行する
   - **バイナリが新しい**: `AskUserQuestion` で確認のうえ `vox-radio install --skills --force` を実行してスキル（と `.skill-version`）を再生成し、最新の references で作業を続行する
   - **スキルが新しい（バイナリが古い）**: `AskUserQuestion` で確認のうえ、バイナリの更新手順（`scripts/install.sh` の再実行、または `go install`）を案内する。バイナリの自動更新は行わない（環境依存のため）

### モード判定

まず作業モードを判断してください。

1. **新規作成**: 設定ファイルが存在しない、または `vox-radio init` を実行していない場合
2. **不備修正**: 設定ファイルは存在するが `check` コマンドでエラーが出る場合

### 新規作成モード

1. `vox-radio init` を実行してテンプレートを生成する

   ```bash
   vox-radio init
   ```

2. 生成された各ファイルを確認し、必要なフィールドを設定する

   - 設定が不明なフィールドは `references/*.md` を参照する
   - `characters` の `styles` に VOICEVOX の話者ID を設定する
   - `llm.openai.api_key_env` に API キーの環境変数名を設定する

3. 各ファイルを `check` コマンドで検証し、エラーがなくなるまで修正を繰り返す（下記「検証ループ」参照）

### 不備修正モード

1. `check` コマンドでエラー内容を確認する
2. `references/*.md` でフィールド定義を確認する
3. 設定ファイルを修正する
4. 再度 `check` で検証する（下記「検証ループ」参照）

### 検証ループ

各設定ファイルについて、**終了コード 0 になるまで** check コマンドを繰り返し実行してください。

```bash
# 共通設定の検証
vox-radio config check --config vox-radio.yaml

# エピソード仕様の検証（--config で共通設定も一緒に検証）
vox-radio episodegen check episode-spec.yaml --config vox-radio.yaml

# アセット設定の検証
vox-radio assets check assets/assets.yaml

# フィード設定の検証
vox-radio feedgen check

# Slack 設定の検証
vox-radio slackpost check
```

check コマンドがエラーを報告した場合は、エラーメッセージに従って設定ファイルを修正し、再度実行してください。

## 注意事項

- フィールド値を SKILL.md 本体に直接記載せず、常に `references/*.md` を参照すること
- キャラID（`vox-radio.yaml` の `characters` キー）は `episode-spec.yaml` の `corners[].cast` と一致させること
- `episode-spec.yaml` の `assets_files` に列挙したパスが実際に存在することを確認すること
- `vox-radio init` は既存ファイルを上書きしないため、再実行しても安全
