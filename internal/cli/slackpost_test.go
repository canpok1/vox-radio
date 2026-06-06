package cli_test

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/canpok1/vox-radio/internal/cli"
)

func writeSlackSpecForTest(t *testing.T, dir, channel string) string {
	t.Helper()
	content := `slack:
  channel: "` + channel + `"
`
	path := filepath.Join(dir, "slack-spec.yaml")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write slack spec: %v", err)
	}
	return path
}

func writeManifestForTest(t *testing.T, dir string) string {
	t.Helper()
	manifest := map[string]any{
		"title":          "テスト番組",
		"episode_number": 1,
		"episode_title":  "第1回",
		"summary":        "テスト",
		"audio_file":     "ep1.mp3",
		"corners":        []any{},
	}
	data, _ := json.Marshal(manifest)
	path := filepath.Join(dir, "manifest.json")
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}
	_ = os.WriteFile(filepath.Join(dir, "ep1.mp3"), []byte("fake mp3"), 0o644)
	return path
}

func writeConfigForSlackTest(t *testing.T, dir string) string {
	t.Helper()
	content := `llm:
  provider: openai
  temperature: 0.7
  max_retries: 3
  steps:
    write: { temperature: 0.8 }
  openai:
    base_url: https://example.com/
    api_key_env: OPENAI_API_KEY
    model: gpt-4

voicevox:
  url: http://localhost:50021

characters: {}

slack:
  bot_token_env: TEST_SLACKPOST_TOKEN
`
	path := filepath.Join(dir, "vox-radio.yaml")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}
	return path
}

func TestSlackpostCheck_ValidSpec_Success(t *testing.T) {
	dir := t.TempDir()
	specPath := writeSlackSpecForTest(t, dir, "C0123456789")

	cmd := cli.NewRootCmd()
	buf := &bytes.Buffer{}
	cmd.SetOut(buf)
	cmd.SetArgs([]string{"slackpost", "check", specPath})
	err := cmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(buf.String(), "OK") {
		t.Errorf("expected OK in output, got: %s", buf.String())
	}
}

func TestSlackpostCheck_EmptyChannel_Error(t *testing.T) {
	dir := t.TempDir()
	content := `slack:
  channel: ""
`
	specPath := filepath.Join(dir, "slack-spec.yaml")
	if err := os.WriteFile(specPath, []byte(content), 0o644); err != nil {
		t.Fatalf("write spec: %v", err)
	}

	cmd := cli.NewRootCmd()
	cmd.SetArgs([]string{"slackpost", "check", specPath})
	if err := cmd.Execute(); err == nil {
		t.Error("expected error for empty channel")
	}
}

func TestSlackpostCheck_UnknownKey_Error(t *testing.T) {
	dir := t.TempDir()
	content := `slack:
  channel: "C0123456789"
unknown_key: value
`
	specPath := filepath.Join(dir, "slack-spec.yaml")
	if err := os.WriteFile(specPath, []byte(content), 0o644); err != nil {
		t.Fatalf("write spec: %v", err)
	}

	cmd := cli.NewRootCmd()
	cmd.SetArgs([]string{"slackpost", "check", specPath})
	if err := cmd.Execute(); err == nil {
		t.Error("expected error for unknown key in strict mode")
	}
}

// program_id は SlackSpec から削除されたため、slackpost check で unknown key エラーになること
func TestSlackpostCheck_ProgramID_RaisesUnknownKey(t *testing.T) {
	dir := t.TempDir()
	content := `program_id: my-radio
slack:
  channel: "C0123456789"
`
	specPath := filepath.Join(dir, "slack-spec.yaml")
	if err := os.WriteFile(specPath, []byte(content), 0o644); err != nil {
		t.Fatalf("write spec: %v", err)
	}

	cmd := cli.NewRootCmd()
	cmd.SetArgs([]string{"slackpost", "check", specPath})
	if err := cmd.Execute(); err == nil {
		t.Error("expected error for program_id (unknown key) in slackpost check, got nil")
	}
}

func TestSlackpostCheck_MissingSpecArg_Error(t *testing.T) {
	cmd := cli.NewRootCmd()
	cmd.SetArgs([]string{"slackpost", "check"})
	if err := cmd.Execute(); err == nil {
		t.Error("expected error when spec path is missing")
	}
}

func TestSlackpost_DryRun_Success(t *testing.T) {
	dir := t.TempDir()
	manifestPath := writeManifestForTest(t, dir)
	specPath := writeSlackSpecForTest(t, dir, "C0123456789")
	configPath := writeConfigForSlackTest(t, dir)

	t.Setenv("TEST_SLACKPOST_TOKEN", "xoxb-test")

	cmd := cli.NewRootCmd()
	buf := &bytes.Buffer{}
	cmd.SetOut(buf)
	cmd.SetArgs([]string{
		"--config", configPath,
		"slackpost",
		"--manifest", manifestPath,
		"--spec", specPath,
		"--dry-run",
	})
	err := cmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "ep1.mp3") {
		t.Errorf("output should contain audio filename, got: %q", out)
	}
}
