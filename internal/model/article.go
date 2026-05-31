package model

type Article struct {
	URL   string `json:"url"`
	Title string `json:"title"`
	Body  string `json:"body"`
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
