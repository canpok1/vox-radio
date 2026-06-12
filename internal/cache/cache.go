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
	DedupKey string   `json:"dedup_key,omitempty"` // 重複判定キー（sha256:hex）。旧エントリでは空
	Title    string   `json:"title"`
	URL      string   `json:"url,omitempty"`
	Summary  string   `json:"summary"`
	Points   []string `json:"points"`
}

// CornerEntry holds corner data for a single episode history entry.
type CornerEntry struct {
	ID       string         `json:"id"`
	Title    string         `json:"title"`
	Summary  string         `json:"summary"`
	Points   []string       `json:"points"`
	Articles []ArticleEntry `json:"articles"`
}

// CastEntry holds cast data for a single episode history entry.
type CastEntry struct {
	CharacterID string `json:"character_id"`
	Type        string `json:"type"` // "regular" | "guest"
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
	Casts             []CastEntry              `json:"casts"`
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
	return model.NonNil(entries), nil
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

// stripCornerHeavyFields returns a copy of corners with only identity fields (ID, Title) kept,
// dropping heavy fields (Summary, Points, Articles). Corner IDs must survive compaction so that
// CornerAppearances can count appearances across the full history (mirrors how Casts are kept).
func stripCornerHeavyFields(corners []CornerEntry) []CornerEntry {
	stripped := make([]CornerEntry, len(corners))
	for i, c := range corners {
		stripped[i] = CornerEntry{ID: c.ID, Title: c.Title}
	}
	return stripped
}

// Compact keeps all entries but drops heavy corner fields (Summary, Points, Articles) and
// ConversationNotes for entries outside the detailed window (most recent maxEntries entries that
// are within retentionDays). Corner identity (ID, Title) is preserved so appearance counting works.
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
			e.Corners = stripCornerHeavyFields(e.Corners)
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
// Rundown data (summary, points) is merged into the manifest's article references by DedupKey.
// bytes and durationSec are from mediainfo (0 if not available).
func BuildEntryFromManifest(programID string, m model.Manifest, rd model.Rundown, bytes int64, durationSec int) Entry {
	rdArticleByDedupKey := make(map[string]model.RundownArticle)
	for _, c := range rd.Corners {
		for _, a := range c.Articles {
			if a.DedupKey != "" {
				rdArticleByDedupKey[a.DedupKey] = a
			}
		}
	}

	corners := make([]CornerEntry, len(m.Corners))
	for i, mc := range m.Corners {
		articles := make([]ArticleEntry, len(mc.Articles))
		for j, ar := range mc.Articles {
			ae := ArticleEntry{
				DedupKey: ar.DedupKey,
				Title:    ar.Title,
				URL:      ar.URL,
				Points:   make([]string, 0),
			}
			if rda, ok := rdArticleByDedupKey[ar.DedupKey]; ok {
				ae.Summary = rda.Summary
				ae.Points = rda.Points
			}
			articles[j] = ae
		}
		corners[i] = CornerEntry{ID: mc.ID, Title: mc.Title, Summary: mc.Summary, Points: mc.Points, Articles: articles}
	}

	casts := make([]CastEntry, len(m.Casts))
	for i, c := range m.Casts {
		casts[i] = CastEntry{CharacterID: c.CharacterID, Type: c.Type}
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
		ConversationNotes: m.ConversationNotes,
		Casts:             casts,
	}
}

// aggregateAppearances walks entries in chronological order and aggregates appearance stats per ID.
// idsOf extracts the relevant IDs from a single entry; empty IDs are skipped (legacy entries with
// no ID). For each occurrence, the running count is incremented and lastEpisode is set to the
// entry's EpisodeNumber (so the most recent appearance wins). build converts the aggregated
// (count, lastEpisode) pair into the caller's result type. Shared by CornerAppearances and
// CastAppearances, which differ only in which IDs they extract and which type they return.
func aggregateAppearances[T any](entries []Entry, idsOf func(Entry) []string, build func(count, lastEpisode int) T) map[string]T {
	type stats struct {
		count       int
		lastEpisode int
	}
	acc := make(map[string]stats)
	for _, e := range entries {
		for _, id := range idsOf(e) {
			if id == "" {
				continue
			}
			s := acc[id]
			s.count++
			s.lastEpisode = e.EpisodeNumber
			acc[id] = s
		}
	}
	result := make(map[string]T, len(acc))
	for id, s := range acc {
		result[id] = build(s.count, s.lastEpisode)
	}
	return result
}

// CastAppearance holds aggregated appearance stats for a single cast member across history.
type CastAppearance struct {
	Count             int // number of past episodes the cast member appeared in (excluding the current episode)
	LastEpisodeNumber int // episode_number of the most recent past appearance (0 if none/unknown)
}

// CastAppearances returns a map from character ID to its aggregated appearance stats.
// Entries are walked in chronological order; for each character ID, Count is incremented and
// LastEpisodeNumber is set to the entry's EpisodeNumber (so the last appearance wins).
// Entries without Casts (legacy entries) contribute nothing.
func CastAppearances(entries []Entry) map[string]CastAppearance {
	return aggregateAppearances(entries, func(e Entry) []string {
		ids := make([]string, len(e.Casts))
		for i, c := range e.Casts {
			ids[i] = c.CharacterID
		}
		return ids
	}, func(count, lastEpisode int) CastAppearance {
		return CastAppearance{Count: count, LastEpisodeNumber: lastEpisode}
	})
}

// CornerAppearance holds aggregated appearance stats for a single corner across history.
type CornerAppearance struct {
	Count             int // number of past episodes the corner appeared in (excluding the current episode)
	LastEpisodeNumber int // episode_number of the most recent past appearance (0 if none/unknown)
}

// CornerAppearances returns a map from corner ID to its aggregated appearance stats.
// Entries are walked in chronological order; for each corner ID, Count is incremented and
// LastEpisodeNumber is set to the entry's EpisodeNumber (so the last appearance wins).
// Corners without an ID (legacy entries) are ignored (0 appearances = treated as new corner).
func CornerAppearances(entries []Entry) map[string]CornerAppearance {
	return aggregateAppearances(entries, func(e Entry) []string {
		ids := make([]string, len(e.Corners))
		for i, c := range e.Corners {
			ids[i] = c.ID
		}
		return ids
	}, func(count, lastEpisode int) CornerAppearance {
		return CornerAppearance{Count: count, LastEpisodeNumber: lastEpisode}
	})
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
// Deprecated: Use PastDedupKeys for new code. PastURLs is kept for legacy cache entries.
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

// PastDedupKeys extracts all unique article DedupKeys from entries, in order.
// Entries without a DedupKey (legacy) are skipped.
func PastDedupKeys(entries []Entry) []string {
	keys := make([]string, 0)
	seen := make(map[string]bool)
	for _, e := range entries {
		for _, corner := range e.Corners {
			for _, a := range corner.Articles {
				if a.DedupKey != "" && !seen[a.DedupKey] {
					seen[a.DedupKey] = true
					keys = append(keys, a.DedupKey)
				}
			}
		}
	}
	return keys
}
