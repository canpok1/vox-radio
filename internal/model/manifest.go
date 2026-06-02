package model

// ArticleRef is a reference to an article included in a manifest corner.
type ArticleRef struct {
	Title string `json:"title"`
	URL   string `json:"url"`
}

// CornerSummary holds the LLM-generated summary for a single corner.
type CornerSummary struct {
	Summary string
	Points  []string
}

// ConversationNote is a single piece of noteworthy information from the episode's conversation
// that is not part of the rundown (article facts). Examples: character status updates,
// interactions, reactions, happenings, or ongoing threads.
type ConversationNote struct {
	Category     string   `json:"category"`      // free-form label (e.g. 近況/掛け合い/感想/ハプニング/継続ネタ)
	CharacterIDs []string `json:"character_ids"` // speaker_role IDs involved; empty array if not character-specific
	Note         string   `json:"note"`          // memo text that lets you recall the conversation later
}

// ProgramSummary is the output of the program summarization step.
type ProgramSummary struct {
	Summary           string             `json:"summary"`
	EpisodeTitle      string             `json:"episode_title"`
	ConversationNotes []ConversationNote `json:"conversation_notes"`
}

// ManifestCorner represents a corner in the manifest with its articles.
type ManifestCorner struct {
	Title    string       `json:"title"`
	Summary  string       `json:"summary"`
	Points   []string     `json:"points"`
	Articles []ArticleRef `json:"articles"`
}

// Manifest is the content manifest output alongside an mp3 episode.
type Manifest struct {
	Title             string             `json:"title"`
	EpisodeNumber     int                `json:"episode_number,omitempty"`
	EpisodeTitle      string             `json:"episode_title,omitempty"`
	Description       string             `json:"description"`
	Summary           string             `json:"summary"`
	Datetime          string             `json:"datetime"`
	AudioFile         string             `json:"audio_file"`
	Corners           []ManifestCorner   `json:"corners"`
	ConversationNotes []ConversationNote `json:"conversation_notes"`
}
