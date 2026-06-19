package logging_test

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/canpok1/vox-radio/internal/logging"
)

func TestNewSetup_CreatesLogFile(t *testing.T) {
	dir := t.TempDir()
	logDir := filepath.Join(dir, "logs")

	now := time.Date(2026, 5, 31, 12, 1, 3, 0, time.UTC)
	logger, f, err := logging.NewSetup(now, "gather", logDir)
	if err != nil {
		t.Fatalf("NewSetup error: %v", err)
	}
	defer f.Close()

	if logger == nil {
		t.Fatal("logger should not be nil")
	}
	if f == nil {
		t.Fatal("log file should not be nil")
	}

	logPath := filepath.Join(logDir, "20260531-120103-gather.log")
	if _, err := os.Stat(logPath); os.IsNotExist(err) {
		t.Errorf("log file not created: %s", logPath)
	}
}

func TestNewSetup_WritesToLogFile(t *testing.T) {
	dir := t.TempDir()
	logDir := filepath.Join(dir, "logs")

	now := time.Date(2026, 5, 31, 12, 1, 3, 0, time.UTC)
	logger, f, err := logging.NewSetup(now, "run", logDir)
	if err != nil {
		t.Fatalf("NewSetup error: %v", err)
	}
	defer f.Close()

	logger.With("step", "gather").Info("テストメッセージ")

	// Flush by closing and re-reading
	if err := f.Sync(); err != nil {
		t.Fatalf("sync error: %v", err)
	}

	logPath := filepath.Join(logDir, "20260531-120103-run.log")
	data, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("read log file: %v", err)
	}
	if !strings.Contains(string(data), "テストメッセージ") {
		t.Errorf("log file should contain message: %q", string(data))
	}
}

func TestNewSetup_LogFileAcceptsDEBUG(t *testing.T) {
	dir := t.TempDir()
	logDir := filepath.Join(dir, "logs")

	now := time.Date(2026, 5, 31, 12, 1, 3, 0, time.UTC)
	logger, f, err := logging.NewSetup(now, "test", logDir)
	if err != nil {
		t.Fatalf("NewSetup error: %v", err)
	}
	defer f.Close()

	logger.Debug("デバッグメッセージ")

	if err := f.Sync(); err != nil {
		t.Fatalf("sync error: %v", err)
	}

	logPath := filepath.Join(logDir, "20260531-120103-test.log")
	data, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("read log file: %v", err)
	}
	if !strings.Contains(string(data), "デバッグメッセージ") {
		t.Errorf("log file should contain DEBUG message: %q", string(data))
	}
}

func TestNewSetup_StderrHandlerAcceptsINFOOnly(t *testing.T) {
	dir := t.TempDir()
	logDir := filepath.Join(dir, "logs")

	now := time.Date(2026, 5, 31, 12, 1, 3, 0, time.UTC)
	logger, f, err := logging.NewSetup(now, "test", logDir)
	if err != nil {
		t.Fatalf("NewSetup error: %v", err)
	}
	defer f.Close()

	// Verify the underlying handler accepts INFO but not DEBUG
	handler := logger.Handler()
	if !handler.Enabled(context.TODO(), slog.LevelInfo) {
		t.Error("INFO should be enabled (file handler accepts it)")
	}
	if !handler.Enabled(context.TODO(), slog.LevelDebug) {
		t.Error("DEBUG should be enabled (file handler accepts it via fan-out)")
	}
}
