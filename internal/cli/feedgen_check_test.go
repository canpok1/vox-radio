package cli_test

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/canpok1/vox-radio/internal/cli"
)

func feedSpecTestdataPath(rel string) string {
	return filepath.Join(cliTestSrcDir, "..", "config", "testdata", rel)
}

func TestFeedgenCheck_ValidSpec_Success(t *testing.T) {
	cmd := cli.NewRootCmd()
	buf := &bytes.Buffer{}
	cmd.SetOut(buf)
	cmd.SetArgs([]string{"feedgen", "check", feedSpecTestdataPath("feed_spec.yaml")})
	err := cmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(buf.String(), "OK") {
		t.Errorf("expected OK in output, got: %s", buf.String())
	}
}

func TestFeedgenCheck_UnknownKey_Error(t *testing.T) {
	cmd := cli.NewRootCmd()
	cmd.SetArgs([]string{"feedgen", "check", feedSpecTestdataPath("feed_spec_unknown_key.yaml")})
	err := cmd.Execute()
	if err == nil {
		t.Error("expected error for unknown key in strict mode")
	}
}

func TestFeedgenCheck_MissingRequiredField_Error(t *testing.T) {
	dir := t.TempDir()
	specPath := filepath.Join(dir, "feed-spec.yaml")
	// program_id が欠落した feed-spec.yaml
	content := []byte(`feed:
  language: ja
  author: Test Author
  email: test@example.com
  site_url: https://example.com/
  audio_url_template: "https://example.com/ep-{episode_number}/{audio_file}"
`)
	if err := os.WriteFile(specPath, content, 0o644); err != nil {
		t.Fatalf("write spec: %v", err)
	}

	cmd := cli.NewRootCmd()
	errBuf := &bytes.Buffer{}
	cmd.SetErr(errBuf)
	cmd.SetArgs([]string{"feedgen", "check", specPath})
	err := cmd.Execute()
	if err == nil {
		t.Error("expected error for missing required field")
	}
}

func TestFeedgenCheck_MissingSpecArg_Error(t *testing.T) {
	cmd := cli.NewRootCmd()
	cmd.SetArgs([]string{"feedgen", "check"})
	err := cmd.Execute()
	if err == nil {
		t.Error("expected error when feed spec path argument is missing")
	}
}
