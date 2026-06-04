package model

type RundownArticle struct {
	URL     string   `json:"url"`
	Title   string   `json:"title"`
	Summary string   `json:"summary"`
	Points  []string `json:"points"`
}

type RundownCorner struct {
	Title    string           `json:"title"`
	Flow     string           `json:"flow"`
	Articles []RundownArticle `json:"articles"`
}

// RundownGuest は出演確定したゲスト1人分の情報。
type RundownGuest struct {
	CharacterID string `json:"character_id"`
	Role        string `json:"role"`
}

type Rundown struct {
	Corners []RundownCorner `json:"corners"`
	Guests  []RundownGuest  `json:"guests"` // 出演確定ゲスト（キャラID昇順、0件でも null でなく []）
}

// CornerMap returns a map from corner title to its RundownCorner.
func (r Rundown) CornerMap() map[string]RundownCorner {
	m := make(map[string]RundownCorner, len(r.Corners))
	for _, c := range r.Corners {
		m[c.Title] = c
	}
	return m
}
