package config

import (
	"strings"
	"time"
)

const (
	// DefaultCharsPerMinute is the default number of characters spoken per minute (= 7文字/秒×60),
	// used to convert length_sec to a target character count when chars_per_minute is not configured.
	DefaultCharsPerMinute = 420
	// DefaultProgramSummaryLength is the default summary length (chars) for program-wide summaries.
	DefaultProgramSummaryLength = 200
	// DefaultProgramTimezone is the default IANA timezone for the program.
	DefaultProgramTimezone = "Asia/Tokyo"
	// DefaultCornerSummaryLength is the default summary length (chars) for per-corner summaries.
	DefaultCornerSummaryLength = 100
	// DefaultAudioQuality is the default audio quality preset ("standard" = VBR V2, ~190kbps).
	DefaultAudioQuality = "standard"
)

// ValidAudioQualityPresets lists all accepted values for program.audio_quality (lowercased).
var ValidAudioQualityPresets = []string{"high", "standard", "low"}

// DurationSecToTargetChars converts a duration in seconds to an approximate target character count.
func DurationSecToTargetChars(sec, charsPerMinute int) int {
	return sec * charsPerMinute / 60
}

// FeedEntry defines a single RSS/Atom feed source with an optional item limit.
type FeedEntry struct {
	URL      string `yaml:"url"`
	MaxItems int    `yaml:"max_items"`
}

// FeedsConfig holds a list of feed sources and individual article URLs.
type FeedsConfig struct {
	Feeds    []FeedEntry `yaml:"feeds"`
	Articles []string    `yaml:"articles"`
}

// SourceConfig defines the data sources for a corner (feeds and individual article URLs).
// It is an alias for FeedsConfig, ensuring the two remain in sync.
type SourceConfig = FeedsConfig

// AudioRef references a jingle or SE asset by type and asset ID.
type AudioRef struct {
	Type string `yaml:"type"` // "jingle" or "se"
	ID   string `yaml:"id"`
}

// CornerDefaults defines program-level default values for corner audio/pause settings.
// A nil CornerDefaults means no defaults (each corner is independent).
// Individual corner fields override defaults when set; empty string or empty AudioRef disables the default.
type CornerDefaults struct {
	BGM           *string   `yaml:"bgm,omitempty"`
	StartAudio    *AudioRef `yaml:"start_audio,omitempty"`
	EndAudio      *AudioRef `yaml:"end_audio,omitempty"`
	StartPauseSec *float64  `yaml:"start_pause_sec,omitempty"`
	EndPauseSec   *float64  `yaml:"end_pause_sec,omitempty"`
}

// CornerConfig defines a fixed corner in the program structure.
type CornerConfig struct {
	ID            string            `yaml:"id"` // コーナーを回をまたいで同定する安定キー（必須・番組内で一意）
	Title         string            `yaml:"title"`
	Content       string            `yaml:"content"`
	Direction     string            `yaml:"direction,omitempty"`
	ScriptNote    string            `yaml:"script_note,omitempty"` // コーナー個別の台本指示（write専用・非公開）
	Cast          map[string]string `yaml:"cast"`
	LengthSec     int               `yaml:"length_sec"`
	SummaryLength int               `yaml:"summary_length,omitempty"`
	Source        *SourceConfig     `yaml:"source,omitempty"`
	StartAudio    *AudioRef         `yaml:"start_audio,omitempty"`
	EndAudio      *AudioRef         `yaml:"end_audio,omitempty"`
	BGM           *string           `yaml:"bgm,omitempty"`
	StartPauseSec *float64          `yaml:"start_pause_sec,omitempty"`
	EndPauseSec   *float64          `yaml:"end_pause_sec,omitempty"`
	Condition     *EpisodeCondition `yaml:"condition,omitempty"` // 出現条件（nil なら毎回出る固定コーナー）
}

// EffectiveSummaryLength returns the configured SummaryLength, falling back to DefaultCornerSummaryLength.
func (c CornerConfig) EffectiveSummaryLength() int {
	if c.SummaryLength <= 0 {
		return DefaultCornerSummaryLength
	}
	return c.SummaryLength
}

// EffectiveBGM returns the BGM key, or "" when nil (not set) or explicitly empty (disabled).
func (c CornerConfig) EffectiveBGM() string {
	if c.BGM == nil {
		return ""
	}
	return *c.BGM
}

// EffectiveStartPauseSec returns the start pause duration, or 0 when nil.
func (c CornerConfig) EffectiveStartPauseSec() float64 {
	if c.StartPauseSec == nil {
		return 0
	}
	return *c.StartPauseSec
}

// EffectiveEndPauseSec returns the end pause duration, or 0 when nil.
func (c CornerConfig) EffectiveEndPauseSec() float64 {
	if c.EndPauseSec == nil {
		return 0
	}
	return *c.EndPauseSec
}

// ProgramConfig holds program-wide settings for content generation.
type ProgramConfig struct {
	// ID is required: it is the cache key (episodes are stored per program.id).
	ID             string `yaml:"id"`
	Title          string `yaml:"title"`
	Author         string `yaml:"author,omitempty"` // MP3 アーティストタグ（TPE1）に埋め込む番組作者名
	Description    string `yaml:"description"`
	Direction      string `yaml:"direction,omitempty"`   // 番組全体の演出指示（direct専用）
	ScriptNote     string `yaml:"script_note,omitempty"` // 番組全体の台本指示（write専用・非公開）
	SummaryLength  int    `yaml:"summary_length,omitempty"`
	Timezone       string `yaml:"timezone,omitempty"`         // IANA tz名。未設定時は DefaultProgramTimezone
	CharsPerMinute int    `yaml:"chars_per_minute,omitempty"` // 台本の文字数換算に使用する1分あたりの文字数。未設定時は DefaultCharsPerMinute
	AudioQuality   string `yaml:"audio_quality,omitempty"`    // 音質プリセット: "high" / "standard" / "low"。未設定時は DefaultAudioQuality
}

// EffectiveSummaryLength returns the configured SummaryLength, falling back to DefaultProgramSummaryLength.
func (p ProgramConfig) EffectiveSummaryLength() int {
	if p.SummaryLength <= 0 {
		return DefaultProgramSummaryLength
	}
	return p.SummaryLength
}

// EffectiveCharsPerMinute returns the configured CharsPerMinute, falling back to DefaultCharsPerMinute.
func (p ProgramConfig) EffectiveCharsPerMinute() int {
	if p.CharsPerMinute <= 0 {
		return DefaultCharsPerMinute
	}
	return p.CharsPerMinute
}

// EffectiveAudioQuality returns the lowercased AudioQuality, falling back to DefaultAudioQuality.
func (p ProgramConfig) EffectiveAudioQuality() string {
	if p.AudioQuality == "" {
		return DefaultAudioQuality
	}
	return strings.ToLower(p.AudioQuality)
}

// EffectiveTimezone returns Timezone, falling back to DefaultProgramTimezone.
func (p ProgramConfig) EffectiveTimezone() string {
	if p.Timezone == "" {
		return DefaultProgramTimezone
	}
	return p.Timezone
}

// Location resolves EffectiveTimezone() to *time.Location via time.LoadLocation.
// Returns an error if the timezone name is invalid.
func (p ProgramConfig) Location() (*time.Location, error) {
	return time.LoadLocation(p.EffectiveTimezone())
}
