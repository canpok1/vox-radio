package model

type Article struct {
	URL   string `json:"url"`
	Title string `json:"title"`
	Body  string `json:"body"`
}

type Articles struct {
	Articles []Article `json:"articles"`
}
