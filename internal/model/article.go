package model

type Article struct {
	DedupKey    string `json:"dedup_key"`     // 重複判定キー（sha256:hex）。gather が設定する
	URL         string `json:"url,omitempty"` // 表示用リンク（空可）
	Title       string `json:"title"`
	Description string `json:"description,omitempty"` // フィード由来のテキスト（RSS/Atom content/description）
	Body        string `json:"body,omitempty"`        // 記事ページ直接取得のテキスト（fetchArticle）
	Source      string `json:"source,omitempty"`      // 媒体名（RSS feed.Title）
	Author      string `json:"author,omitempty"`      // 著者名（ベストエフォート）
	Published   string `json:"published,omitempty"`   // 配信日時（RFC3339・番組TZ変換済み）
}

// Text returns the effective article text: Body if non-empty, otherwise Description.
func (a Article) Text() string { return textOf(a.Body, a.Description) }

// textOf returns body if non-empty, otherwise description.
func textOf(body, description string) string {
	if body != "" {
		return body
	}
	return description
}

type CornerArticles struct {
	CornerTitle string    `json:"corner_title"`
	Articles    []Article `json:"articles"`
}

type Articles struct {
	Corners []CornerArticles `json:"corners"`
}

// CornerMap returns a map from corner title to its articles.
func (a Articles) CornerMap() map[string][]Article {
	m := make(map[string][]Article, len(a.Corners))
	for _, ca := range a.Corners {
		m[ca.CornerTitle] = ca.Articles
	}
	return m
}
