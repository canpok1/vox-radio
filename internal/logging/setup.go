package logging

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"time"
)

// NewSetup creates a logger that fan-outs to stderr (INFO+) and a new log file (DEBUG+).
// The log file is created at <logDir>/<YYYYMMDD-HHMMSS>-<commandName>.log.
// The caller is responsible for closing the returned *os.File.
func NewSetup(now time.Time, commandName string, logDir string) (*slog.Logger, *os.File, error) {
	if err := os.MkdirAll(logDir, 0o755); err != nil {
		return nil, nil, fmt.Errorf("create log dir: %w", err)
	}

	logFileName := fmt.Sprintf("%s-%s.log", now.Format("20060102-150405"), commandName)
	logPath := filepath.Join(logDir, logFileName)

	logFile, err := os.Create(logPath)
	if err != nil {
		return nil, nil, fmt.Errorf("create log file: %w", err)
	}

	stderrHandler := NewTextHandler(os.Stderr, slog.LevelInfo)
	fileHandler := NewTextHandler(logFile, slog.LevelDebug)
	logger := slog.New(NewFanOutHandler(stderrHandler, fileHandler))

	return logger, logFile, nil
}
