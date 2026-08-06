# 開発ガイド

vox-radio の開発・コントリビュートに必要な情報をまとめています。ツールの利用方法はリポジトリルートの [README](../../README.md) を参照してください。

> `make` 系のコマンドは開発者向けです。ツールの利用者はリリース版バイナリのみで完結します。

## 開発環境のセットアップ

開発環境は devcontainer での構築を推奨します。以下は devcontainer を使う場合の手順です（Docker と [Dev Containers](https://containers.dev/) 対応エディタが必要）。Go の開発環境をローカルに用意すれば、devcontainer なしでも開発できます。

1. `.devcontainer/.env-template` をコピーして `.devcontainer/.env` を作成する

   ```bash
   cp .devcontainer/.env-template .devcontainer/.env
   ```

2. `.devcontainer/.env` に各自の値を設定する

   | 変数名 | 説明 |
   |--------|------|
   | `GEMINI_API_KEY` | Gemini API キー（[Google AI Studio](https://aistudio.google.com/) で取得） |

3. devcontainer をリビルドして起動する

> **注意:** `.devcontainer/.env` には秘密情報が含まれるため、コミットしないこと（`.gitignore` で除外済み）。

## git フック（lefthook）

devcontainer を使う場合は起動時に `post-create.sh` → `make setup` が自動実行され、フックが有効になります。手動セットアップの場合は `make setup` を実行してください。

`make setup` 後は以下のフックが有効になります。

| タイミング | 処理 |
|---|---|
| pre-commit | `main`/`master` への直接コミットを拒否 |
| pre-commit | `gofmt` による自動フォーマット（変更ファイルのみ、自動再ステージ） |
| pre-commit | `golangci-lint` による lint チェック（エラー時はコミット中断） |
| pre-push | `go test ./...`（失敗時は push 中断） |

`git commit --no-verify` / `git push --no-verify` でバイパス可能ですが、CI でも同等のチェックが走ります。

## ビルド

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

## 動作確認用サンプル実行

`vox-radio init --sample --output-dir sample` で生成される「すぐ動くサンプル設定一式」（`sample/`）を使ってパイプライン全体を試すには `make run-sample` を実行します。`make run-sample` は内部で `init --sample --output-dir sample` を実行してから `sample/episode-spec.yaml` を生成・実行します。

```bash
make run-sample
```

出力先は `output/<YYYYMMDDHHMMSS>/` ディレクトリになります（例: `output/20260601053357/my-tech-radio_ep001.mp3`）。

プロファイルや出力先を変更する場合は `PROFILE` / `OUT_DIR` 変数で上書きできます。

```bash
# 別のプロファイルを使う
make run-sample PROFILE=path/to/your-episode-spec.yaml

# 出力先を指定する
make run-sample OUT_DIR=output/test
```

> **前提条件:** `GEMINI_API_KEY` 環境変数と VOICEVOX Engine が必要です。

## e2e テスト（BDD / Gherkin）

プロダクトの主要動線（init・各 check・episodegen 各ステップ/一括・feedgen・slackpost・キャッシュ連携）を、CLI バイナリを実際に実行して検証する e2e テストがあります。テストケースは `e2e/features/*.feature`（日本語 Gherkin）に仕様書として記述され、[godog](https://github.com/cucumber/godog) がそのまま実行します（ADR-0054）。

```bash
make e2e
```

- 外部依存（LLM / VOICEVOX / RSS フィード / Slack API）はモックサーバーで差し替えるため、API キーや実サービスは不要です。
- ffmpeg / ffprobe のみ実バイナリを使います。見つからない環境では `@ffmpeg` タグ付きシナリオが自動的にスキップされます。
- CI（`build.yml` の `e2e` ジョブ）では ffmpeg をインストールして全シナリオを実行します。
- テストは `e2e` ビルドタグで分離されており、通常の `make test` には含まれません。

## リリース設定の検証

`.goreleaser.yaml` を編集した後は、CI を待たずにローカルで構文・設定を検証できます。

```bash
make release-check
```

`goreleaser check` を実行し、設定の構文エラーや不整合を検出します。`goreleaser` は `make setup` ではインストールせず、`make release-check` 実行時に公式スクリプトでビルド済みバイナリを都度取得します。

## アーキテクチャ

プロダクトコード（Go）の層構造・依存ルールは [architecture.md](architecture.md) を参照してください。`.go` ファイルを変更するときは同ドキュメントの依存ルールに従います。

重要な技術判断は [ADR（docs/adr/）](../adr/) に記録しています。
