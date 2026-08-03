# 0102. golangci-lint のインストールを GitHub API 非依存にする

- ステータス: 採用
- 日付: 2026-08-03

## コンテキスト

golangci-lint は自身をビルドした Go のバージョンが対象プロジェクトの `go.mod` の `go` ディレクティブより低いと実行を拒否する。`go.mod` は `go 1.26.1` を要求しており、Makefile は互換バージョン（`v2.12.2`、公式バイナリは go1.26.2 ビルド）を指定済みだった。

しかし `make setup` は golangci-lint の公式 `install.sh` を使っており、これは `api.github.com` でタグ・チェックサム情報を解決する。GitHub アクセスが対象リポジトリ（`canpok1/vox-radio`）のみに制限される Claude Code のリモートセッションでは、golangci-lint リポジトリへの `api.github.com` 呼び出しが 403 になり `make setup` が失敗する。結果、コンテナに元から入っている古い golangci-lint（go1.25 ビルド）が使われ続け、バージョン不一致エラーが多発していた。

## 決定

`install.sh` を使うのをやめ、GitHub Releases のアセット（tarball と checksums.txt）を `api.github.com` を経由しない直接 URL（`github.com/<repo>/releases/download/...`）から取得し、sha256 検証してから配置するスクリプト `scripts/install-golangci-lint.sh` に置き換えた。既存の `scripts/install-lefthook.sh` と同じ設計。

SessionStart フック（ADR-0084）は `make setup` を呼ぶだけなので、追加変更なしでこの対策の恩恵を受ける。

## 結果

Claude Code のリモートセッションでも `make setup` / `golangci-lint run` が成功するようになった。直接ダウンロード方式は通常の開発機・GitHub Actions CI でも同様に動作するため、退行はない。今後 `go.mod` の `go` バージョンを上げる際は、`GOLANGCI_LINT_VERSION` もビルド Go バージョンが対応するものへ追従して更新する必要がある。

## 検討した代替案

- **`go install` + `GOTOOLCHAIN` 固定**: ソースビルドのため 60〜90 秒かかり、`GOTOOLCHAIN` を `go.mod` のバージョンへ追従させる仕組みが別途必要で複雑なため却下。
- **`go.mod` の `go` ディレクティブを据え置く**: 根本原因（インストール手段の GitHub API 依存）を残したままの回避策であり、Go の新バージョン採用自体を制約するため却下。
