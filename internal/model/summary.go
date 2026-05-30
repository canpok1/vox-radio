package model

type Summary struct {
	URL     string   `json:"url"`
	Summary string   `json:"summary"`
	Points  []string `json:"points"`
}

type Summaries struct {
	Summaries []Summary `json:"summaries"`
}
