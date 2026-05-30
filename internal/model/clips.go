package model

// ClipMeta holds metadata for a single synthesized speech clip
type ClipMeta struct {
	Index       int     `json:"index"`
	File        string  `json:"file"`
	DurationSec float64 `json:"duration_sec"`
	SpeakerRole string  `json:"speaker_role"`
	Text        string  `json:"text"`
}

// ClipsMeta holds all synthesized clips and their metadata
type ClipsMeta struct {
	Clips []ClipMeta `json:"clips"`
}
