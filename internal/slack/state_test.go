package slack

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultStatePath(t *testing.T) {
	tests := []struct {
		name         string
		manifestPath string
		want         string
	}{
		{
			name:         "json extension",
			manifestPath: "output/manifest.json",
			want:         "output/manifest.slackpost-state.json",
		},
		{
			name:         "yaml extension stripped",
			manifestPath: "output/manifest.yaml",
			want:         "output/manifest.slackpost-state.json",
		},
		{
			name:         "absolute path",
			manifestPath: "/tmp/output/manifest.json",
			want:         "/tmp/output/manifest.slackpost-state.json",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := DefaultStatePath(tt.manifestPath)
			if got != tt.want {
				t.Errorf("DefaultStatePath(%q) = %q, want %q", tt.manifestPath, got, tt.want)
			}
		})
	}
}

func TestPostState_Matches(t *testing.T) {
	s := PostState{AudioFile: "episode.mp3", EpisodeNumber: 13}
	tests := []struct {
		name          string
		audioFile     string
		episodeNumber int
		want          bool
	}{
		{"same audio and episode", "episode.mp3", 13, true},
		{"different episode_number", "episode.mp3", 14, false},
		{"different audio_file", "other.mp3", 13, false},
		{"both different", "other.mp3", 99, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := s.Matches(tt.audioFile, tt.episodeNumber); got != tt.want {
				t.Errorf("Matches(%q, %d) = %v, want %v", tt.audioFile, tt.episodeNumber, got, tt.want)
			}
		})
	}
}

func TestSaveAndLoadState_RoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.slackpost-state.json")

	want := PostState{
		AudioFile:     "ep42.mp3",
		EpisodeNumber: 42,
		Channel:       "C0123",
		FileID:        "F0456",
		ThreadTS:      "1700000000.000100",
		Replied:       true,
	}

	if err := saveState(path, want); err != nil {
		t.Fatalf("saveState: %v", err)
	}

	got, err := loadState(path)
	if err != nil {
		t.Fatalf("loadState: %v", err)
	}
	if *got != want {
		t.Errorf("loadState = %+v, want %+v", *got, want)
	}
}

func TestLoadState_FileNotExist_ReturnsError(t *testing.T) {
	_, err := loadState(filepath.Join(t.TempDir(), "nonexistent.json"))
	if err == nil {
		t.Error("expected error when file does not exist")
	}
}

func TestSaveState_CreatesFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "new.slackpost-state.json")

	if err := saveState(path, PostState{AudioFile: "ep1.mp3"}); err != nil {
		t.Fatalf("saveState: %v", err)
	}

	if _, err := os.Stat(path); err != nil {
		t.Errorf("state file should exist after saveState: %v", err)
	}
}
