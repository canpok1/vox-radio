package model

type Article struct {
	DedupKey  string `json:"dedup_key"`     // 重複判定キー（sha256:hex）。collect が設定する
	URL       string `json:"url,omitempty"` // 表示用リンク（空可）
	Title     string `json:"title"`
	Body      string `json:"body"`
	Source    string `json:"source,omitempty"`    // 媒体名（RSS feed.Title）
	Author    string `json:"author,omitempty"`    // 著者名（ベストエフォート）
	Published string `json:"published,omitempty"` // 配信日時（RFC3339・番組TZ変換済み）
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
