package testutil

import (
	"os"
	"path/filepath"
	"testing"
)

// Ptr returns a pointer to v.
func Ptr[T any](v T) *T { return &v }

// WriteTempFile writes content to a temp file named name inside t.TempDir() and returns its path.
func WriteTempFile(t *testing.T, name string, content []byte) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(path, content, 0o644); err != nil {
		t.Fatalf("write %s: %v", name, err)
	}
	return path
}
