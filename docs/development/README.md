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
   | `GH_TOKEN` | GitHub Personal Access Token |
   | `GEMINI_API_KEY` | Gemini API キー（[Google AI Studio](https://aistudio.google.com/) で取得） |
   | `TODOIST_API_TOKEN` | Todoist API トークン（[Todoist 設定 > 連携サービス](https://app.todoist.com/app/settings/integrations/developer) で取得） |

3. devcontainer をリビルドして起動する

> **注意:** `.devcontainer/.env` には秘密情報が含まれるため、コミットしないこと（`.gitignore` で除外済み）。

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

`vox-radio init --sample` で生成される「すぐ動くサンプル設定一式」（`sample/`）を使ってパイプライン全体を試すには `make run-sample` を実行します。`make run-sample` は内部で `init --sample` を実行してから `sample/episode-spec.yaml` を生成・実行します。

```bash
make run-sample
```

出力先は `output/<YYYYMMDDHHMMSS>/` ディレクトリになります（例: `output/20260601053357/episode.mp3`）。

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

## プロンプト品質評価フレームワーク

組み込みプロンプト（`internal/cli/prompts/*.md`）の品質を LLM-as-judge 方式で自動採点します。
第一弾として `proofread.md`（発音校正プロンプト）を対象にしています。

### 実行方法

```bash
export GEMINI_API_KEY=<your-api-key>
make eval
```

### 主な環境変数

| 変数名 | 説明 | デフォルト |
|---|---|---|
| `GEMINI_API_KEY` | Gemini API キー（必須） | — |
| `VOX_EVAL_MODEL` | 対象実行・judge に使うモデル | `gemini-3.1-flash-lite` |
| `VOX_EVAL_JUDGE_MODEL` | judge のみモデルを上書き | `VOX_EVAL_MODEL` と同じ |
| `VOX_EVAL_MIN_INTERVAL_MS` | API 呼び出しの最低間隔（ms） | `4500` |
| `VOX_EVAL_PROOFREAD_THRESHOLD` | 合否閾値（全体平均スコア） | `4.0` |
| `VOX_EVAL_SAMPLE_SIZE` | 汎化プールからのサンプル数 | `8` |
| `VOX_EVAL_SAMPLE_SEED` | サンプリング seed（省略時 = ISO 週番号） | ISO 週番号 |

### 評価の仕組み（二層データセット）

- **回帰セット**（`testdata/proofread_regression_cases.json`）: 連濁・熟字訓など既知の致命的誤読を必ず検出することを保証する固定ケース。毎回全件実行し、`expectation`（正解説明）付きで採点します。
- **汎化プール**（`testdata/proofread_pool_cases.json`）: 多様な語彙・文型を貯めたプールから週番号 seed でサンプリングし、judge が `original_text` から自律採点します。同一週は再現可能、週またぎで対象が入れ替わり固定データへの過適応を防ぎます。

4 観点（検出網羅性・誤検出抑制・修正正確さ・理由妥当性）を各 1〜5 点で採点し、全ケース×全観点の平均が閾値（デフォルト 4.0）以上で合格です。レートリミット・API エラーは inconclusive（`t.Skip`）扱いで fail にしません。

評価は週次の GitHub Actions ワークフロー（`.github/workflows/prompt-eval.yml`）でも自動実行されます。通常の開発 CI（`build.yml`）は変更せず、実 API を叩きません。

## リリース設定の検証

`.goreleaser.yaml` を編集した後は、CI を待たずにローカルで構文・設定を検証できます。

```bash
make release-check
```

`goreleaser check` を実行し、設定の構文エラーや不整合を検出します。`goreleaser` は devcontainer 起動時または `make setup` 実行時に自動インストールされます。

## アーキテクチャ

プロダクトコード（Go）の層構造・依存ルールは [architecture.md](architecture.md) を参照してください。`.go` ファイルを変更するときは同ドキュメントの依存ルールに従います。

重要な技術判断は [ADR（docs/adr/）](../adr/) に記録しています。
