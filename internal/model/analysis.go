package model

// Analysis holds the post-generation analysis for a single episode (ADR-0098).
type Analysis struct {
	Metrics  AnalysisMetrics   `json:"metrics"`
	Findings []AnalysisFinding `json:"findings"`
	Patterns []AnalysisPattern `json:"patterns"`
}

// AnalysisMetrics holds mechanically aggregated indicators that need no LLM judgement.
type AnalysisMetrics struct {
	Corners              []AnalysisCornerMetrics  `json:"corners"`
	Speakers             []AnalysisSpeakerMetrics `json:"speakers"`
	ProofreadCorrections int                      `json:"proofread_corrections"`
}

// AnalysisCornerMetrics compares a corner's target character count against what was actually
// written, and breaks the actual length down into speech vs non-speech so an effective speech
// rate (CharsPerSec) can be read directly instead of inferred from a length polluted by jingles/SE/pauses.
type AnalysisCornerMetrics struct {
	ID                 string  `json:"id"`
	TargetChars        int     `json:"target_chars"`
	ActualChars        int     `json:"actual_chars"`
	LineCount          int     `json:"line_count"`
	ActualLengthSec    float64 `json:"actual_length_sec"`
	SpeechLengthSec    float64 `json:"speech_length_sec"`
	NonSpeechLengthSec float64 `json:"non_speech_length_sec"`
	CharsPerSec        float64 `json:"chars_per_sec"` // ActualChars / SpeechLengthSec; 0 when SpeechLengthSec is 0
}

// AnalysisSpeakerMetrics holds per-character line and character counts for the episode.
type AnalysisSpeakerMetrics struct {
	CharacterID string `json:"character_id"`
	LineCount   int    `json:"line_count"`
	CharCount   int    `json:"char_count"`
}

// AnalysisFinding is one qualitative issue found in the episode by the LLM.
type AnalysisFinding struct {
	Aspect   string `json:"aspect"`
	Severity string `json:"severity"` // "high" | "medium" | "low"
	Detail   string `json:"detail"`
	Evidence string `json:"evidence,omitempty"`
}

// AnalysisPattern records a structural habit of this episode's dialogue. Unlike a finding it is
// not a defect on its own; retro compares patterns across episodes to spot ones that never vary.
type AnalysisPattern struct {
	Aspect string `json:"aspect"` // つかみ / 振り方 / 掛け合い / オチ など
	Detail string `json:"detail"`
}
