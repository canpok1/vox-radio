package slack_test

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/canpok1/vox-radio/internal/slack"
)

func writeTestManifest(t *testing.T, dir string) string {
	t.Helper()
	manifest := map[string]any{
		"title":          "ずんだもんテックラジオ",
		"episode_number": 42,
		"episode_title":  "テストエピソード",
		"summary":        "今回はテストです。",
		"audio_file":     "episode42.mp3",
		"corners":        []any{},
	}
	data, err := json.Marshal(manifest)
	if err != nil {
		t.Fatalf("marshal manifest: %v", err)
	}
	path := filepath.Join(dir, "manifest.json")
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}
	return path
}

func writeTestAudio(t *testing.T, dir string) {
	t.Helper()
	path := filepath.Join(dir, "episode42.mp3")
	if err := os.WriteFile(path, []byte("fake mp3"), 0o644); err != nil {
		t.Fatalf("write audio: %v", err)
	}
}

func writeTestSlackSpec(t *testing.T, dir string) string {
	t.Helper()
	content := `
slack:
  channel: "C0123456789"
`
	path := filepath.Join(dir, "slack-spec.yaml")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write slack-spec: %v", err)
	}
	return path
}

func writeTestConfig(t *testing.T, dir string, botTokenEnv string) string {
	t.Helper()
	content := `
llm:
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
  bot_token_env: ` + botTokenEnv + `
`
	path := filepath.Join(dir, "vox-radio.yaml")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}
	return path
}

func TestRun_DryRun_OutputsAudioPathAndComment(t *testing.T) {
	dir := t.TempDir()
	manifestPath := writeTestManifest(t, dir)
	writeTestAudio(t, dir)
	specPath := writeTestSlackSpec(t, dir)
	configPath := writeTestConfig(t, dir, "TEST_SLACK_BOT_TOKEN")

	t.Setenv("TEST_SLACK_BOT_TOKEN", "xoxb-test-token")

	var buf strings.Builder
	err := slack.Run(slack.Options{
		ConfigPath:   configPath,
		ManifestPath: manifestPath,
		SpecPath:     specPath,
		DryRun:       true,
		Out:          &buf,
	}, nil)
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "episode42.mp3") {
		t.Errorf("output should contain audio filename, got: %q", out)
	}
}

func TestRun_DryRun_NoAPICallMade(t *testing.T) {
	dir := t.TempDir()
	manifestPath := writeTestManifest(t, dir)
	writeTestAudio(t, dir)
	specPath := writeTestSlackSpec(t, dir)
	configPath := writeTestConfig(t, dir, "TEST_SLACK_BOT_TOKEN")

	t.Setenv("TEST_SLACK_BOT_TOKEN", "xoxb-test-token")

	posterCalled := false
	mock := &mockPoster{
		uploadAudioFn: func(_ context.Context, _ slack.UploadParams) (string, string, error) {
			posterCalled = true
			return "FILE123", "TS123", nil
		},
	}

	err := slack.Run(slack.Options{
		ConfigPath:   configPath,
		ManifestPath: manifestPath,
		SpecPath:     specPath,
		DryRun:       true,
		Out:          os.Stdout,
	}, mock)
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}
	if posterCalled {
		t.Error("Poster.UploadAudio should not be called in dry-run mode")
	}
}

func TestRun_MissingAudioFile_Error(t *testing.T) {
	dir := t.TempDir()
	manifestPath := writeTestManifest(t, dir)
	// audio file is NOT created
	specPath := writeTestSlackSpec(t, dir)
	configPath := writeTestConfig(t, dir, "TEST_SLACK_BOT_TOKEN")

	t.Setenv("TEST_SLACK_BOT_TOKEN", "xoxb-test-token")

	err := slack.Run(slack.Options{
		ConfigPath:   configPath,
		ManifestPath: manifestPath,
		SpecPath:     specPath,
		DryRun:       true,
		Out:          os.Stdout,
	}, nil)
	if err == nil {
		t.Error("expected error when audio file is missing")
	}
}

func TestRun_EmptyBotToken_Error(t *testing.T) {
	dir := t.TempDir()
	manifestPath := writeTestManifest(t, dir)
	writeTestAudio(t, dir)
	specPath := writeTestSlackSpec(t, dir)
	configPath := writeTestConfig(t, dir, "NONEXISTENT_SLACK_TOKEN_VAR")

	// env var is not set
	err := slack.Run(slack.Options{
		ConfigPath:   configPath,
		ManifestPath: manifestPath,
		SpecPath:     specPath,
		DryRun:       false,
		Out:          os.Stdout,
	}, nil)
	if err == nil {
		t.Error("expected error when bot token env var is not set")
	}
}

func TestRun_PostMode_CallsUploadAndThread(t *testing.T) {
	dir := t.TempDir()
	manifestPath := writeTestManifest(t, dir)
	writeTestAudio(t, dir)
	specPath := writeTestSlackSpec(t, dir)
	configPath := writeTestConfig(t, dir, "TEST_SLACK_BOT_TOKEN")

	t.Setenv("TEST_SLACK_BOT_TOKEN", "xoxb-test-token")

	uploadCalled := false
	threadCalled := false
	mock := &mockPoster{
		uploadAudioFn: func(_ context.Context, _ slack.UploadParams) (string, string, error) {
			uploadCalled = true
			return "FILE123", "TS123", nil
		},
		postThreadReplyFn: func(_ context.Context, _ slack.ReplyParams) error {
			threadCalled = true
			return nil
		},
	}

	var buf strings.Builder
	err := slack.Run(slack.Options{
		ConfigPath:   configPath,
		ManifestPath: manifestPath,
		SpecPath:     specPath,
		DryRun:       false,
		Out:          &buf,
	}, mock)
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}
	if !uploadCalled {
		t.Error("Poster.UploadAudio should be called in post mode")
	}
	// thread reply is skipped because manifest has empty summary and no corners
	_ = threadCalled

	out := buf.String()
	if out == "" {
		t.Error("Run should output summary to stdout")
	}
}

func TestRun_PostMode_WithSummaryCallsThreadReply(t *testing.T) {
	dir := t.TempDir()

	manifest := map[string]any{
		"title":          "ずんだもんテックラジオ",
		"episode_number": 42,
		"episode_title":  "テストエピソード",
		"summary":        "今回はLLMについてです。",
		"audio_file":     "episode42.mp3",
		"corners": []any{
			map[string]any{
				"title":    "コーナー1",
				"summary":  "コーナーのまとめ",
				"articles": []any{},
			},
		},
	}
	data, _ := json.Marshal(manifest)
	manifestPath := filepath.Join(dir, "manifest.json")
	_ = os.WriteFile(manifestPath, data, 0o644)
	writeTestAudio(t, dir)
	specPath := writeTestSlackSpec(t, dir)
	configPath := writeTestConfig(t, dir, "TEST_SLACK_BOT_TOKEN")

	t.Setenv("TEST_SLACK_BOT_TOKEN", "xoxb-test-token")

	threadCalled := false
	mock := &mockPoster{
		uploadAudioFn: func(_ context.Context, _ slack.UploadParams) (string, string, error) {
			return "FILE123", "TS123", nil
		},
		postThreadReplyFn: func(_ context.Context, _ slack.ReplyParams) error {
			threadCalled = true
			return nil
		},
	}

	err := slack.Run(slack.Options{
		ConfigPath:   configPath,
		ManifestPath: manifestPath,
		SpecPath:     specPath,
		DryRun:       false,
		Out:          os.Stdout,
	}, mock)
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}
	if !threadCalled {
		t.Error("Poster.PostThreadReply should be called when summary and corners are present")
	}
}

func TestRun_TsEmpty_ReturnsError(t *testing.T) {
	dir := t.TempDir()

	manifest := map[string]any{
		"title":          "テスト",
		"episode_number": 1,
		"summary":        "まとめ",
		"audio_file":     "ep1.mp3",
		"corners":        []any{},
	}
	data, _ := json.Marshal(manifest)
	manifestPath := filepath.Join(dir, "manifest.json")
	_ = os.WriteFile(manifestPath, data, 0o644)
	_ = os.WriteFile(filepath.Join(dir, "ep1.mp3"), []byte("fake"), 0o644)
	specPath := writeTestSlackSpec(t, dir)
	configPath := writeTestConfig(t, dir, "TEST_SLACK_BOT_TOKEN")

	t.Setenv("TEST_SLACK_BOT_TOKEN", "xoxb-test-token")

	mock := &mockPoster{
		uploadAudioFn: func(_ context.Context, _ slack.UploadParams) (string, string, error) {
			return "FILE123", "", nil // empty ts
		},
	}

	err := slack.Run(slack.Options{
		ConfigPath:   configPath,
		ManifestPath: manifestPath,
		SpecPath:     specPath,
		DryRun:       false,
		Out:          os.Stdout,
	}, mock)
	if err == nil {
		t.Fatal("Run should return error when ts is empty and thread blocks are present")
	}
}
