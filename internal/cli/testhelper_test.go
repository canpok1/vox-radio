package cli_test

import (
	"path/filepath"
	"runtime"
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
