package slack_test

import (
	"context"
	"errors"
	"testing"

	slackgo "github.com/slack-go/slack"

	"github.com/canpok1/vox-radio/internal/slack"
)

// mockPoster is a test double for the Poster interface.
type mockPoster struct {
	verifyScopesFn    func(ctx context.Context, required []string) error
	uploadAudioFn     func(ctx context.Context, u slack.UploadParams) (string, error)
	resolveThreadTSFn func(ctx context.Context, fileID, channel string) (string, error)
	postThreadReplyFn func(ctx context.Context, p slack.ReplyParams) error
}

func (m *mockPoster) VerifyScopes(ctx context.Context, required []string) error {
	if m.verifyScopesFn != nil {
		return m.verifyScopesFn(ctx, required)
	}
	return nil
}

func (m *mockPoster) UploadAudio(ctx context.Context, u slack.UploadParams) (string, error) {
	if m.uploadAudioFn != nil {
		return m.uploadAudioFn(ctx, u)
	}
	return "FILE123", nil
}

func (m *mockPoster) ResolveThreadTS(ctx context.Context, fileID, channel string) (string, error) {
	if m.resolveThreadTSFn != nil {
		return m.resolveThreadTSFn(ctx, fileID, channel)
	}
	return "1234567890.123456", nil
}

func (m *mockPoster) PostThreadReply(ctx context.Context, p slack.ReplyParams) error {
	if m.postThreadReplyFn != nil {
		return m.postThreadReplyFn(ctx, p)
	}
	return nil
}

func TestPoster_Interface(t *testing.T) {
	var _ slack.Poster = &mockPoster{}
}

func TestUploadParams_Fields(t *testing.T) {
	params := slack.UploadParams{
		Channel:        "C0123456789",
		FilePath:       "/path/to/test-prog_ep001.mp3",
		Title:          "Episode Title",
		Filename:       "test-prog_ep001.mp3",
		InitialComment: "🎙️ テスト番組",
	}
	if params.Channel == "" {
		t.Error("Channel must not be empty")
	}
	if params.FilePath == "" {
		t.Error("FilePath must not be empty")
	}
}

func TestReplyParams_Fields(t *testing.T) {
	blocks := []slackgo.Block{
		slackgo.NewSectionBlock(
			slackgo.NewTextBlockObject(slackgo.MarkdownType, "test", false, false),
			nil, nil,
		),
	}
	params := slack.ReplyParams{
		Channel:  "C0123456789",
		ThreadTS: "1234567890.123456",
		Blocks:   blocks,
		Text:     "fallback",
	}
	if params.Channel == "" {
		t.Error("Channel must not be empty")
	}
	if params.ThreadTS == "" {
		t.Error("ThreadTS must not be empty")
	}
	if len(params.Blocks) == 0 {
		t.Error("Blocks must not be empty")
	}
}

func TestMockPoster_UploadAudio_ReturnsID(t *testing.T) {
	mock := &mockPoster{
		uploadAudioFn: func(_ context.Context, _ slack.UploadParams) (string, error) {
			return "FILE_ID_123", nil
		},
	}

	fileID, err := mock.UploadAudio(context.Background(), slack.UploadParams{
		Channel:  "C0123456789",
		FilePath: "/path/to/ep.mp3",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if fileID != "FILE_ID_123" {
		t.Errorf("fileID = %q, want %q", fileID, "FILE_ID_123")
	}
}

func TestMockPoster_UploadAudio_Error(t *testing.T) {
	mock := &mockPoster{
		uploadAudioFn: func(_ context.Context, _ slack.UploadParams) (string, error) {
			return "", errors.New("upload failed")
		},
	}

	_, err := mock.UploadAudio(context.Background(), slack.UploadParams{})
	if err == nil {
		t.Error("expected error")
	}
}

func TestMockPoster_ResolveThreadTS_ReturnsTS(t *testing.T) {
	mock := &mockPoster{
		resolveThreadTSFn: func(_ context.Context, _, _ string) (string, error) {
			return "TS_123", nil
		},
	}

	ts, err := mock.ResolveThreadTS(context.Background(), "FILE_ID", "C0123456789")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ts != "TS_123" {
		t.Errorf("ts = %q, want %q", ts, "TS_123")
	}
}

func TestMockPoster_PostThreadReply_Success(t *testing.T) {
	called := false
	mock := &mockPoster{
		postThreadReplyFn: func(_ context.Context, _ slack.ReplyParams) error {
			called = true
			return nil
		},
	}

	err := mock.PostThreadReply(context.Background(), slack.ReplyParams{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !called {
		t.Error("PostThreadReply should have been called")
	}
}
