# 0083. lefthook による git フック管理（保護ブランチへの直接コミット禁止・品質ゲート）

- ステータス: 採用
- 日付: 2026-06-18

## コンテキスト

エージェント（Claude Code）が誤って `main` ブランチへ直接コミットする事故が時々発生していた。また、コミット時のフォーマット漏れや push 時のテスト未実施による品質低下も課題だった。

git フックで上記を自動化したいが、次の要件がある。

- `main`/`master` への直接コミットを拒否する（エージェント・人手の双方で有効にしたい）。
- コミット前に `gofmt` で自動フォーマットし、`golangci-lint` でエラーがあればコミットを中断する。
- push 前に `go test ./...` を実行し、失敗時は push を中断する。
- 設定ファイルをリポジトリで共有し、チーム全体で同じフックを使えるようにする。
- devcontainer 環境で自動的に有効化される仕組みが必要。

## 決定

`lefthook.yml` をリポジトリ直下に追加し、pre-commit・pre-push フックを定義する。`make setup` に lefthook のインストール（`go install`）と `lefthook install`（git フックの有効化）を含める。devcontainer の `post-create.sh` から `make setup` を呼び出すことで、コンテナ起動時にフックが自動有効化される。

フック設計：

- **pre-commit（piped: true）**: ブランチガードを最初に実行し、`main`/`master` への直接コミットを拒否。次に `gofmt` で変更ファイルを自動フォーマット（`stage_fixed: true` で修正済みファイルを再ステージ）。最後に `golangci-lint run` で lint チェック。
- **pre-push**: `go test ./...` を実行し、失敗時に push を中断。

`piped: true` を使うことで、ブランチガード失敗時に fmt/lint を無駄に走らせない。

lefthook を採用した理由：

- Go プロジェクトの `go install` でインストールでき、外部パッケージマネージャ（Homebrew 等）不要。
- `lefthook.yml` をリポジトリで共有でき、チーム全体で同一設定を使える。
- `stage_fixed: true` など、ステージングの自動化を簡潔に記述できる。

## 結果

- `main`/`master` への直接コミットが pre-commit フックで拒否され、エージェント・人手両面での事故防止になる。
- コミット前に gofmt・golangci-lint が自動実行され、フォーマット漏れ・lint エラーの混入が抑止される。
- push 前に単体テストが実行され、テスト未実施のコードが push されにくくなる。
- `git commit --no-verify` / `git push --no-verify` でバイパス可能なため、完全な強制ではなく事故防止策の位置付け。
- devcontainer 起動時に `make setup` 経由で自動有効化される。

## 検討した代替案

- **husky（Node.js）**: Node.js が必要で、Go プロジェクトとの相性が悪い。lefthook はネイティブバイナリで Go 依存なし（`go install` 利用可）のため却下。
- **git hooks を直接管理**: `.git/hooks/` はリポジトリ管理外であり、チーム間で共有できない。lefthook の共有設定ファイル方式を採用。
- **pre-commit（Python製）**: Python 環境が必要で、同様の理由で却下。
