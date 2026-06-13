# docs

プロジェクトのドキュメント置き場。サブディレクトリごとに用途を分けている。

| ディレクトリ | 用途 |
|---|---|
| [`development/`](development/) | 開発・コントリビュート向けドキュメント（開発環境・ビルド・テスト・プロンプト評価・アーキテクチャルール）。入口は [`development/README.md`](development/README.md)。 |
| [`adr/`](adr/) | Architecture Decision Records。重要度の高い判断を記録する（MADR 軽量版・日本語）。新規作成は `create-adr` スキルを使う。一覧は [`adr/README.md`](adr/README.md)。 |
| [`cli/`](cli/) | CLI コマンドの自動生成リファレンス。`make docs` で再生成する。 |

## 運用メモ

- **重要な判断は ADR に残す**: アーキテクチャ・技術選定など後から「なぜ」を辿りたい判断は `adr/` に記録する。些末な実装詳細は対象外（ADR の冗長化を防ぐ）。
