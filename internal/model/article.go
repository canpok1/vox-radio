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
