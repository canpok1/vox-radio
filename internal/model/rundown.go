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
	AppearanceCount int    `json:"appearance_count"` // その回を含めた出演回数。1 = 初登場（その回に初めて出演）
}

// PastAppearanceCount returns the number of past appearances (excluding this episode).
// This is the LLM-facing value: AppearanceCount - 1, clamped to 0.
func (c RundownCast) PastAppearanceCount() int {
	if c.AppearanceCount <= 1 {
		return 0
	}
	return c.AppearanceCount - 1
}

// CastsForLLM returns a copy of casts with AppearanceCount replaced by PastAppearanceCount()
// for each element, converting from the persisted definition to the LLM-facing definition.
func CastsForLLM(casts []RundownCast) []RundownCast {
	result := make([]RundownCast, len(casts))
	for i, c := range casts {
		result[i] = c
		result[i].AppearanceCount = c.PastAppearanceCount()
	}
	return result
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
