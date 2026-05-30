# 0008. collect パッケージの RSS/HTML パースライブラリ選定

- ステータス: 採用
- 日付: 2026-05-30

## コンテキスト

Issue #8 で RSS フィード・個別 URL から記事本文を収集する `internal/collect` パッケージを実装するにあたり、RSS パースと HTML 本文抽出の方法を選択する必要があった。プロジェクトはすでに `gopkg.in/yaml.v3` と `github.com/xeipuuv/gojsonschema` を使っており、外部依存の追加は最小限に留めたい。

## 決定

- **RSS パース**: 標準ライブラリ `encoding/xml` を使い、RSS 2.0 の構造体（`rssFeed/rssChannel/rssItem`）に直接アンマーシャルする。
- **HTML パース**: `golang.org/x/net/html` を追加し、DOM ツリーを走査する。本文抽出は `<article>` → `<main>` → `<body>` の優先順で最初に見つかった要素のテキストノードを連結する。
- RSS アイテムの `<description>` に HTML が含まれる場合は同ライブラリで剥ぎ取り、プレーンテキストを `body` として使う。

## 結果

`encoding/xml` は標準ライブラリなので依存追加なしに RSS 2.0 を処理できる。`golang.org/x/net/html` は Go 公式サブリポジトリであり、追加依存として許容範囲と判断した。本文抽出のヒューリスティックはシンプルだが、セマンティック HTML（`<article>`/`<main>`）を持つページでは nav・footer を自然に除外できる。構造化されていないページでは `<body>` 全体が fallback となりノイズが混入しうる。

## 検討した代替案

- **`github.com/mmcdole/gofeed`**: RSS/Atom/JSON フィードを一括対応するが、外部依存が増えることと今回は RSS 2.0 のみで十分なため採用しなかった。
- **`github.com/go-shiori/go-readability`**: Mozilla Readability 相当の精度を持つが、間接依存が多く導入コストが高いため採用しなかった。
- **`golang.org/x/net/html` + goquery**: goquery はより柔軟なセレクタを提供するが、`golang.org/x/net/html` の直接走査で要件を満たせるため不要と判断した。
