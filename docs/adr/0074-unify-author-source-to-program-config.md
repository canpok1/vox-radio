# 0074. 番組の著者名の出典を番組設定に一本化する

- ステータス: 採用
- 日付: 2026-06-16

## コンテキスト

番組の著者名が番組設定（`ProgramConfig.Author`）と feed 設定（`FeedConfig.Author`）に二重記載されていた。番組名・説明は番組設定 → manifest → cache → feed と伝播し番組設定が唯一の出典になっているが、著者名だけ manifest に出力されていなかった。`ProgramConfig.Author` は MP3 の artist タグ専用で、feed の channel `itunes:author` は `FeedConfig.Author`（必須）から独立取得していた。そのため同じ著者名を2箇所に書く必要があり、不整合の余地があった。

## 決定

著者名の唯一の出典を `ProgramConfig.Author` に一本化する。`ProgramConfig.Author` → `model.Manifest.Author`（新設）→ `cache.Entry.Author`（新設）→ feed channel `itunes:author` と、Title/Description と同じ経路で伝播させる。`FeedConfig.Author`（`feed.author`）と author 必須バリデーションを廃止する。

## 結果

- **良い面**: 著者名の記載が番組設定1箇所に集約され、二重管理と不整合が解消する。Title/Description と伝播経路が揃い一貫する。
- **移行/悪い面**: `feed.author` を廃する設定スキーマの破壊的変更。`feedgen check` は strict ロードのため、既存 `feed-spec.yaml` に `author` キーが残ると unknown key エラーになり、移行手順（`feed.author` 削除）が必要。
- **トレードオフ**: channel `itunes:author` は最新 cache エントリの Author を使うため、旧エントリしか無い場合は空になりうる（最新エピソード再生成で反映される）。

## 検討した代替案

- **`feed.author` をフォールバックとして残す**（cache に author があれば優先、無ければ `feed.author`）: 後方互換は高いが設定が2箇所残り、重複が完全には解消しないため却下した。
