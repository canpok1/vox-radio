package model

type RundownArticle struct {
	URL     string   `json:"url"`
	Title   string   `json:"title"`
	Summary string   `json:"summary"`
	Points  []string `json:"points"`
}

type RundownCorner struct {
	Title           string           `json:"title"`
	Flow            string           `json:"flow"`             // フェーズ2で全コーナー分を生成
	SelectionReason string           `json:"selection_reason"` // フェーズ1の選別理由（記事なしコーナーは空）
	Articles        []RundownArticle `json:"articles"`
}

// RundownCast は出演確定したキャスト1人分の情報。
type RundownCast struct {
	CharacterID     string `json:"character_id"`
	Role            string `json:"role"`
	Type            string `json:"type"`             // "regular" | "guest"
	AppearanceCount int    `json:"appearance_count"` // 過去の出演エピソード数（今回含まず）。0 = 初登場
}

type Rundown struct {
	Corners []RundownCorner `json:"corners"`
	Casts   []RundownCast   `json:"casts"` // その回の出演者（キャラID昇順、0件でも null でなく []）
}

// CornerMap returns a map from corner title to its RundownCorner.
func (r Rundown) CornerMap() map[string]RundownCorner {
	m := make(map[string]RundownCorner, len(r.Corners))
	for _, c := range r.Corners {
		m[c.Title] = c
	}
	return m
}
