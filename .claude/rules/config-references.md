---
paths:
  - "internal/config/**/*.go"
---

# 設定スキーマ↔referencesドキュメント反映ルール

`internal/config/**/*.go` の構造体フィールド（yaml タグ）を追加・変更・削除したときは、対応する `internal/cli/skills/vox-radio/references/*.md` を同じコミットで更新すること。

## 対応表（どの構造体がどの references を正とするか）

| ルート構造体 | 対応ドキュメント |
|---|---|
| `Config`（`vox-radio.yaml`） | `references/vox-radio.md` |
| `EpisodeSpec`（`episode-spec.yaml`） | `references/episode-spec.md` |
| `AssetsConfig`（アセット設定YAML） | `references/assets.md` |

## 実施手順

1. 変更したフィールドについて、対応する references の表に行を追加/更新/削除する（型・必須任意・デフォルト値・効果を明記）。
2. `go test ./internal/config/...` を実行し、`TestConfigFieldsDocumentedInReferences` が通ることを確認する（yaml タグ名が references に `` `field_name` `` の形式で記載されているかを機械的に突き合わせるテスト）。
3. 意図的にドキュメント化しないフィールド（内部専用等）がある場合は、`internal/config/references_test.go` の `refDocAllowlist` に理由コメント付きで追加する（テストをスキップさせるだけの対処はしない）。

このテストは表への記載有無のみを見る粗いチェックのため、内容の正確さ（デフォルト値・説明文の妥当性）は上記1で目視確認すること。
