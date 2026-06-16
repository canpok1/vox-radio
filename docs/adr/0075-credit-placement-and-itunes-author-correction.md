# 0075. クレジット/帰属の記載先を整理し item itunes:author の用途違いを是正する

- ステータス: 採用
- 日付: 2026-06-16

## コンテキスト

`feed.credit` が item の `itunes:author` に出力され、サンプルではデータソース帰属（「気象庁データを加工して作成」）に流用されていた。`itunes:author` は仕様上「著者名」を入れるフィールドで、データ帰属を入れるのは用途違い。一方、アセット・声のクレジットは ADR 0065 で `manifest.credits` → description 末尾のクレジット節に集約済みである。クレジット/帰属の記載先が分散し、`itunes:author` の意味も崩れていた。

## 決定

- item `itunes:author` は省略（channel 継承）にし、`feed.credit` を廃止する。
- データソース帰属（番組固定の出典表記）の記載先を description 末尾のクレジット節に統合し、アセット・声のクレジットと同じ `manifest.credits` に集約する。
- 番組固定のデータソース帰属は `ProgramConfig` の新フィールド（`Credits []string`）に持たせ、`manifest.Build` / `CollectCredits` で `manifest.credits` に統合（重複排除）する。
- 記事の出典（ADR 0046、台本セリフ＝音声紹介）は現状維持とし、feed テキストには出さない。
- これにより ADR 0065 の「`feed.credit`（`itunes:author`）を配信者表記として残す」決定を置換する。

## 結果

- **良い面**: `itunes:author` が本来の著者名用途に戻る。クレジット/帰属の記載先が description クレジット節に一元化され、番組固定の出典も番組設定に集約される。
- **移行/悪い面**: `feed.credit` 廃止は設定スキーマの破壊的変更。strict ロードで unknown key になるため、移行手順（`credit` を `vox-radio.yaml` の `credits` へ移設）が必要。
- **トレードオフ**: item 単位で著者を変える表現は失うが、現状その需要は無く channel 継承で十分。

## 検討した代替案

- **channel `<copyright>` タグ新設**: 番組全体の権利表記に適すが、item 単位不可・クライアントで目立ちにくい・既存クレジット集約と分散するため却下。RSS 標準・iTunes 名前空間に item 単位の copyright は存在しないことも確認した。
- **`feed.credit` を配信者表記として存置（ADR 0065 現状）**: `itunes:author` の用途違いが残るため却下した。
