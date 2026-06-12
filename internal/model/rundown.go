package model

import "encoding/json"

// NewRundownArticle creates a RundownArticle with Points guaranteed non-nil.
func NewRundownArticle(dedupKey, url, title, summary string, points []string, source, author, published string) RundownArticle {
	return RundownArticle{
		DedupKey: dedupKey, URL: url, Title: title, Summary: summary,
		Points: NonNil(points), Source: source, Author: author, Published: published,
	}
}

// UnmarshalJSON implements json.Unmarshaler to normalize Points to a non-nil empty slice.
func (a *RundownArticle) UnmarshalJSON(data []byte) error {
	type alias RundownArticle
	var raw alias
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	*a = RundownArticle(raw)
	a.Points = NonNil(a.Points)
	return nil
}

type RundownArticle struct {
	DedupKey  string   `json:"dedup_key"`     // 重複判定キー（sha256:hex）
	URL       string   `json:"url,omitempty"` // 表示用リンク（空可）
	Title     string   `json:"title"`
	Summary   string   `json:"summary"`
	Points    []string `json:"points"`
	Source    string   `json:"source,omitempty"`    // 媒体名
	Author    string   `json:"author,omitempty"`    // 著者名
	Published string   `json:"published,omitempty"` // 配信日時（RFC3339）
}

type RundownCorner struct {
	ID                string           `json:"id"`
	Title             string           `json:"title"`
	Flow              string           `json:"flow"`             // フェーズ2で全コーナー分を生成
	SelectionReason   string           `json:"selection_reason"` // フェーズ1の選別理由（記事なしコーナーは空）
	Articles          []RundownArticle `json:"articles"`
	AppearanceCount   int              `json:"appearance_count"`              // その回を含めた扱い回数。1 = 新コーナー（その回が初出）
	LastEpisodeNumber int              `json:"last_episode_number,omitempty"` // 前回このコーナーを扱った回番号。0 = 過去に扱いなし
}

// RundownCast は出演確定したキャスト1人分の情報。
type RundownCast struct {
	CharacterID       string `json:"character_id"`
	Role              string `json:"role"`
	Type              string `json:"type"`                          // "regular" | "guest"
	AppearanceCount   int    `json:"appearance_count"`              // その回を含めた出演回数。1 = 初登場（その回に初めて出演）
	LastEpisodeNumber int    `json:"last_episode_number,omitempty"` // 前回出演した回番号。0 = 過去に出演なし
}

// PastAppearanceCount returns the number of past appearances (excluding this episode).
// This is the LLM-facing value: AppearanceCount - 1, clamped to 0.
func (c RundownCast) PastAppearanceCount() int {
	return max(0, c.AppearanceCount-1)
}

// CastsForLLM returns a copy of casts with AppearanceCount replaced by PastAppearanceCount()
// for each element, converting from the persisted definition to the LLM-facing definition.
// LastEpisodeNumber is an absolute episode number and is propagated unchanged (no boundary
// conversion), mirroring how corners expose last_episode_number.
func CastsForLLM(casts []RundownCast) []RundownCast {
	result := make([]RundownCast, len(casts))
	for i, c := range casts {
		c.AppearanceCount = c.PastAppearanceCount()
		result[i] = c
	}
	return result
}

type Rundown struct {
	Corners []RundownCorner `json:"corners"`
	Casts   []RundownCast   `json:"casts"` // その回の出演者（キャラID昇順、0件でも null でなく []）
}

// CornerMap returns a map from corner ID to its RundownCorner.
func (r Rundown) CornerMap() map[string]RundownCorner {
	m := make(map[string]RundownCorner, len(r.Corners))
	for _, c := range r.Corners {
		m[c.ID] = c
	}
	return m
}
