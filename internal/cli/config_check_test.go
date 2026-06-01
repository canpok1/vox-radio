package cli_test

import (
	"bytes"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/canpok1/vox-radio/internal/cli"
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

func TestConfigCheck_ValidYAML_Success(t *testing.T) {
	cmd := cli.NewRootCmd()
	buf := &bytes.Buffer{}
	cmd.SetOut(buf)
	cmd.SetArgs([]string{"config", "check", configTestdataPath("config.yaml")})
	err := cmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(buf.String(), "OK") {
		t.Errorf("expected OK in output, got: %s", buf.String())
	}
}

func TestConfigCheck_UnknownKey_Error(t *testing.T) {
	cmd := cli.NewRootCmd()
	cmd.SetArgs([]string{"config", "check", configTestdataPath("config_unknown_key.yaml")})
	err := cmd.Execute()
	if err == nil {
		t.Error("expected error for unknown key in strict mode")
	}
}

func TestConfigCheck_InvalidDefaultStyle_Error(t *testing.T) {
	cmd := cli.NewRootCmd()
	cmd.SetArgs([]string{"config", "check", configTestdataPath("config_invalid_default_style.yaml")})
	err := cmd.Execute()
	if err == nil {
		t.Error("expected error for invalid default_style")
	}
}

func TestConfigCheck_DefaultPath_MissingFile_Error(t *testing.T) {
	chdirTemp(t)

	cmd := cli.NewRootCmd()
	cmd.SetArgs([]string{"config", "check"})
	err := cmd.Execute()
	if err == nil {
		t.Error("expected error when vox-radio.yaml does not exist in cwd")
	}
}
