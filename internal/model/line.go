package model

type Line struct {
	SpeakerRole string `json:"speaker_role"`
	Style       string `json:"style,omitempty"`
	Text        string `json:"text"`
}

type Lines struct {
	Lines []Line `json:"lines"`
}
