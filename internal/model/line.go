package model

type Line struct {
	SpeakerRole string `json:"speaker_role"`
	Text        string `json:"text"`
}

type Lines struct {
	Lines []Line `json:"lines"`
}
