package cli_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/canpok1/vox-radio/internal/cli"
	"github.com/canpok1/vox-radio/internal/testutil"
)

func TestFeedgenCheck_ValidSpec_Success(t *testing.T) {
	cmd := cli.NewRootCmd()
	buf := &bytes.Buffer{}
	cmd.SetOut(buf)
	cmd.SetArgs([]string{"feedgen", "check", configTestdataPath("feed_spec.yaml")})
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
	cmd.SetArgs([]string{"feedgen", "check", configTestdataPath("feed_spec_unknown_key.yaml")})
	err := cmd.Execute()
	if err == nil {
		t.Error("expected error for unknown key in strict mode")
	}
}

func TestFeedgenCheck_MissingRequiredField_Error(t *testing.T) {
	// feed.language が欠落した feed-spec.yaml
	specPath := testutil.WriteTempFile(t, "feed-spec.yaml", []byte(`feed:
  author: Test Author
  email: test@example.com
  site_url: https://example.com/
  audio_url_template: "https://example.com/ep-{episode_number}/{audio_file}"
`))

	cmd := cli.NewRootCmd()
	errBuf := &bytes.Buffer{}
	cmd.SetErr(errBuf)
	cmd.SetArgs([]string{"feedgen", "check", specPath})
	err := cmd.Execute()
	if err == nil {
		t.Error("expected error for missing required field")
	}
}

// program_id は FeedSpec から削除されたため、feedgen check で unknown key エラーになること
func TestFeedgenCheck_ProgramID_RaisesUnknownKey(t *testing.T) {
	specPath := testutil.WriteTempFile(t, "feed-spec.yaml", []byte(`program_id: my-radio
feed:
  language: ja
  author: Test Author
  email: test@example.com
  site_url: https://example.com/
  audio_url_template: "https://example.com/ep-{episode_number}/{audio_file}"
`))

	cmd := cli.NewRootCmd()
	cmd.SetArgs([]string{"feedgen", "check", specPath})
	if err := cmd.Execute(); err == nil {
		t.Error("expected error for program_id (unknown key) in feedgen check, got nil")
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
