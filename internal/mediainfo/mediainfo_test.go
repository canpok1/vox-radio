package mediainfo_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/canpok1/vox-radio/internal/mediainfo"
)

func TestFileSize_ReturnsCorrectSize(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.txt")
	content := []byte("hello world")
	if err := os.WriteFile(path, content, 0o644); err != nil {
		t.Fatalf("setup: %v", err)
	}

	got, err := mediainfo.FileSize(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != int64(len(content)) {
		t.Errorf("got %d, want %d", got, len(content))
	}
}

func TestFileSize_NonExistentFile_ReturnsError(t *testing.T) {
	_, err := mediainfo.FileSize("/nonexistent/path/to/file.mp3")
	if err == nil {
		t.Error("expected error for non-existent file, got nil")
	}
}

func TestDuration_NonExistentFile_ReturnsError(t *testing.T) {
	_, err := mediainfo.Duration("/nonexistent/path/to/file.mp3")
	if err == nil {
		t.Error("expected error for non-existent file, got nil")
	}
}
