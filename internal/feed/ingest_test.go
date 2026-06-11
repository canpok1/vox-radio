package feed_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/canpok1/vox-radio/internal/cache"
	"github.com/canpok1/vox-radio/internal/feed"
)

func setupIngestDirs(t *testing.T) (cachePath, publicDir string) {
	t.Helper()
	dir := t.TempDir()
	return filepath.Join(dir, "cache.jsonl"),
		filepath.Join(dir, "public")
}

func writeCacheJSONL(t *testing.T, path string, entries []cache.Entry) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	f, err := os.Create(path)
	if err != nil {
		t.Fatalf("create cache file: %v", err)
	}
	defer func() { _ = f.Close() }()
	enc := json.NewEncoder(f)
	for _, e := range entries {
		if err := enc.Encode(e); err != nil {
			t.Fatalf("encode cache entry: %v", err)
		}
	}
}

func TestRun_GeneratesFeedXML(t *testing.T) {
	cachePath, publicDir := setupIngestDirs(t)

	entries := []cache.Entry{
		{
			ProgramID:     "test-radio",
			Datetime:      time.Now().AddDate(0, 0, -7).Format(time.RFC3339),
			EpisodeNumber: 1,
			Title:         "テストラジオ",
			Summary:       "第1回概要",
			Description:   "番組説明",
			AudioFile:     "episode.mp3",
			Bytes:         1000,
			DurationSec:   600,
		},
	}
	writeCacheJSONL(t, cachePath, entries)

	spec := feed.FeedSpec{
		Feed: feed.FeedConfig{
			AudioURLTemplate: "https://example.com/{episode_number}/{audio_file}",
		},
		Output: feed.OutputConfig{Public: publicDir},
	}

	_, n, err := feed.Run(feed.Options{
		CachePath: cachePath,
		Spec:      spec,
	})
	if err != nil {
		t.Fatalf("Run: unexpected error: %v", err)
	}

	if n != 1 {
		t.Errorf("Run: got %d items, want 1", n)
	}

	feedPath := filepath.Join(publicDir, "feed.xml")
	if _, err := os.Stat(feedPath); os.IsNotExist(err) {
		t.Errorf("Run: expected feed.xml at %s to exist", feedPath)
	}

	content, err := os.ReadFile(feedPath)
	if err != nil {
		t.Fatalf("read feed.xml: %v", err)
	}
	if len(content) == 0 {
		t.Error("Run: feed.xml is empty")
	}
}

// program_id フィルタが廃止されたため、cache の全エントリが対象になること
func TestRun_AllEntriesIncluded_WhenProgramIDsDiffer(t *testing.T) {
	cachePath, publicDir := setupIngestDirs(t)

	entries := []cache.Entry{
		{
			ProgramID:     "test-radio",
			Datetime:      time.Now().AddDate(0, 0, -7).Format(time.RFC3339),
			EpisodeNumber: 1,
			Title:         "テストラジオ",
			Summary:       "第1回概要",
			AudioFile:     "episode.mp3",
		},
		{
			ProgramID:     "other-radio",
			Datetime:      time.Now().Format(time.RFC3339),
			EpisodeNumber: 2,
			Title:         "別番組",
			Summary:       "別番組概要",
			AudioFile:     "episode.mp3",
		},
	}
	writeCacheJSONL(t, cachePath, entries)

	spec := feed.FeedSpec{
		Feed: feed.FeedConfig{
			AudioURLTemplate: "https://example.com/{episode_number}/{audio_file}",
		},
		Output: feed.OutputConfig{Public: publicDir},
	}

	_, n, err := feed.Run(feed.Options{
		CachePath: cachePath,
		Spec:      spec,
	})
	if err != nil {
		t.Fatalf("Run: unexpected error: %v", err)
	}

	// program_id フィルタ廃止のため、全エントリが含まれること
	if n != 2 {
		t.Errorf("Run: got %d items, want 2 (all entries included regardless of program_id)", n)
	}
}

func TestRun_ErrorOnEpisodeNumberZero(t *testing.T) {
	cachePath, publicDir := setupIngestDirs(t)

	entries := []cache.Entry{
		{
			ProgramID:     "test-radio",
			Datetime:      time.Now().Format(time.RFC3339),
			EpisodeNumber: 0,
			Title:         "テストラジオ",
			Summary:       "概要",
			AudioFile:     "episode.mp3",
		},
	}
	writeCacheJSONL(t, cachePath, entries)

	spec := feed.FeedSpec{
		Feed: feed.FeedConfig{
			AudioURLTemplate: "https://example.com/{episode_number}/{audio_file}",
		},
		Output: feed.OutputConfig{Public: publicDir},
	}

	_, _, err := feed.Run(feed.Options{
		CachePath: cachePath,
		Spec:      spec,
	})
	if err == nil {
		t.Error("Run: expected error for episode_number=0, got nil")
	}
}

func TestRun_EmptyCache(t *testing.T) {
	cachePath, publicDir := setupIngestDirs(t)

	writeCacheJSONL(t, cachePath, []cache.Entry{})

	spec := feed.FeedSpec{
		Feed: feed.FeedConfig{
			AudioURLTemplate: "https://example.com/{episode_number}/{audio_file}",
		},
		Output: feed.OutputConfig{Public: publicDir},
	}

	_, n, err := feed.Run(feed.Options{
		CachePath: cachePath,
		Spec:      spec,
	})
	if err != nil {
		t.Fatalf("Run: unexpected error for empty cache: %v", err)
	}
	if n != 0 {
		t.Errorf("Run: got %d items, want 0 for empty cache", n)
	}
}
