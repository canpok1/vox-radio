package model

type Summary struct {
	URL     string   `json:"url"`
	Summary string   `json:"summary"`
	Points  []string `json:"points"`
}

type CornerSummaries struct {
	CornerTitle string    `json:"corner_title"`
	Summaries   []Summary `json:"summaries"`
}

type Summaries struct {
	Corners []CornerSummaries `json:"corners"`
}
