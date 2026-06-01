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
	ProgramID string        `json:"program_id"`
	Datetime  string        `json:"datetime"`
	Title     string        `json:"title"`
	Summary   string        `json:"summary"`
	Corners   []CornerEntry `json:"corners"`
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

// Append loads existing entries, adds the new entry, prunes if needed, and writes back.
func (m *Manager) Append(entry Entry, maxEntries int, retentionDays int) error {
	entries, err := m.Load()
	if err != nil {
		return err
	}
	entries = append(entries, entry)
	entries = prune(entries, maxEntries, retentionDays)
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

// prune removes entries that are too old or exceed the max count, keeping the newest.
func prune(entries []Entry, maxEntries int, retentionDays int) []Entry {
	cutoff := time.Now().AddDate(0, 0, -retentionDays)

	filtered := make([]Entry, 0, len(entries))
	for _, e := range entries {
		t, err := time.Parse(time.RFC3339, e.Datetime)
		if err != nil || !t.Before(cutoff) {
			filtered = append(filtered, e)
		}
	}

	if len(filtered) > maxEntries {
		filtered = filtered[len(filtered)-maxEntries:]
	}
	return filtered
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

// BuildEntryFromManifest constructs a cache Entry from a program ID, manifest, and rundown.
// Rundown data (summary, points) is merged into the manifest's article references by URL.
func BuildEntryFromManifest(programID string, m model.Manifest, rd model.Rundown) Entry {
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

	return Entry{
		ProgramID: programID,
		Datetime:  m.Datetime,
		Title:     m.Title,
		Summary:   m.Summary,
		Corners:   corners,
	}
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
