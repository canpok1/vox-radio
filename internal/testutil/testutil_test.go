package testutil_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/canpok1/vox-radio/internal/testutil"
)

func TestWriteTempFile_ReturnsPathToWrittenFile(t *testing.T) {
	content := []byte("hello: world\n")
	path := testutil.WriteTempFile(t, "test.yaml", content)

	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read file: %v", err)
	}
	if string(got) != string(content) {
		t.Errorf("content = %q, want %q", got, content)
	}
	if filepath.Base(path) != "test.yaml" {
		t.Errorf("filename = %q, want %q", filepath.Base(path), "test.yaml")
	}
}

func TestWriteTempFile_FileIsCleanedUpAfterTest(t *testing.T) {
	var savedPath string
	t.Run("inner", func(t *testing.T) {
		savedPath = testutil.WriteTempFile(t, "cleanup.yaml", []byte("data"))
	})
	if _, err := os.Stat(savedPath); !os.IsNotExist(err) {
		t.Errorf("expected file to be cleaned up, but it still exists: %s", savedPath)
	}
}
