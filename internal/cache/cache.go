package cache

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/canpok1/vox-radio/internal/model"
)

// ArticleEntry holds article data for a single episode history entry.
type ArticleEntry struct {
	Title   string   `json:"title"`
	URL     string   `json:"url"`
	Summary string   `json:"summary"`
	Points  []string `json:"points"`
}

// CornerEntry holds corner data for a single episode history entry.
type CornerEntry struct {
	Title    string         `json:"title"`
	Summary  string         `json:"summary"`
	Points   []string       `json:"points"`
	Articles []ArticleEntry `json:"articles"`
}

// Entry represents a single episode in the JSONL history file.
type Entry struct {
	ProgramID         string                   `json:"program_id"`
	Datetime          string                   `json:"datetime"`
	EpisodeNumber     int                      `json:"episode_number,omitempty"`
	EpisodeTitle      string                   `json:"episode_title,omitempty"`
	Title             string                   `json:"title"`
	Summary           string                   `json:"summary"`
	Description       string                   `json:"description,omitempty"`
	AudioFile         string                   `json:"audio_file,omitempty"`
	Bytes             int64                    `json:"bytes,omitempty"`
	DurationSec       int                      `json:"duration_sec,omitempty"`
	Corners           []CornerEntry            `json:"corners"`
	ConversationNotes []model.ConversationNote `json:"conversation_notes"`
}

// Manager handles JSONL cache file operations for a single program.
type Manager struct {
	path string
}

// New creates a Manager for the given cache file path.
func New(path string) *Manager {
	return &Manager{path: path}
}

// Load reads all entries from the cache file.
// Returns an empty slice if the file does not exist.
func (m *Manager) Load() ([]Entry, error) {
	f, err := os.Open(m.path)
	if os.IsNotExist(err) {
		return make([]Entry, 0), nil
	}
	if err != nil {
		return nil, fmt.Errorf("open cache: %w", err)
	}
	defer func() { _ = f.Close() }()

	var entries []Entry
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}
		var e Entry
		if err := json.Unmarshal(line, &e); err != nil {
			return nil, fmt.Errorf("unmarshal cache entry: %w", err)
		}
		entries = append(entries, e)
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("scan cache: %w", err)
	}
	if entries == nil {
		return make([]Entry, 0), nil
	}
	return entries, nil
}

// Append loads existing entries, adds the new entry, compacts if needed, and writes back.
func (m *Manager) Append(entry Entry, maxEntries int, retentionDays int) error {
	entries, err := m.Load()
	if err != nil {
		return err
	}
	entries = append(entries, entry)
	entries = Compact(entries, maxEntries, retentionDays)
	return m.write(entries)
}

func (m *Manager) write(entries []Entry) error {
	if err := os.MkdirAll(filepath.Dir(m.path), 0o755); err != nil {
		return fmt.Errorf("create cache dir: %w", err)
	}
	f, err := os.Create(m.path)
	if err != nil {
		return fmt.Errorf("create cache file: %w", err)
	}
	defer func() { _ = f.Close() }()

	enc := json.NewEncoder(f)
	for _, e := range entries {
		if err := enc.Encode(e); err != nil {
			return fmt.Errorf("encode cache entry: %w", err)
		}
	}
	return nil
}

// Compact keeps all entries but empties heavy fields (Corners, ConversationNotes) for entries
// outside the detailed window (most recent maxEntries entries that are within retentionDays).
func Compact(entries []Entry, maxEntries int, retentionDays int) []Entry {
	cutoff := time.Now().AddDate(0, 0, -retentionDays)

	// Find indices of entries within retention window
	recentIndices := make([]int, 0, len(entries))
	for i, e := range entries {
		t, err := time.Parse(time.RFC3339, e.Datetime)
		if err != nil || !t.Before(cutoff) {
			recentIndices = append(recentIndices, i)
		}
	}

	// Among recent entries, the last maxEntries form the detailed window
	detailedStart := 0
	if len(recentIndices) > maxEntries {
		detailedStart = len(recentIndices) - maxEntries
	}
	detailed := make(map[int]bool, len(recentIndices)-detailedStart)
	for _, idx := range recentIndices[detailedStart:] {
		detailed[idx] = true
	}

	result := make([]Entry, len(entries))
	for i, e := range entries {
		if !detailed[i] {
			e.Corners = make([]CornerEntry, 0)
			e.ConversationNotes = make([]model.ConversationNote, 0)
		}
		result[i] = e
	}
	return result
}

// Recent returns the last n entries (most recent).
func Recent(entries []Entry, n int) []Entry {
	if n <= 0 || len(entries) == 0 {
		return make([]Entry, 0)
	}
	if len(entries) <= n {
		return entries
	}
	return entries[len(entries)-n:]
}

// BuildEntryFromManifest constructs a cache Entry from a program ID, manifest, rundown, and media info.
// Rundown data (summary, points) is merged into the manifest's article references by URL.
// bytes and durationSec are from mediainfo (0 if not available).
func BuildEntryFromManifest(programID string, m model.Manifest, rd model.Rundown, bytes int64, durationSec int) Entry {
	rdArticleByURL := make(map[string]model.RundownArticle)
	for _, c := range rd.Corners {
		for _, a := range c.Articles {
			rdArticleByURL[a.URL] = a
		}
	}

	corners := make([]CornerEntry, len(m.Corners))
	for i, mc := range m.Corners {
		articles := make([]ArticleEntry, len(mc.Articles))
		for j, ar := range mc.Articles {
			ae := ArticleEntry{
				Title:  ar.Title,
				URL:    ar.URL,
				Points: make([]string, 0),
			}
			if rda, ok := rdArticleByURL[ar.URL]; ok {
				ae.Summary = rda.Summary
				if rda.Points != nil {
					ae.Points = rda.Points
				}
			}
			articles[j] = ae
		}
		points := mc.Points
		if points == nil {
			points = make([]string, 0)
		}
		corners[i] = CornerEntry{Title: mc.Title, Summary: mc.Summary, Points: points, Articles: articles}
	}

	notes := m.ConversationNotes
	if notes == nil {
		notes = make([]model.ConversationNote, 0)
	}
	return Entry{
		ProgramID:         programID,
		Datetime:          m.Datetime,
		EpisodeNumber:     m.EpisodeNumber,
		EpisodeTitle:      m.EpisodeTitle,
		Title:             m.Title,
		Summary:           m.Summary,
		Description:       m.Description,
		AudioFile:         m.AudioFile,
		Bytes:             bytes,
		DurationSec:       durationSec,
		Corners:           corners,
		ConversationNotes: notes,
	}
}

// NextEpisodeNumber returns the episode number to assign to the next episode.
// If entries is empty, returns 1.
// If the latest entry has EpisodeNumber > 0, returns that number + 1.
// Otherwise (legacy entries without episode_number), returns len(entries) + 1.
func NextEpisodeNumber(entries []Entry) int {
	if len(entries) == 0 {
		return 1
	}
	latest := entries[len(entries)-1]
	if latest.EpisodeNumber > 0 {
		return latest.EpisodeNumber + 1
	}
	return len(entries) + 1
}

// PastURLs extracts all unique article URLs from entries, in order.
func PastURLs(entries []Entry) []string {
	urls := make([]string, 0)
	seen := make(map[string]bool)
	for _, e := range entries {
		for _, corner := range e.Corners {
			for _, a := range corner.Articles {
				if !seen[a.URL] {
					seen[a.URL] = true
					urls = append(urls, a.URL)
				}
			}
		}
	}
	return urls
}
