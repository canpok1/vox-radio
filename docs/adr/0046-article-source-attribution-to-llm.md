# 0046. 記事の出典（サイト名・著者名）を rundown 経由で生成 LLM に渡す

- ステータス: 採用
- 日付: 2026-06-06

## コンテキスト

記事を紹介する台本（セリフ）で、記事の出典（サイト名・著者名）を紹介できるようにしたい（Issue #325）。

現状、生成 LLM に渡る記事情報は `RundownArticle{URL, Title, Summary, Points}` のみで、出典にあたるサイト名（媒体名）・著者名がデータ構造に存在しない。collect の RSS 取得（`internal/collect/rss.go` の `fetchFeed`）は `item.Title/Link/Content` のみを使い、`feed.Title`（サイト名）・`item.Author`/`item.Authors`（著者）を取得せず破棄している。そのため write 段階（`write.md` の `{{articles}}`）に出典が届かず、台本で出典紹介ができない。

記事情報の正は rundown（ADR-0015）であり、メタ情報を rundown 経由で生成 LLM に渡す前例として ADR-0037（出演回数）がある。本 ADR では出典をどのデータに持たせ、どこから取得し、どの経路で LLM に渡すかを決める。

## 決定

- **出典項目はサイト名・著者名の2つ**とし、記事 URL/ドメインはセリフに含めない（`.claude/rules/config-prompt.md` の「URL 等の長い文字列はセリフに含めない」と整合）。
- **サイト名は RSS の `feed.Title` から自動取得する**。設定（`FeedEntry`）での静的指定は導入しない。
- **著者名はベストエフォート取得**とする。取得元は `item.Authors[0].Name`（非空優先）→ `item.Author.Name` の順、前後空白は trim。取得できた記事のみ紹介し、無ければ省略する。
- **email 形式の著者は省略**する（`@` を含む値は RSS の author 要素にありがちな email 形式で読み上げに不向きなため空扱い）。
- **rundown へ流す**: `model.Article` と `model.RundownArticle` に `Source`/`Author`（`json:",omitempty"`）を追加し、`Title` と同じ経路（`rundown.go` の `articleByURL`）で `RundownArticle` へ伝播する。write には `{{articles}}` の JSON マーシャルで自動的に渡し、`write.md` には「出典があれば自然に紹介・無ければ省略」を情報提供として明記する。
- **select・summarize へは配線しない**（選別・要約は出典を必要としない）。
- **直接 URL 記事（`source.articles:`）は出典なし**（フィードを介さずサイト名・著者が取れないため空のまま）。
- **読み変換は既存の direct ステップ（ADR-0027）に委ねる**。サイト名・著者名は英字を含みうるが、write 側に新たな読み変換処理は追加しない。

## 結果

**良い影響**: 台本で記事の出典を紹介できる。`feed.Title` 自動取得のため設定追記が不要で、既存フィード設定のまま機能する。`omitempty` により出典が無い記事は中間生成物・プロンプト JSON を汚さず、後方互換も保たれる。出典を rundown に焼き込むため単独 `script` コマンドや再実行でも同じ出典が得られる（ADR-0015 の「rundown が正」と一貫）。

**トレードオフ**: 著者名はフィードに著者情報がある場合のみで、多くのフィードでは空になる。RSS の author 要素が email のみのフィードでは著者を省略する（誤読・読み上げ回避を優先）。直接 URL 記事には出典が付かない。

## 検討した代替案

- **サイト名を設定（`FeedEntry`）で静的に手動指定する**: 表示を完全制御できるが、全フィードへの記入が必要で運用負荷が高い。`feed.Title` 自動取得で十分なため却下。
- **記事 URL/ドメインを出典としてセリフに含める**: URL 等の長い文字列をセリフに含めない方針（config-prompt ルール）と整合せず却下。
- **直接 URL 記事の媒体名を HTML（`<title>`/`og:site_name`）から取得する**: feed 限定方針からスコープが広がり、抽出の信頼性も低いため今回は見送り。
