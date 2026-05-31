package model

type SegmentType string

const (
	SegmentTypeSpeech SegmentType = "speech"
	SegmentTypeSE     SegmentType = "se"
	SegmentTypeBGM    SegmentType = "bgm"
	SegmentTypeJingle SegmentType = "jingle"
	SegmentTypePause  SegmentType = "pause"
)

type ScriptSegment struct {
	Type        SegmentType `json:"type"`
	SpeakerRole string      `json:"speaker_role,omitempty"`
	Style       string      `json:"style,omitempty"`
	Intonation  string      `json:"intonation,omitempty"`
	Pitch       string      `json:"pitch,omitempty"`
	Speed       string      `json:"speed,omitempty"`
	Text        string      `json:"text,omitempty"`
	AssetName   string      `json:"asset_name,omitempty"`   // se/bgm/jingle のアセットキー。bgm かつ空 = 停止
	DurationSec float64     `json:"duration_sec,omitempty"` // pause セグメントの無音秒数
}

type Script struct {
	Segments []ScriptSegment `json:"segments"`
}
