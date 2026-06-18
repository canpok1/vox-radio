# 0085. 開発ツールをビルド済みバイナリで取得し make setup を高速化する

- ステータス: 採用
- 日付: 2026-06-18

## コンテキスト

`make setup` は golangci-lint・goreleaser・lefthook の3ツールを `go install`（ソースからビルド）で導入していた。これらのビルドは重く、特に Claude Code on the web のように**セッションごとにフレッシュなコンテナで毎回 `make setup` が走る環境**（ADR-0084）ではコールド時間が長く（実測で約3分）、開発・セッション開始のたびに待たされていた。

加えて、golangci-lint をソースからビルドする際は go.mod の go ディレクティブとのバージョン不整合を避けるため `GOTOOLCHAIN` を明示する回避策が必要で、Makefile が複雑になっていた。

また goreleaser は `make setup` で導入していたが、実際に使うのはローカルの `make release-check` だけで、CI のリリースは `goreleaser/goreleaser-action` を使う。常時経路の `make setup` に重いツールが含まれているのは無駄だった。

## 決定

`make setup` の `go install` を、公式が配布するビルド済みバイナリの取得に置き換える。

- **golangci-lint**: 公式 install.sh（`raw.githubusercontent.com/golangci/golangci-lint/HEAD/install.sh`）でバージョン指定取得する。チェックサム検証込み。ソースからビルドしないため `GOTOOLCHAIN` 回避策も不要になり削除する。
- **lefthook**: GitHub Releases のバイナリを取得する専用スクリプト `scripts/install-lefthook.sh` を追加する。OS/arch を判定（macOS は `MacOS`、Linux ARM は `arm64` に正規化）し、`lefthook_checksums.txt` で sha256 検証してから配置する。
- **goreleaser**: `make setup` から外す。`make release-check` で公式 run スクリプト（`goreleaser.com/static/run`、`VERSION` 指定）を使い、実行のたびにビルド済みバイナリを取得して `check` する。

ツールのバージョンは Makefile 冒頭の変数（`GOLANGCI_LINT_VERSION` / `GORELEASER_VERSION` / `LEFTHOOK_VERSION`）で一元管理する。`lefthook install` は `$GOPATH/bin` が PATH 外でも動くようフルパス（`$(GOBIN)/lefthook`）で呼ぶ。

## 結果

- `make setup` のコールド時間が大幅に短縮された（実測で約3分 → 約2秒）。ADR-0084 のセッション開始ごとの `make setup` 実行コストが実質無視できる水準になった。
- 常時経路から重い goreleaser の導入が消え、`make setup` は lint・git フックに必要なツールだけを入れるようになった。
- `GOTOOLCHAIN` 回避策が不要になり Makefile の `setup` ターゲットが簡潔になった。
- トレードオフ:
  - `make setup` 後に `goreleaser` 単体コマンドは使えなくなる（`make release-check` 経由で使う）。CI リリースは goreleaser-action を使うため影響なし。
  - `static/run` はキャッシュせず毎回ダウンロードするが、`release-check` は稀にしか実行しないため許容する。
  - インストールがネットワーク（GitHub Releases・goreleaser.com）に依存する。`go install` も同様にネットワーク依存だったため新規の制約ではない。

## 検討した代替案

- **goreleaser も setup に残してバイナリ取得化する**: 単体コマンドの利用を維持できるが、常時経路に不要なツールを残すことになり、setup から外す案を採用した。
- **lefthook をパッケージマネージャ（apt/brew 等）で導入**: 環境ごとに導入方法が分岐し、バージョン固定がしにくい。GitHub Releases のバイナリ直取得でクロスプラットフォーム・バージョン固定を両立した。
- **go install のまま GOTOOLCHAIN 等を調整して高速化**: ビルド自体が重く本質的に遅いため、ビルド済みバイナリ取得に切り替えた。
