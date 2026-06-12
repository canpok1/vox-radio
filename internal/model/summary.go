package model

type Summary struct {
	URL    string   `json:"url"`
	Points []string `json:"points"`
}

type CornerSummaries struct {
	CornerTitle string    `json:"corner_title"`
	Summaries   []Summary `json:"summaries"`
}

type Summaries struct {
	Corners []CornerSummaries `json:"corners"`
}

// CornerMap returns a map from corner title to its summaries.
func (s Summaries) CornerMap() map[string][]Summary {
	m := make(map[string][]Summary, len(s.Corners))
	for _, cs := range s.Corners {
		m[cs.CornerTitle] = cs.Summaries
	}
	return m
}
