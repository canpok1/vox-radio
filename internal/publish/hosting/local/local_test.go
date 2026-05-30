package local_test

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/canpok1/vox-radio/internal/model"
	"github.com/canpok1/vox-radio/internal/publish/hosting/local"
)

func TestHosting_PutAudio(t *testing.T) {
	dir := t.TempDir()
	h := local.New(dir, "https://example.com")

	content := []byte("fake mp3 data")
	url, err := h.PutAudio(context.Background(), "episode_2026-05-30.mp3", bytes.NewReader(content))
	if err != nil {
		t.Fatalf("PutAudio: %v", err)
	}

	want := "https://example.com/audio/episode_2026-05-30.mp3"
	if url != want {
		t.Errorf("url = %q, want %q", url, want)
	}

	got, err := os.ReadFile(filepath.Join(dir, "audio", "episode_2026-05-30.mp3"))
	if err != nil {
		t.Fatalf("read file: %v", err)
	}
	if !bytes.Equal(got, content) {
		t.Errorf("file content mismatch")
	}
}

func TestHosting_PutAudio_TrailingSlash(t *testing.T) {
	dir := t.TempDir()
	h := local.New(dir, "https://example.com/vox-radio/")

	url, err := h.PutAudio(context.Background(), "ep.mp3", bytes.NewReader([]byte{}))
	if err != nil {
		t.Fatalf("PutAudio: %v", err)
	}

	if strings.Contains(url, "//audio/") {
		t.Errorf("double slash in url: %q", url)
	}
	want := "https://example.com/vox-radio/audio/ep.mp3"
	if url != want {
		t.Errorf("url = %q, want %q", url, want)
	}
}

func TestHosting_PutFeed(t *testing.T) {
	dir := t.TempDir()
	h := local.New(dir, "https://example.com")

	feedXML := []byte(`<?xml version="1.0"?><rss></rss>`)
	url, err := h.PutFeed(context.Background(), feedXML)
	if err != nil {
		t.Fatalf("PutFeed: %v", err)
	}

	want := "https://example.com/feed.xml"
	if url != want {
		t.Errorf("url = %q, want %q", url, want)
	}

	got, err := os.ReadFile(filepath.Join(dir, "feed.xml"))
	if err != nil {
		t.Fatalf("read file: %v", err)
	}
	if !bytes.Equal(got, feedXML) {
		t.Errorf("feed.xml content mismatch")
	}
}

func TestHosting_LoadEpisodes_NotFound(t *testing.T) {
	dir := t.TempDir()
	h := local.New(dir, "https://example.com")

	eps, err := h.LoadEpisodes(context.Background())
	if err != nil {
		t.Fatalf("LoadEpisodes (not found): %v", err)
	}
	if len(eps.Episodes) != 0 {
		t.Errorf("expected empty episodes, got %d", len(eps.Episodes))
	}
}

func TestHosting_LoadEpisodes_Existing(t *testing.T) {
	dir := t.TempDir()
	h := local.New(dir, "https://example.com")

	data := `{"episodes":[{"guid":"ep1","title":"T","description":"D","pub_date":"2026-05-30T21:00:00Z","audio_url":"https://example.com/a.mp3","bytes":100,"duration":"00:01:00"}]}`
	if err := os.WriteFile(filepath.Join(dir, "episodes.json"), []byte(data), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}

	eps, err := h.LoadEpisodes(context.Background())
	if err != nil {
		t.Fatalf("LoadEpisodes: %v", err)
	}
	if len(eps.Episodes) != 1 {
		t.Fatalf("expected 1 episode, got %d", len(eps.Episodes))
	}
	if eps.Episodes[0].GUID != "ep1" {
		t.Errorf("guid = %q, want ep1", eps.Episodes[0].GUID)
	}
}

func TestHosting_SaveEpisodes(t *testing.T) {
	dir := t.TempDir()
	h := local.New(dir, "https://example.com")

	eps := model.Episodes{
		Episodes: []model.Episode{
			{
				GUID:        "ep1",
				Title:       "T",
				Description: "D",
				PubDate:     "2026-05-30T21:00:00Z",
				AudioURL:    "https://example.com/a.mp3",
				Bytes:       100,
				Duration:    "00:01:00",
			},
		},
	}
	if err := h.SaveEpisodes(context.Background(), eps); err != nil {
		t.Fatalf("SaveEpisodes: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(dir, "episodes.json"))
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if !strings.Contains(string(data), "ep1") {
		t.Errorf("episodes.json does not contain ep1")
	}
}

func TestHosting_SaveLoadRoundTrip(t *testing.T) {
	dir := t.TempDir()
	h := local.New(dir, "https://example.com")

	original := model.Episodes{
		Episodes: []model.Episode{
			{
				GUID:        "ep1",
				Title:       "T",
				Description: "D",
				PubDate:     "2026-05-30T21:00:00Z",
				AudioURL:    "https://example.com/a.mp3",
				Bytes:       100,
				Duration:    "00:01:00",
			},
		},
	}
	if err := h.SaveEpisodes(context.Background(), original); err != nil {
		t.Fatalf("SaveEpisodes: %v", err)
	}

	loaded, err := h.LoadEpisodes(context.Background())
	if err != nil {
		t.Fatalf("LoadEpisodes: %v", err)
	}
	if len(loaded.Episodes) != 1 {
		t.Fatalf("expected 1 episode, got %d", len(loaded.Episodes))
	}
	if loaded.Episodes[0].GUID != "ep1" {
		t.Errorf("guid = %q, want ep1", loaded.Episodes[0].GUID)
	}
}

func TestHosting_DeleteAudio(t *testing.T) {
	dir := t.TempDir()
	h := local.New(dir, "https://example.com")

	audioDir := filepath.Join(dir, "audio")
	if err := os.MkdirAll(audioDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(audioDir, "ep.mp3"), []byte("data"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}

	if err := h.DeleteAudio(context.Background(), "ep.mp3"); err != nil {
		t.Fatalf("DeleteAudio: %v", err)
	}
	if _, err := os.Stat(filepath.Join(audioDir, "ep.mp3")); !os.IsNotExist(err) {
		t.Error("file should be deleted")
	}
}

func TestHosting_DeleteAudio_NonExistent(t *testing.T) {
	dir := t.TempDir()
	h := local.New(dir, "https://example.com")

	if err := h.DeleteAudio(context.Background(), "nonexistent.mp3"); err != nil {
		t.Errorf("DeleteAudio (non-existent) should be no-op: %v", err)
	}
}
