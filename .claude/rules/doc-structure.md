---
paths:
  - "README.md"
  - "docs/**/*"
---

# ドキュメント構成ルール

`README.md` または `docs/**` を変更したときは、[ADR-0062](../../docs/adr/0062-split-user-and-developer-docs.md) で定めた利用者向け/開発者向けの分離方針を維持すること。

変更後の確認には `doc-structure-checker` サブエージェントを呼び出して構成の崩れを検出する（read-only、修正はしない）。検出された NG 項目は ADR-0062 と `docs/README.md` の方針表に沿って修正する。

## 配置方針（どこに何を書くか）

- 利用者向け（インストール・設定・使い方）→ ルート `README.md`
- 開発者向け（環境構築・ビルド・テスト・評価・アーキテクチャ）→ `docs/development/`
- コマンド説明 → cobra `Short`/`Long` を正として `docs/cli/` に自動生成（README へ手書きしない。詳細は CLIドキュメント反映ルールを参照）
- 設定フィールド定義 → `internal/cli/skills/vox-radio/references/`（`install --skills` で配布）

## 主なチェック観点

- 撤去済みパス（`docs/architecture/`・`docs/draft/`）への参照を復活させない
- README に手書きコマンド表・`make` 実行手順を持ち込まない（開発者向けは `docs/development/`、コマンドは `docs/cli/`）
- `docs/README.md` の「ドキュメント方針」表を維持する
