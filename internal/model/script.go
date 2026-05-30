package model

type SegmentType string

const (
	SegmentTypeSpeech SegmentType = "speech"
	SegmentTypeSE     SegmentType = "se"
)

type ScriptSegment struct {
	Type        SegmentType `json:"type"`
	SpeakerRole string      `json:"speaker_role,omitempty"`
	Text        string      `json:"text,omitempty"`
	SEName      string      `json:"se_name,omitempty"`
}

type Script struct {
	Segments []ScriptSegment `json:"segments"`
}
