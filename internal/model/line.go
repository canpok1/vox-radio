package model

type Line struct {
	SpeakerRole string `json:"speaker_role"`
	Style       string `json:"style,omitempty"`
	Intonation  string `json:"intonation,omitempty"`
	Pitch       string `json:"pitch,omitempty"`
	Speed       string `json:"speed,omitempty"`
	Text        string `json:"text"`
}

type Lines struct {
	Lines []Line `json:"lines"`
}

// CornerLines holds the lines and direction for one corner.
type CornerLines struct {
	Title     string `json:"title"`
	Direction string `json:"direction,omitempty"`
	Lines     []Line `json:"lines"`
}

// ScriptLines is the root structure of 03_lines.json.
type ScriptLines struct {
	Corners []CornerLines `json:"corners"`
}
