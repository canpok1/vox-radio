package model

import "encoding/json"

// ArticleRef is a reference to an article included in a manifest corner.
type ArticleRef struct {
	DedupKey string `json:"dedup_key"` // 重複判定キー（sha256:hex）
	Title    string `json:"title"`
	URL      string `json:"url,omitempty"` // 表示用リンク（空可）
}

// CornerSummary holds the LLM-generated summary for a single corner.
type CornerSummary struct {
	Summary string
	Points  []string
}

// NewCornerSummary creates a CornerSummary with Points guaranteed non-nil.
func NewCornerSummary(summary string, points []string) CornerSummary {
	return CornerSummary{Summary: summary, Points: NonNil(points)}
}

// ConversationNote is a single piece of noteworthy information from the episode's conversation
// that is not part of the rundown (article facts). Examples: character status updates,
// interactions, reactions, happenings, or ongoing threads.
type ConversationNote struct {
	Category     string   `json:"category"`      // free-form label (e.g. 近況/掛け合い/感想/ハプニング/継続ネタ)
	CharacterIDs []string `json:"character_ids"` // speaker_role IDs involved; empty array if not character-specific
	Note         string   `json:"note"`          // memo text that lets you recall the conversation later
}

// UnmarshalJSON implements json.Unmarshaler to normalize CharacterIDs to a non-nil empty slice.
func (n *ConversationNote) UnmarshalJSON(data []byte) error {
	type alias ConversationNote
	var raw alias
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	*n = ConversationNote(raw)
	n.CharacterIDs = NonNil(n.CharacterIDs)
	return nil
}

// ProgramSummary is the output of the program summarization step.
type ProgramSummary struct {
	Summary           string             `json:"summary"`
	EpisodeTitle      string             `json:"episode_title"`
	ConversationNotes []ConversationNote `json:"conversation_notes"`
}

// UnmarshalJSON implements json.Unmarshaler to normalize ConversationNotes to a non-nil empty slice.
func (p *ProgramSummary) UnmarshalJSON(data []byte) error {
	type alias ProgramSummary
	var raw alias
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	*p = ProgramSummary(raw)
	p.ConversationNotes = NonNil(p.ConversationNotes)
	return nil
}

// ManifestCorner represents a corner in the manifest with its articles.
type ManifestCorner struct {
	ID          string       `json:"id"`
	Title       string       `json:"title"`
	Summary     string       `json:"summary"`
	Points      []string     `json:"points"`
	Articles    []ArticleRef `json:"articles"`
	TargetSec   int          `json:"target_sec,omitempty"`
	SpeechSec   float64      `json:"speech_sec,omitempty"`
	DurationSec float64      `json:"duration_sec,omitempty"`
	CharCount   int          `json:"char_count,omitempty"`
}

// NewManifestCorner creates a ManifestCorner with Points guaranteed non-nil.
func NewManifestCorner(id, title, summary string, points []string, articles []ArticleRef) ManifestCorner {
	return ManifestCorner{
		ID: id, Title: title, Summary: summary,
		Points: NonNil(points), Articles: articles,
	}
}

// CornerTiming holds the total playback duration for a single corner.
type CornerTiming struct {
	ID          string  `json:"id"`
	DurationSec float64 `json:"duration_sec"`
}

// Timeline holds per-corner timing information produced by the mix step.
// Persisted as 06_timeline.json.
type Timeline struct {
	Corners []CornerTiming `json:"corners"`
}

// Map converts the ordered Corners slice to a map keyed by CornerID for fast lookup.
func (t Timeline) Map() map[string]float64 {
	m := make(map[string]float64, len(t.Corners))
	for _, c := range t.Corners {
		m[c.ID] = c.DurationSec
	}
	return m
}

// UnmarshalJSON implements json.Unmarshaler to normalize Points to a non-nil empty slice.
func (c *ManifestCorner) UnmarshalJSON(data []byte) error {
	type alias ManifestCorner
	var raw alias
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	*c = ManifestCorner(raw)
	c.Points = NonNil(c.Points)
	return nil
}

// Manifest is the content manifest output alongside an mp3 episode.
type Manifest struct {
	Title             string             `json:"title"`
	EpisodeNumber     int                `json:"episode_number,omitempty"`
	EpisodeTitle      string             `json:"episode_title,omitempty"`
	Author            string             `json:"author,omitempty"`
	Description       string             `json:"description"`
	Summary           string             `json:"summary"`
	Datetime          string             `json:"datetime"`
	AudioFile         string             `json:"audio_file"`
	Corners           []ManifestCorner   `json:"corners"`
	ConversationNotes []ConversationNote `json:"conversation_notes"`
	Casts             []RundownCast      `json:"casts"`
	Credits           []string           `json:"credits"`
}
