package synth

// AudioQuery represents the audio synthesis query for VOICEVOX /audio_query and /synthesis endpoints
type AudioQuery struct {
	AccentPhrases      []AccentPhrase `json:"accent_phrases"`
	SpeedScale         float64        `json:"speedScale"`
	PitchScale         float64        `json:"pitchScale"`
	IntonationScale    float64        `json:"intonationScale"`
	VolumeScale        float64        `json:"volumeScale"`
	PrePhonemeLength   float64        `json:"prePhonemeLength"`
	PostPhonemeLength  float64        `json:"postPhonemeLength"`
	OutputSamplingRate int            `json:"outputSamplingRate"`
	OutputStereo       bool           `json:"outputStereo"`
	Kana               string         `json:"kana,omitempty"`
}

// AccentPhrase represents an accent phrase in an AudioQuery
type AccentPhrase struct {
	Moras           []Mora `json:"moras"`
	Accent          int    `json:"accent"`
	PauseMora       *Mora  `json:"pause_mora"`
	IsInterrogative bool   `json:"is_interrogative"`
}

// Mora represents a phonetic unit within an accent phrase
type Mora struct {
	Text            string   `json:"text"`
	Consonant       *string  `json:"consonant"`
	ConsonantLength *float64 `json:"consonant_length"`
	Vowel           string   `json:"vowel"`
	VowelLength     float64  `json:"vowel_length"`
	Pitch           float64  `json:"pitch"`
}
