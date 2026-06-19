# 0087. コーナーの source を type 付き配列（discriminated array）へ再設計する

- ステータス: 採用
- 日付: 2026-06-19
- 旧決定を一部改訂: [0011-restructure-config-schema-characters-program-corners.md](0011-restructure-config-schema-characters-program-corners.md)

## コンテキスト

ADR-0011 でコーナーがデータソースを `source` として持つ構成を導入したが、`source` は `feeds`（RSS/Atom）と `articles`（Web ページ URL）を別フィールドに分けて持つ形（`SourceConfig = FeedsConfig{Feeds, Articles}`）になっていた。

この形には次の課題がある。

- 「情報ソース」という同一概念が種別ごとに別フィールドへ分散し、設定の見通しが悪い。
- ソースの記述順（feed と web の混在順）を保持できない。
- 種別が増えた場合にフィールドが増え続け、構造が拡張しづらい。

## 決定

`source` を 1 つの配列にまとめ、各要素の `type` フィールド（`feed` / `web`）で種別を判別する形へ再設計する。

```yaml
source:
  - type: feed
    url: https://example.com/feed
    max_items: 5
  - type: web
    url: https://example.com/articles/1
```

- 型は `SourceEntry{Type, URL, MaxItems}` を新設し、`SourceConfig = []SourceEntry` とする（`FeedEntry` / `FeedsConfig` は廃止）。`CornerConfig.Source` はスライス型にし、`len()==0` を「ソースなし」とみなす。
- type 値は `feed` / `web`。`max_items` は `feed` のみ有効。
- 設定ロード時に source をバリデーションする（type が `feed`/`web` 以外、`url` 空、`web` での `max_items` 指定をエラーにする）。
- 収集処理（gather）は要素ごとに `type` で分岐し、既存の `fetchFeed` / `fetchArticle` を呼び分ける。
- **破壊的変更**とし、旧書式（`source.feeds` / `source.articles`）は廃止する。strict ロードで旧キーはエラーになるため、リリースノートで移行を案内する。

## 結果

- feed と web を同一概念として記述順を保ったまま列挙でき、設定の見通しが良くなる。種別追加時も `type` の値を増やすだけで拡張できる。
- ロード時バリデーションにより、不正な type や url 欠落を実行前に検出できる。
- 旧書式の設定ファイルは strict ロードでエラーになるため、既存ユーザーは新書式への移行が必要（リリースノートで案内）。

## 検討した代替案

- **旧書式（feeds/articles の別フィールド）を維持**: 既存設定を壊さないが、概念分散・順序非保持・拡張性の課題が残るため却下。
- **旧書式も読める後方互換を持たせる**: 移行は楽だが、2 書式の併存でロード処理とドキュメントが複雑化するため却下（個人運用で移行コストは小さい）。
- **type 値を `feed` / `article`（旧内部名準拠）にする**: 利用者視点では「web ページ」の方が直感的なため `web` を採用。
