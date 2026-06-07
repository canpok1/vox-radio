package model

type Line struct {
	SpeakerRole string `json:"speaker_role"`
	Style       string `json:"style,omitempty"`
	Intonation  string `json:"intonation,omitempty"`
	Pitch       string `json:"pitch,omitempty"`
	Speed       string `json:"speed,omitempty"`
	Text        string `json:"text"`
}

// Lines is the LLM response envelope used by the write step.
type Lines struct {
	Lines []Line `json:"lines"`
}

// CornerAudio holds a boundary audio reference for a corner (jingle or SE).
type CornerAudio struct {
	Type      SegmentType `json:"type"`
	AssetName string      `json:"asset_name"`
}

// CornerLines holds the lines and direction for one corner.
// Asset fields (StartAudio, EndAudio, BGM) and pause fields (StartPauseSec, EndPauseSec)
// are transferred from CornerConfig and persisted in 03_lines.json for deterministic segment injection.
type CornerLines struct {
	Title         string       `json:"title"`
	Direction     string       `json:"direction,omitempty"`
	Lines         []Line       `json:"lines"`
	StartAudio    *CornerAudio `json:"start_audio,omitempty"`
	EndAudio      *CornerAudio `json:"end_audio,omitempty"`
	BGM           string       `json:"bgm,omitempty"`
	StartPauseSec float64      `json:"start_pause_sec,omitempty"`
	EndPauseSec   float64      `json:"end_pause_sec,omitempty"`
}

// ScriptLines is the root structure of 03_lines.json.
type ScriptLines struct {
	Direction string        `json:"direction,omitempty"` // 番組全体の演出指示（direct専用）
	Corners   []CornerLines `json:"corners"`
}

// TotalLines returns the total number of lines across all corners.
func (s ScriptLines) TotalLines() int {
	total := 0
	for _, c := range s.Corners {
		total += len(c.Lines)
	}
	return total
}
