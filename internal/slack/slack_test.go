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

	"github.com/canpok1/vox-radio/internal/model"
	"github.com/canpok1/vox-radio/internal/slack"
)

func buildTestManifest() model.Manifest {
	return model.Manifest{
		Title:         "ずんだもんテックラジオ",
		EpisodeNumber: 42,
		EpisodeTitle:  "テストエピソード",
		Summary:       "",
		AudioFile:     "episode42.mp3",
		Corners:       []model.ManifestCorner{},
	}
}

func buildTestManifestWithSummary() model.Manifest {
	return model.Manifest{
		Title:         "ずんだもんテックラジオ",
		EpisodeNumber: 42,
		EpisodeTitle:  "テストエピソード",
		Summary:       "今回はLLMについてです。",
		AudioFile:     "episode42.mp3",
		Corners:       []model.ManifestCorner{},
	}
}

func buildTestSlackSpec() slack.SlackSpec {
	return slack.SlackSpec{
		Slack: slack.SlackChannelConfig{
			ChannelEnv: "SLACK_CHANNEL_ID",
		},
	}
}

func writeTestAudio(t *testing.T, dir string) {
	t.Helper()
	path := filepath.Join(dir, "episode42.mp3")
	if err := os.WriteFile(path, []byte("fake mp3"), 0o644); err != nil {
		t.Fatalf("write audio: %v", err)
	}
}

func TestRun_DryRun_OutputsAudioPathAndComment(t *testing.T) {
	dir := t.TempDir()
	writeTestAudio(t, dir)
	audioPath := filepath.Join(dir, "episode42.mp3")

	var buf strings.Builder
	err := slack.Run(slack.Options{
		Manifest:  buildTestManifest(),
		AudioPath: audioPath,
		Spec:      buildTestSlackSpec(),
		DryRun:    true,
		Out:       &buf,
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
	writeTestAudio(t, dir)
	audioPath := filepath.Join(dir, "episode42.mp3")

	posterCalled := false
	mock := &mockPoster{
		uploadAudioFn: func(_ context.Context, _ slack.UploadParams) (string, error) {
			posterCalled = true
			return "FILE123", nil
		},
	}

	err := slack.Run(slack.Options{
		Manifest:  buildTestManifest(),
		AudioPath: audioPath,
		Spec:      buildTestSlackSpec(),
		DryRun:    true,
		Out:       os.Stdout,
	}, mock)
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}
	if posterCalled {
		t.Error("Poster.UploadAudio should not be called in dry-run mode")
	}
}

// dry-run は音声ファイルの存在チェックをスキップするため、ファイルがなくても成功する
func TestRun_DryRun_SucceedsWithMissingAudioFile(t *testing.T) {
	dir := t.TempDir()
	audioPath := filepath.Join(dir, "nonexistent.mp3")

	err := slack.Run(slack.Options{
		Manifest:  buildTestManifest(),
		AudioPath: audioPath,
		Spec:      buildTestSlackSpec(),
		DryRun:    true,
		Out:       io.Discard,
	}, nil)
	if err != nil {
		t.Errorf("dry-run should succeed even when audio file is missing, got: %v", err)
	}
}

func TestRun_MissingAudioFile_Error(t *testing.T) {
	dir := t.TempDir()
	audioPath := filepath.Join(dir, "nonexistent.mp3")

	err := slack.Run(slack.Options{
		Manifest:  buildTestManifest(),
		AudioPath: audioPath,
		Spec:      buildTestSlackSpec(),
		Token:     "xoxb-test-token",
		Channel:   "C0123456789",
		StatePath: filepath.Join(dir, "state.json"),
		DryRun:    false,
		Out:       os.Stdout,
	}, nil)
	if err == nil {
		t.Error("expected error when audio file is missing")
	}
}

func TestRun_PostMode_CallsUploadAndThread(t *testing.T) {
	dir := t.TempDir()
	writeTestAudio(t, dir)
	audioPath := filepath.Join(dir, "episode42.mp3")

	uploadCalled := false
	mock := &mockPoster{
		uploadAudioFn: func(_ context.Context, _ slack.UploadParams) (string, error) {
			uploadCalled = true
			return "FILE123", nil
		},
	}

	var buf strings.Builder
	err := slack.Run(slack.Options{
		Manifest:  buildTestManifest(),
		AudioPath: audioPath,
		Spec:      buildTestSlackSpec(),
		Token:     "xoxb-test-token",
		Channel:   "C0123456789",
		DryRun:    false,
		Out:       &buf,
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
	writeTestAudio(t, dir)
	audioPath := filepath.Join(dir, "episode42.mp3")

	manifest := model.Manifest{
		Title:         "ずんだもんテックラジオ",
		EpisodeNumber: 42,
		EpisodeTitle:  "テストエピソード",
		Summary:       "今回はLLMについてです。",
		AudioFile:     "episode42.mp3",
		Corners: []model.ManifestCorner{
			{
				Title:   "コーナー1",
				Summary: "コーナーのまとめ",
			},
		},
	}

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
		Manifest:  manifest,
		AudioPath: audioPath,
		Spec:      buildTestSlackSpec(),
		Token:     "xoxb-test-token",
		Channel:   "C0123456789",
		DryRun:    false,
		Out:       os.Stdout,
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
	audioPath := filepath.Join(dir, "ep1.mp3")
	_ = os.WriteFile(audioPath, []byte("fake"), 0o644)

	manifest := model.Manifest{
		Title:         "テスト",
		EpisodeNumber: 1,
		Summary:       "まとめ",
		AudioFile:     "ep1.mp3",
		Corners:       []model.ManifestCorner{},
	}
	statePath := filepath.Join(dir, "state.json")

	mock := &mockPoster{
		uploadAudioFn: func(_ context.Context, _ slack.UploadParams) (string, error) {
			return "FILE123", nil
		},
		resolveThreadTSFn: func(_ context.Context, _, _ string) (string, error) {
			return "", errors.New("poll timeout")
		},
	}

	err := slack.Run(slack.Options{
		Manifest:  manifest,
		AudioPath: audioPath,
		Spec:      buildTestSlackSpec(),
		Token:     "xoxb-test-token",
		Channel:   "C0123456789",
		StatePath: statePath,
		DryRun:    false,
		Out:       os.Stdout,
	}, mock)
	if err == nil {
		t.Fatal("Run should return error when ResolveThreadTS fails")
	}

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
	writeTestAudio(t, dir)
	audioPath := filepath.Join(dir, "episode42.mp3")
	statePath := filepath.Join(dir, "state.json")

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
		Manifest:  buildTestManifestWithSummary(),
		AudioPath: audioPath,
		Spec:      buildTestSlackSpec(),
		Token:     "xoxb-test-token",
		Channel:   "C0123456789",
		StatePath: statePath,
		Out:       io.Discard,
	}, mock)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

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
	writeTestAudio(t, dir)
	audioPath := filepath.Join(dir, "episode42.mp3")
	statePath := filepath.Join(dir, "state.json")

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

	opts := slack.Options{
		Manifest:  buildTestManifestWithSummary(),
		AudioPath: audioPath,
		Spec:      buildTestSlackSpec(),
		Token:     "xoxb-test-token",
		Channel:   "C0123456789",
		StatePath: statePath,
		Out:       io.Discard,
	}

	err := slack.Run(opts, mock)
	if err == nil {
		t.Error("expected error on first run (ts timeout)")
	}
	if uploadCount != 1 {
		t.Errorf("UploadAudio should be called once on first run, got %d", uploadCount)
	}
	if threadCount != 0 {
		t.Errorf("PostThreadReply should not be called on first run, got %d", threadCount)
	}

	err = slack.Run(opts, mock)
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
	writeTestAudio(t, dir)
	audioPath := filepath.Join(dir, "episode42.mp3")
	statePath := filepath.Join(dir, "state.json")

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
		Manifest:  buildTestManifestWithSummary(),
		AudioPath: audioPath,
		Spec:      buildTestSlackSpec(),
		Token:     "xoxb-test-token",
		Channel:   "C0123456789",
		StatePath: statePath,
		Out:       &buf,
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

	out := buf.String()
	if !strings.Contains(out, "FILE_OLD") {
		t.Errorf("output should contain file_id from state, got: %q", out)
	}
}

// ④ audio_file 不一致の古い状態を無視
func TestRun_IgnoresStateMismatch(t *testing.T) {
	dir := t.TempDir()
	writeTestAudio(t, dir)
	audioPath := filepath.Join(dir, "episode42.mp3")
	statePath := filepath.Join(dir, "state.json")

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
		Manifest:  buildTestManifest(),
		AudioPath: audioPath,
		Spec:      buildTestSlackSpec(),
		Token:     "xoxb-test-token",
		Channel:   "C0123456789",
		StatePath: statePath,
		Out:       io.Discard,
	}, mock)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !uploadCalled {
		t.Error("UploadAudio should be called when state audio_file doesn't match")
	}
}

// ⑤ episode_number が異なる場合は前の状態を無視して新規投稿
func TestRun_IgnoresStateWithDifferentEpisodeNumber(t *testing.T) {
	dir := t.TempDir()
	audioPath := filepath.Join(dir, "episode14.mp3")
	_ = os.WriteFile(audioPath, []byte("fake mp3"), 0o644)
	statePath := filepath.Join(dir, "state.json")

	manifest := model.Manifest{
		Title:         "ずんだもんテックラジオ",
		EpisodeNumber: 14,
		EpisodeTitle:  "エピソード14",
		Summary:       "",
		AudioFile:     "episode14.mp3",
		Corners:       []model.ManifestCorner{},
	}

	stateJSON := `{"audio_file":"episode13.mp3","episode_number":13,"channel":"C_OLD","file_id":"FILE_OLD","thread_ts":"TS_OLD","replied":true}`
	_ = os.WriteFile(statePath, []byte(stateJSON), 0o644)

	uploadCalled := false
	mock := &mockPoster{
		uploadAudioFn: func(_ context.Context, _ slack.UploadParams) (string, error) {
			uploadCalled = true
			return "FILE_NEW", nil
		},
	}

	var buf strings.Builder
	if err := slack.Run(slack.Options{
		Manifest:  manifest,
		AudioPath: audioPath,
		Spec:      buildTestSlackSpec(),
		Token:     "xoxb-test-token",
		Channel:   "C0123456789",
		StatePath: statePath,
		Out:       &buf,
	}, mock); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !uploadCalled {
		t.Error("UploadAudio should be called when episode_number differs from state")
	}
	out := buf.String()
	if strings.Contains(out, "FILE_OLD") {
		t.Errorf("output should not contain old file_id from mismatched state, got: %q", out)
	}
}

// ⑥ dry-run で状態ファイル不介入
func TestRun_DryRun_NoStateFile(t *testing.T) {
	dir := t.TempDir()
	writeTestAudio(t, dir)
	audioPath := filepath.Join(dir, "episode42.mp3")
	statePath := filepath.Join(dir, "state.json")

	err := slack.Run(slack.Options{
		Manifest:  buildTestManifest(),
		AudioPath: audioPath,
		Spec:      buildTestSlackSpec(),
		StatePath: statePath,
		DryRun:    true,
		Out:       io.Discard,
	}, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if _, err := os.Stat(statePath); !os.IsNotExist(err) {
		t.Error("state file should not exist after dry-run")
	}
}
