package model

type Corner struct {
	Title       string   `json:"title"`
	Topic       string   `json:"topic"`
	Points      []string `json:"points"`
	TargetChars int      `json:"target_chars"`
	SummaryURLs []string `json:"summary_urls"`
}

type Rundown struct {
	Corners []Corner `json:"corners"`
}
