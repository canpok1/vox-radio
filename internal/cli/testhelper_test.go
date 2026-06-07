package cli_test

import (
	"path/filepath"
	"runtime"
	"testing"

	"github.com/canpok1/vox-radio/internal/testutil"
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

func writeFeedSpecForTest(t *testing.T, content []byte) string {
	t.Helper()
	return testutil.WriteTempFile(t, "feed-spec.yaml", content)
}

func writeSlackSpecRawForTest(t *testing.T, content []byte) string {
	t.Helper()
	return testutil.WriteTempFile(t, "slack-spec.yaml", content)
}
