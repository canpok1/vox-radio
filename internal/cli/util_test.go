package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSetupLogger_DefaultLogDir(t *testing.T) {
	// os.Chdir changes the process-wide cwd, so no t.Parallel().
	origDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	tmpDir := t.TempDir()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("chdir to tmpDir: %v", err)
	}
	defer func() {
		_ = os.Chdir(origDir)
	}()

	logger, f, err := setupLogger("collect", "")
	if err != nil {
		t.Fatalf("setupLogger: %v", err)
	}
	defer f.Close()

	if logger == nil {
		t.Fatal("expected non-nil logger")
	}

	logPath, err := filepath.Abs(f.Name())
	if err != nil {
		t.Fatalf("abs: %v", err)
	}
	wantPrefix := filepath.Join(tmpDir, ".vox-radio", "logs")
	if !strings.HasPrefix(logPath, wantPrefix) {
		t.Errorf("log file path %q does not start with %q", logPath, wantPrefix)
	}
}
