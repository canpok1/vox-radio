package cli_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/canpok1/vox-radio/internal/cli"
)

func TestConfigCheck_ValidYAML_Success(t *testing.T) {
	cmd := cli.NewRootCmd()
	buf := &bytes.Buffer{}
	cmd.SetOut(buf)
	cmd.SetArgs([]string{"config", "check", "--config", configTestdataPath("config.yaml")})
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
	cmd.SetArgs([]string{"config", "check", "--config", configTestdataPath("config_unknown_key.yaml")})
	err := cmd.Execute()
	if err == nil {
		t.Error("expected error for unknown key in strict mode")
	}
}

func TestConfigCheck_InvalidDefaultStyle_Error(t *testing.T) {
	cmd := cli.NewRootCmd()
	cmd.SetArgs([]string{"config", "check", "--config", configTestdataPath("config_invalid_default_style.yaml")})
	err := cmd.Execute()
	if err == nil {
		t.Error("expected error for invalid default_style")
	}
}

func TestConfigCheck_ConfigFlag_DifferentDir(t *testing.T) {
	chdirTemp(t) // cwd に vox-radio.yaml がない状態でも --config で別ディレクトリを指定できる

	cmd := cli.NewRootCmd()
	buf := &bytes.Buffer{}
	cmd.SetOut(buf)
	cmd.SetArgs([]string{"config", "check", "--config", configTestdataPath("config.yaml")})
	err := cmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error with --config pointing outside cwd: %v", err)
	}
	if !strings.Contains(buf.String(), "OK") {
		t.Errorf("expected OK in output, got: %s", buf.String())
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
