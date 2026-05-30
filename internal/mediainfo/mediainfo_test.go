package mediainfo_test

import (
	"os"
	"os/exec"
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

func TestDuration_ReturnsPositiveDuration(t *testing.T) {
	if _, err := exec.LookPath("ffprobe"); err != nil {
		t.Skip("ffprobe not available")
	}
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		t.Skip("ffmpeg not available for test file generation")
	}

	dir := t.TempDir()
	wavPath := filepath.Join(dir, "test.wav")
	if err := exec.Command("ffmpeg", "-f", "lavfi", "-i", "sine=frequency=440:duration=1", wavPath).Run(); err != nil {
		t.Fatalf("setup: generate test wav: %v", err)
	}

	got, err := mediainfo.Duration(wavPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got <= 0 {
		t.Errorf("got non-positive duration: %f", got)
	}
}

func TestDuration_NonExistentFile_ReturnsError(t *testing.T) {
	_, err := mediainfo.Duration("/nonexistent/path/to/file.mp3")
	if err == nil {
		t.Error("expected error for non-existent file, got nil")
	}
}
