package slack_test

import (
	"context"
	"encoding/json"
	"errors"
	"io"
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
		"summary":        "",
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

// writeTestManifestWithSummary creates a manifest that generates non-empty thread blocks.
func writeTestManifestWithSummary(t *testing.T, dir string) string {
	t.Helper()
	manifest := map[string]any{
		"title":          "ずんだもんテックラジオ",
		"episode_number": 42,
		"episode_title":  "テストエピソード",
		"summary":        "今回はLLMについてです。",
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
		uploadAudioFn: func(_ context.Context, _ slack.UploadParams) (string, error) {
			posterCalled = true
			return "FILE123", nil
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
	mock := &mockPoster{
		uploadAudioFn: func(_ context.Context, _ slack.UploadParams) (string, error) {
			uploadCalled = true
			return "FILE123", nil
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
		uploadAudioFn: func(_ context.Context, _ slack.UploadParams) (string, error) {
			return "FILE123", nil
		},
		resolveThreadTSFn: func(_ context.Context, _, _ string) (string, error) {
			return "TS123", nil
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

func TestRun_ResolveThreadTSFails_SavesStateAndReturnsError(t *testing.T) {
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
		uploadAudioFn: func(_ context.Context, _ slack.UploadParams) (string, error) {
			return "FILE123", nil
		},
		resolveThreadTSFn: func(_ context.Context, _, _ string) (string, error) {
			return "", errors.New("poll timeout")
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
		t.Fatal("Run should return error when ResolveThreadTS fails")
	}

	// State file should exist with fileID saved for resume
	statePath := slack.DefaultStatePath(manifestPath)
	stateData, err2 := os.ReadFile(statePath)
	if err2 != nil {
		t.Fatalf("state file should exist after upload: %v", err2)
	}
	var state map[string]any
	if err2 := json.Unmarshal(stateData, &state); err2 != nil {
		t.Fatalf("parse state file: %v", err2)
	}
	if state["file_id"] != "FILE123" {
		t.Errorf("state file should have file_id=FILE123, got %v", state)
	}
	if replied, _ := state["replied"].(bool); replied {
		t.Error("state file should have replied=false when ts resolution failed")
	}
}

// ① 正常系で replied:true まで進む
func TestRun_StateFile_HappyPath_RepliedTrue(t *testing.T) {
	dir := t.TempDir()
	manifestPath := writeTestManifestWithSummary(t, dir)
	writeTestAudio(t, dir)
	specPath := writeTestSlackSpec(t, dir)
	configPath := writeTestConfig(t, dir, "TEST_SLACK_BOT_TOKEN")

	t.Setenv("TEST_SLACK_BOT_TOKEN", "xoxb-test-token")

	mock := &mockPoster{
		uploadAudioFn: func(_ context.Context, _ slack.UploadParams) (string, error) {
			return "FILE123", nil
		},
		resolveThreadTSFn: func(_ context.Context, _, _ string) (string, error) {
			return "TS123", nil
		},
		postThreadReplyFn: func(_ context.Context, _ slack.ReplyParams) error {
			return nil
		},
	}

	err := slack.Run(slack.Options{
		ConfigPath:   configPath,
		ManifestPath: manifestPath,
		SpecPath:     specPath,
		Out:          io.Discard,
	}, mock)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	statePath := slack.DefaultStatePath(manifestPath)
	stateData, err := os.ReadFile(statePath)
	if err != nil {
		t.Fatalf("state file should exist after successful run: %v", err)
	}
	var state map[string]any
	if err := json.Unmarshal(stateData, &state); err != nil {
		t.Fatalf("parse state file: %v", err)
	}
	if replied, _ := state["replied"].(bool); !replied {
		t.Errorf("state file should have replied=true, got %v", state)
	}
}

// ② ts タイムアウト後の再実行で再アップロードせず返信が投稿
func TestRun_ResumeAfterTSTimeout_SkipsUploadAndPostsReply(t *testing.T) {
	dir := t.TempDir()
	manifestPath := writeTestManifestWithSummary(t, dir)
	writeTestAudio(t, dir)
	specPath := writeTestSlackSpec(t, dir)
	configPath := writeTestConfig(t, dir, "TEST_SLACK_BOT_TOKEN")

	t.Setenv("TEST_SLACK_BOT_TOKEN", "xoxb-test-token")

	uploadCount := 0
	resolveCount := 0
	threadCount := 0

	mock := &mockPoster{
		uploadAudioFn: func(_ context.Context, _ slack.UploadParams) (string, error) {
			uploadCount++
			return "FILE123", nil
		},
		resolveThreadTSFn: func(_ context.Context, _, _ string) (string, error) {
			resolveCount++
			if resolveCount == 1 {
				return "", errors.New("poll timeout")
			}
			return "TS123", nil
		},
		postThreadReplyFn: func(_ context.Context, _ slack.ReplyParams) error {
			threadCount++
			return nil
		},
	}

	// First run: upload succeeds, ts resolution fails
	err := slack.Run(slack.Options{
		ConfigPath:   configPath,
		ManifestPath: manifestPath,
		SpecPath:     specPath,
		Out:          io.Discard,
	}, mock)
	if err == nil {
		t.Error("expected error on first run (ts timeout)")
	}
	if uploadCount != 1 {
		t.Errorf("UploadAudio should be called once on first run, got %d", uploadCount)
	}
	if threadCount != 0 {
		t.Errorf("PostThreadReply should not be called on first run, got %d", threadCount)
	}

	// Second run: resume from state file, skip upload
	err = slack.Run(slack.Options{
		ConfigPath:   configPath,
		ManifestPath: manifestPath,
		SpecPath:     specPath,
		Out:          io.Discard,
	}, mock)
	if err != nil {
		t.Fatalf("expected no error on second run: %v", err)
	}
	if uploadCount != 1 {
		t.Errorf("UploadAudio should NOT be called on second run (resume), still got %d", uploadCount)
	}
	if threadCount != 1 {
		t.Errorf("PostThreadReply should be called exactly once on second run, got %d", threadCount)
	}
}

// ③ replied:true 状態での再実行が何も投稿しない
func TestRun_SkipsAllWhenAlreadyReplied(t *testing.T) {
	dir := t.TempDir()
	manifestPath := writeTestManifestWithSummary(t, dir)
	writeTestAudio(t, dir)
	specPath := writeTestSlackSpec(t, dir)
	configPath := writeTestConfig(t, dir, "TEST_SLACK_BOT_TOKEN")

	t.Setenv("TEST_SLACK_BOT_TOKEN", "xoxb-test-token")

	// Write state file with replied=true
	statePath := slack.DefaultStatePath(manifestPath)
	stateJSON := `{"audio_file":"episode42.mp3","episode_number":42,"channel":"C0123456789","file_id":"FILE_OLD","thread_ts":"TS_OLD","replied":true}`
	_ = os.WriteFile(statePath, []byte(stateJSON), 0o644)

	uploadCalled := false
	threadCalled := false
	mock := &mockPoster{
		uploadAudioFn: func(_ context.Context, _ slack.UploadParams) (string, error) {
			uploadCalled = true
			return "FILE_NEW", nil
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
		Out:          &buf,
	}, mock)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if uploadCalled {
		t.Error("UploadAudio should not be called when already replied")
	}
	if threadCalled {
		t.Error("PostThreadReply should not be called when already replied")
	}

	// Output should contain the state values
	out := buf.String()
	if !strings.Contains(out, "FILE_OLD") {
		t.Errorf("output should contain file_id from state, got: %q", out)
	}
}

// ④ audio_file 不一致の古い状態を無視
func TestRun_IgnoresStateMismatch(t *testing.T) {
	dir := t.TempDir()
	manifestPath := writeTestManifest(t, dir) // audio_file = "episode42.mp3"
	writeTestAudio(t, dir)
	specPath := writeTestSlackSpec(t, dir)
	configPath := writeTestConfig(t, dir, "TEST_SLACK_BOT_TOKEN")

	t.Setenv("TEST_SLACK_BOT_TOKEN", "xoxb-test-token")

	// Write state file with different audio_file
	statePath := slack.DefaultStatePath(manifestPath)
	stateJSON := `{"audio_file":"different-episode.mp3","file_id":"FILE_OLD","replied":false}`
	_ = os.WriteFile(statePath, []byte(stateJSON), 0o644)

	uploadCalled := false
	mock := &mockPoster{
		uploadAudioFn: func(_ context.Context, _ slack.UploadParams) (string, error) {
			uploadCalled = true
			return "FILE_NEW", nil
		},
	}

	err := slack.Run(slack.Options{
		ConfigPath:   configPath,
		ManifestPath: manifestPath,
		SpecPath:     specPath,
		Out:          io.Discard,
	}, mock)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !uploadCalled {
		t.Error("UploadAudio should be called when state audio_file doesn't match")
	}
}

// ⑥ dry-run で状態ファイル不介入
func TestRun_DryRun_NoStateFile(t *testing.T) {
	dir := t.TempDir()
	manifestPath := writeTestManifest(t, dir)
	writeTestAudio(t, dir)
	specPath := writeTestSlackSpec(t, dir)
	configPath := writeTestConfig(t, dir, "TEST_SLACK_BOT_TOKEN")

	t.Setenv("TEST_SLACK_BOT_TOKEN", "xoxb-test-token")

	err := slack.Run(slack.Options{
		ConfigPath:   configPath,
		ManifestPath: manifestPath,
		SpecPath:     specPath,
		DryRun:       true,
		Out:          io.Discard,
	}, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	statePath := slack.DefaultStatePath(manifestPath)
	if _, err := os.Stat(statePath); !os.IsNotExist(err) {
		t.Error("state file should not exist after dry-run")
	}
}
