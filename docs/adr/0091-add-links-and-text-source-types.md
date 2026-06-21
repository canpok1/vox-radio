# 0091. コーナーの source に links / text タイプを追加する

- ステータス: 採用
- 日付: 2026-06-21
- 関連: [0087-corner-source-as-typed-array.md](0087-corner-source-as-typed-array.md)（type 付き配列の基盤）, [0057-feed-prompt-injection-defense.md](0057-feed-prompt-injection-defense.md)（収集テキストの sanitize）
- 一部改訂: [0058-decouple-article-dedup-key-from-url.md](0058-decouple-article-dedup-key-from-url.md)（feed の DedupKey material 順を見直す）

## コンテキスト

ADR-0087 でコーナーの `source` は `type` 付き配列（`feed` / `web`）になった。運用で次の不便が出た。

- 参考ページ URL を多数渡すのに `web` の列挙は冗長。`feed` の `file://` でも RSS/Atom を手書きするのは手間。
- RSS/Web に依らない任意の参考テキストを素材にする手段がない。

## 決定

`source` に `links`（URL 一覧テキスト）と `text`（参考テキスト本文）を追加する。ローカルファイルを新フィールド `path` で指定する（spec 基準で相対解決）。`url` と別にして `file://` 手書きを避ける。

```yaml
source:
  - type: links            # URL を 1 行 1 つ並べたテキストファイル
    path: refs/urls.txt
  - type: text             # 参考情報テキストの中身をそのまま記事化
    path: refs/note.txt
    title: 今週のメモ        # 任意。省略時は拡張子を除いたファイル名
```

- `links`: 各行を trim し、空行と `#` 始まりの行はスキップ。各行 URL を `web` と同じ `fetchArticle` で記事化する（`https://` / `file://` 可）。
- `text`: ファイル内容を HTML パースせず本文（`Article.Body`）に入れる。タイトルは任意 `title`、省略時は拡張子なしのファイル名。表示用 URL なし。
- 収集テキストは `feed` / `web` 同様に sanitize 対象（ADR-0057）。`text` も一貫性のため含める。

### バリデーション

| type | 必須 | 禁止 |
|---|---|---|
| `feed` | `url` | `path` `title` |
| `web` | `url` | `path` `title` `max_items` |
| `links` | `path` | `url` `max_items` `title` |
| `text` | `path` | `url` `max_items` |

### DedupKey（ADR-0058 の一部改訂を含む）

DedupKey は `sha256(namespace + 区切り + material)`。タイプ別は下表。あわせて `feed` の material 順を `GUID → 内容` から **`GUID → URL → 内容`** に変更し、安定度順（GUID ＞ URL ＞ 内容）で `links` と揃える。

| type | namespace | material |
|---|---|---|
| `feed` | フィード URL | `GUID → item.Link → 正規化(タイトル+本文)` |
| `web` | ページ URL | 正規化(タイトル+本文) |
| `links` | links ファイルのパス | 各行の URL |
| `text` | ファイルパス | 正規化(タイトル+本文) |

- `feed` / `links` はコンテナを namespace、アイテム URL を material とする。内容がドリフトしても同一 URL は同一キーで過去回除外が安定する。
- ADR-0058 が feed を内容ベースにしたのはリンク無しフィード対策で、内容更新での再採用は `web` 向けの動機。`web` は不変で核心は保たれる。`item.Link` 空なら内容にフォールバック。
- `text` は内容ベース。内容更新＝別キー＝再採用可（`web` と同じ）。

## 結果

- URL 一覧や参考テキストを最小記述で素材化でき、番組の幅が広がる。`url` / `path` で意味も明確になる。
- 既存の sanitize・DedupKey を再利用し一貫性を保つ。feed の material 順変更で GUID 無しフィードのドリフト重複が減る。
- 移行影響: GUID 無し・URL 有りの既存記事はキーが変わり直後の1回だけ再採用され得る（ADR-0058 も許容済み）。同一 URL feed の内容更新は再採用されない。

## 検討した代替案

- **`url` に `file://` で統一**: `file://` 手書きの手間が残るため `path` を別フィールドにした。
- **`links` 各行を `feed` 扱い**: 要望は Web ページの一覧で、単一ページ取得が直感的。複数フィードは `feed` を並べる。
- **`links` の material に取得本文を使う**: 内容ドリフトでキーが変わり除外が崩れる。同定対象は「載せた URL」なので URL を採った。
- **`text` のタイトルを先頭行 / sanitize 対象外**: 暗黙ルール化や防御低下を避けて却下した。
