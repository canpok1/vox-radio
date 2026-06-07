package cli_test

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

// cliTestSrcDir is the absolute path of this test file's directory, resolved at init time.
var cliTestSrcDir string

func init() {
	_, file, _, _ := runtime.Caller(0)
	cliTestSrcDir = filepath.Dir(file)
}

func configTestdataPath(rel string) string {
	return filepath.Join(cliTestSrcDir, "..", "config", "testdata", rel)
}

func writeTempFileForTest(t *testing.T, name string, content []byte) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, content, 0o644); err != nil {
		t.Fatalf("write %s: %v", name, err)
	}
	return path
}

func writeFeedSpecForTest(t *testing.T, content []byte) string {
	t.Helper()
	return writeTempFileForTest(t, "feed-spec.yaml", content)
}

func writeSlackSpecRawForTest(t *testing.T, content []byte) string {
	t.Helper()
	return writeTempFileForTest(t, "slack-spec.yaml", content)
}
