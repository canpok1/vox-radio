package testutil

import (
	"os"
	"path/filepath"
	"testing"
)

// WriteTempFile writes content to a temp file named name inside t.TempDir() and returns its path.
func WriteTempFile(t *testing.T, name string, content []byte) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(path, content, 0o644); err != nil {
		t.Fatalf("write %s: %v", name, err)
	}
	return path
}
