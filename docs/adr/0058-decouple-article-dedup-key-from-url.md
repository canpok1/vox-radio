# 0058. 記事の重複判定をURLから内容ベースの識別キー（DedupKey）へ分離する

- ステータス: 採用
- 日付: 2026-06-12
- 一部改訂: [0091-add-links-and-text-source-types.md](0091-add-links-and-text-source-types.md)（feed の material 順を GUID→URL→内容に変更）

## コンテキスト

collect が取得する記事の一意性判定は `model.Article.URL` に一元化され、過去採用 URL を cache に記録し（`PastURLs`）rundown で除外する。だが URL が「識別子」と「表示用参照リンク」の二役を兼ねるため、2つの問題がある。

1. **フィード（RSS/Atom）**: `<item>` の `<link>` は仕様上任意で、一意性は本来 `<guid>`／Atom `<id>` が担う。だが collect は `item.Link` のみ読み `item.GUID` を無視するため、link 無しフィードでは `Article.URL` が空になり、除外フィルタ・マップキー（`articleByURL`）が空文字で衝突し破綻する。
2. **定期更新ページ（`articles[]`、天気予報等）**: URL が安定なため、内容が更新されても同一 URL 扱いで除外され、更新後の内容を取り込めない。

## 決定

重複判定の識別子を URL から分離し、`Article.DedupKey` を新設する。URL は表示専用（空可）へ降格する。

1. **DedupKey はソース種別ごとに算出し、ソース URL で名前空間化する**: 識別材料はフィードなら `item.GUID`（無ければ正規化した本文）、Web ページなら正規化した本文。最終キーは `sha256(ソースURL + 区切り + 識別材料)`（`sha256:<hex>`）。RSS の `<guid>` は同一フィード内でのみ一意で、フィード間では連番等が衝突し得るため、フィード URL／ページ URL を名前空間に含めて全ソース横断で一意にする。共通ヘルパーを collect に置き両経路から呼ぶ。
2. **識別子の連鎖を URL→DedupKey へ全面移行**: `Article → RundownArticle → ArticleRef(manifest) → ArticleEntry(cache)` まで DedupKey をスレッドし、除外・マージ・選別の各キーを DedupKey にする。cache は `dedup_key` を保存し `PastURLs`→`PastDedupKeys`。
3. **選別 LLM の識別子を `id` へ**: `select.md`／スキーマの `url`/`selected_urls` を `id`/`selected_ids` に改名し DedupKey を授受する。
4. **URL は表示専用**: manifest/slack は URL が空のとき欠落を許容する。

## 結果

- フィードは正規の guid/id で一意判定でき、参照 URL が無くても動作する。
- 更新ページは内容変化で別物となり、更新後を取り込める（内容ハッシュ判定）。
- 既存キャッシュは旧 URL しか持たないため、移行直後の1回だけ既存記事が「新規」扱いで再採用され得る（以後は新キーで安定）。
- 内容ハッシュは軽微な変更でも別物化するため、頻繁に微変動するページは再採用が増える。URL 表示は空になり得るので manifest/slack は欠落を許容する。
- 設定追加は不要（識別はソース種別から自動決定）。

## 検討した代替案

- **URL 識別の維持＋フィードに疑似 URL を付与**: link 無し問題は回避できるが、更新ページ問題が未解決で、guid 軽視は RSS の作法に反する。却下。
- **更新検知に Last-Modified／ETag を使う**: ヘッダ非提供・不正確なページが多く、決定論的な内容ハッシュの方が堅牢でテストも容易。却下。
- **フィードと Web ページで別機構**: 重複実装で保守が困難。統一した DedupKey を採用。
