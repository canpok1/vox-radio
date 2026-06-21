# 0091. コーナーの source に links / text タイプを追加する

- ステータス: 採用
- 日付: 2026-06-21
- 関連: [0087-corner-source-as-typed-array.md](0087-corner-source-as-typed-array.md)（type 付き配列の基盤）, [0057-feed-prompt-injection-defense.md](0057-feed-prompt-injection-defense.md)（収集テキストの sanitize）, [0058-decouple-article-dedup-key-from-url.md](0058-decouple-article-dedup-key-from-url.md)（内容ベース DedupKey）

## コンテキスト

ADR-0087 でコーナーの `source` は `type` 付き配列（`feed` / `web`）に再設計された。運用するなかで次の不便が出てきた。

- 参考ページの URL を多数まとめて渡したいとき、`web` エントリを URL ごとに列挙する必要があり冗長。`feed` の `file://` でローカルファイルを使う手もあるが、RSS/Atom の XML を手書きするのは手間。
- RSS や Web ページに依らない任意の参考情報（手元のメモ・調査テキストなど）を番組素材として使いたいが、現状はテキストをそのまま渡す手段がない。

`source` は ADR-0087 で「`type` の値を増やすだけで拡張できる」設計になっており、この枠組みに新タイプを追加して解決する。

## 決定

`source` に 2 つのタイプを追加する。どちらもローカルテキストファイルを新フィールド `path` で指定する。

```yaml
source:
  - type: links            # URL を 1 行 1 つ並べたテキストファイル
    path: refs/urls.txt
  - type: text             # 参考情報テキストファイルの中身をそのまま記事化
    path: refs/note.txt
    title: 今週のメモ        # 任意。省略時は拡張子を除いたファイル名
```

### 共通

- **ファイル指定は新フィールド `path`** を使う。`feed` / `web` の `url` とは別フィールドとし、spec ファイルのディレクトリを基準に相対解決する（絶対パスも可）。`file://` を手書きせずに済ませることで「手軽さ」を優先する。`path` の解決は config 層（ロード時）で行い、絶対パス化した値を構造体で gather に渡す。
- 収集テキストは既存の `feed` / `web` と同様に**プロンプトインジェクション sanitize の対象**とする（ADR-0057）。`text` は利用者自身のファイルだが、一貫性と安全側を優先する（誤検知時は `prompt_injection` ポリシーで調整可能）。
- 各記事には DedupKey を付与する（ADR-0058）。DedupKey は `sha256(namespace + "\0" + material)` で、namespace はソース識別子（衝突回避）、material は識別内容。タイプごとの namespace / material は下記のとおり定める。

### `links`

- `path` のファイルを行単位で読み、各行を trim する。空行および `#` で始まる行（コメント）はスキップする。
- 各行を URL とみなし、`web` と同じ経路（`fetchArticle`：HTML 取得・タイトル/本文抽出）で記事化する。`https://` のほか `file://` も使える。
- **DedupKey の namespace は links ファイルのパス、material は各行の URL** とする（`feed` の「namespace=フィードURL / material=GUID」に対応づけた構造。links ファイル＝フィード、各行 URL＝GUID）。これにより、利用者が手動更新する URL リストにおいて、取得したページ内容が変動（ドリフト）しても**同一 URL は同一キー**となり、過去回除外が安定して効く。別の links ファイルに同じ URL があれば namespace が異なるため別物として扱う（feed 間分離と同じ）。
- `url` / `max_items` / `title` は指定不可。

### `text`

- `path` のファイル内容を HTML パースせずプレーンテキストのまま記事本文（`Article.Body`）に格納する。
- 記事タイトルは任意の `title` フィールド、省略時は拡張子を除いたファイル名とする。表示用リンク（`URL`）は持たない。
- **DedupKey の namespace はファイルパス、material はタイトル＋本文（内容ベース）** とする。1 ファイル＝1 リファレンスであり、内容を更新したら「新しい参考情報」として再採用できる方が自然なため（内容不変＝同一キーで重複回避、内容更新＝別キーで再採用可）。links と異なり外部ページのドリフト問題は起きない。
- `url` / `max_items` は指定不可。`title` は任意。

### バリデーション

設定ロード時に `type` ごとに必須/禁止フィールドを検証する。

| type | 必須 | 禁止 |
|---|---|---|
| `feed` | `url` | `path` `title` |
| `web` | `url` | `path` `title` `max_items` |
| `links` | `path` | `url` `max_items` `title` |
| `text` | `path` | `url` `max_items` |

## 結果

- 参考ページの URL 一覧や任意の参考テキストを、最小の記述でコーナー素材に取り込めるようになり、番組素材の幅が広がる。
- `feed` / `web`（外部 URL を `url` で指定）と `links` / `text`（ローカルファイルを `path` で指定）でフィールドが分かれ、種別ごとの意味が明確になる。
- 既存の sanitize・DedupKey の仕組みを再利用するため、防御・重複判定の一貫性は保たれる。

## 検討した代替案

- **`url` フィールドに `file://` で指定して既存と統一**: フィールドが増えず一貫するが、`file://` を手書きする手間が残り、本 ADR の動機（手軽さ）に反するため却下。`path` を別フィールドにした。
- **`links` のファイル各行を `feed` として扱う**: フィード URL を並べたいケースもあり得るが、本要望は「参考ページ（Web ページ）の一覧」であり、各行を `web` 同様に単一ページ取得する方が直感的。フィードを複数使いたい場合は従来どおり `feed` エントリを並べる。
- **`links` の DedupKey material に取得した タイトル＋本文（`web` と同じ `HTMLDedupKey`）を使う**: namespace を各行 URL にする案だが、外部ページの内容が変動するたびにキーが変わり、手動更新する URL リストの過去回除外が崩れる。利用者が同定したいのは「どの URL を載せたか」であり、URL を安定識別子（GUID 相当）として material に採る方が運用に合うため却下。
- **`text` のタイトルを先頭行から取る**: Markdown 見出し風に書けるが暗黙ルールになり分かりにくいため、明示的な `title`（省略時ファイル名）を採用した。
- **`text` を sanitize 対象外にする**: 利用者の自前ファイルなので誤検知を避けられるが、収集テキストの防御を一律に保つ一貫性を優先し対象に含めた。
