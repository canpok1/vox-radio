# docs

プロジェクトのドキュメント置き場。サブディレクトリごとに用途を分けている。

| ディレクトリ | 用途 |
|---|---|
| [`development/`](development/) | 開発・コントリビュート向けドキュメント（開発環境・ビルド・テスト・プロンプト評価・アーキテクチャルール）。入口は [`development/README.md`](development/README.md)。 |
| [`adr/`](adr/) | Architecture Decision Records。重要度の高い判断を記録する（MADR 軽量版・日本語）。新規作成は `create-adr` スキルを使う。一覧は [`adr/README.md`](adr/README.md)。 |
| [`cli/`](cli/) | CLI コマンドの自動生成リファレンス。`make docs` で再生成する。 |

## ドキュメント方針（どこに何を書くか）

ドキュメントは利用者向けと開発者向けで置き場所を分けている（背景は [ADR-0062](adr/0062-split-user-and-developer-docs.md)）。新しく書くときは次の表に従う。

| 内容 | 置き場所 |
|---|---|
| 利用者向け（インストール・設定・使い方） | ルート [`README.md`](../README.md) |
| 開発者・コントリビュート向け（環境構築・ビルド・テスト・評価・アーキテクチャ） | [`development/`](development/) |
| CLI コマンド・フラグの説明 | cobra の `Short`/`Long` を編集し `make docs` で [`cli/`](cli/) を再生成（手書きしない） |
| 設定ファイルのフィールド定義 | [`internal/cli/skills/vox-radio/references/`](../internal/cli/skills/vox-radio/references/)（`install --skills` で配布。ここが正） |

## 運用メモ

- **重要な判断は ADR に残す**: アーキテクチャ・技術選定など後から「なぜ」を辿りたい判断は `adr/` に記録する。些末な実装詳細は対象外（ADR の冗長化を防ぐ）。
