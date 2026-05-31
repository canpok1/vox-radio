package model

type SegmentType string

const (
	SegmentTypeSpeech SegmentType = "speech"
	SegmentTypeSE     SegmentType = "se"
	SegmentTypeJingle SegmentType = "jingle"
)

type ScriptSegment struct {
	Type        SegmentType `json:"type"`
	SpeakerRole string      `json:"speaker_role,omitempty"`
	Style       string      `json:"style,omitempty"`
	Text        string      `json:"text,omitempty"`
	SEName      string      `json:"se_name,omitempty"`
	AssetName   string      `json:"asset_name,omitempty"`
}

type Script struct {
	Segments []ScriptSegment `json:"segments"`
}
