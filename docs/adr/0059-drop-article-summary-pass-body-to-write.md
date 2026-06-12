# 0059. rundown の記事要約を廃止し原文を write に引き渡す

- ステータス: 採用
- 日付: 2026-06-12

## コンテキスト

お便り（リスナー投稿）を番組内で**原文のまま抜粋・読み上げ**したい。だが write（セリフ生成）LLM には rundown の記事要約 `Summary` しか渡らず、原文（`Article.Body`）は rundown 構築時に破棄される。ADR-0015 で write は rundown を唯一の正として参照するため、原文を届けるには rundown に原文を載せる必要がある。加えて `RundownArticle` は `Summary`（一行要約）と `Points`（要点）を併持するが、Points が要約の役割を兼ねるため Summary は冗長だった。

## 決定

記事単位の `Summary` フィールドを廃止し、代わりに**原文（`Body`）を `RundownArticle` に載せて write まで引き渡す**（全コーナー一律）。`Points` は維持する。summarize ステップは Points 生成のため存続し、Body は `Article.Body` をそのまま転記する。お便りコーナーを識別する種別・フラグは導入しない。cache の記事単位 `Summary` は消費箇所がないため削除し、原文は cache には持たない（原文は同一エピソード生成内の rundown→write 間でのみ必要）。

## 結果

- write が記事の原文を直接抜粋・引用でき、お便りの原文読み上げが可能になる。
- `Summary` と `Points` の重複が解消され rundown スキーマが簡潔になる。
- ADR-0015 の「rundown が唯一の正」を維持（原文も rundown 経由で流れる）。
- トレードオフ: write LLM が全記事の原文（信頼境界外テキスト）を受け取るため、要約による言い換えフィルタが外れ、プロンプトインジェクション面が write 段階まで広がる。緩和は collect 境界の決定論的サニタイズ（ADR-0057）と write.md の防御節（body 向けに文言修正）に依存する。
- 破壊的変更: rundown スキーマ・summarize.md の出力スキーマ・cache の記事エントリが変わる。

## 検討した代替案

- **コーナー種別/フラグでお便りコーナーのみ原文保持**: 識別の仕組みと分岐が増え設定・実装が複雑化する。全コーナー一律で要件を満たせるため却下。
- **write が `01_articles.json` を直接読む**: ADR-0015（rundown が唯一の正）に違反し、write が collect 出力に直結するため却下。
- **`Summary` を残し `Body` を併載**: Points と要約が重複したまま原文も増え冗長。Summary 廃止を選択。
