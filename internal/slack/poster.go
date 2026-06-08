package slack

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	slackgo "github.com/slack-go/slack"
)

const doublePostWarning = "audio is already uploaded — re-running will 二重投稿"

// UploadParams holds parameters for uploading an audio file to Slack.
type UploadParams struct {
	Channel        string
	FilePath       string
	Title          string
	Filename       string
	InitialComment string
}

// ReplyParams holds parameters for posting a thread reply.
type ReplyParams struct {
	Channel  string
	ThreadTS string
	Blocks   []slackgo.Block
	Text     string
}

// Poster is an interface for sending Slack messages.
type Poster interface {
	// UploadAudio uploads an mp3 file as the parent message and returns the file ID.
	UploadAudio(ctx context.Context, u UploadParams) (fileID string, err error)
	// ResolveThreadTS polls files.info until the thread ts is available for the given file.
	ResolveThreadTS(ctx context.Context, fileID, channel string) (ts string, err error)
	// PostThreadReply posts a thread reply with Block Kit blocks.
	PostThreadReply(ctx context.Context, p ReplyParams) error
}

// realPoster is the production implementation of Poster backed by slack-go.
type realPoster struct {
	client       *slackgo.Client
	pollInterval time.Duration
	pollTimeout  time.Duration
}

// NewPoster creates a Poster backed by the real Slack API.
func NewPoster(token string) Poster {
	return &realPoster{
		client:       slackgo.New(token),
		pollInterval: time.Second,
		pollTimeout:  30 * time.Second,
	}
}

func (r *realPoster) UploadAudio(ctx context.Context, u UploadParams) (string, error) {
	f, err := os.Open(u.FilePath)
	if err != nil {
		return "", fmt.Errorf("open audio file: %w", err)
	}
	defer func() { _ = f.Close() }()

	info, err := f.Stat()
	if err != nil {
		return "", fmt.Errorf("stat audio file: %w", err)
	}

	summary, err := r.client.UploadFileContext(ctx, slackgo.UploadFileParameters{
		Reader:         f,
		FileSize:       int(info.Size()),
		Filename:       u.Filename,
		Title:          u.Title,
		Channel:        u.Channel,
		InitialComment: u.InitialComment,
	})
	if err != nil {
		return "", fmt.Errorf("upload audio: %w", err)
	}
	return summary.ID, nil
}

func (r *realPoster) ResolveThreadTS(ctx context.Context, fileID, channel string) (string, error) {
	return r.pollForTS(ctx, fileID, channel)
}

// nonRetryableCodes is the set of Slack API error codes that will never resolve on retry.
var nonRetryableCodes = map[string]struct{}{
	"missing_scope":    {},
	"not_authed":       {},
	"invalid_auth":     {},
	"account_inactive": {},
	"token_revoked":    {},
	"token_expired":    {},
	"no_permission":    {},
	"file_not_found":   {},
	"file_deleted":     {},
}

func isNonRetryable(err error) bool {
	var slackErr slackgo.SlackErrorResponse
	if !errors.As(err, &slackErr) {
		return false
	}
	_, ok := nonRetryableCodes[slackErr.Err]
	return ok
}

func (r *realPoster) pollForTS(ctx context.Context, fileID, channel string) (string, error) {
	pollCtx, cancel := context.WithTimeout(ctx, r.pollTimeout)
	defer cancel()

	timer := time.NewTimer(r.pollInterval)
	defer timer.Stop()

	var lastErr error
	for {
		fileInfo, _, _, err := r.client.GetFileInfoContext(pollCtx, fileID, 0, 0)
		if err != nil {
			if isNonRetryable(err) {
				return "", r.nonRetryableError(fileID, err)
			}
			lastErr = err
		} else {
			if shares, ok := fileInfo.Shares.Public[channel]; ok && len(shares) > 0 {
				return shares[0].Ts, nil
			}
			if shares, ok := fileInfo.Shares.Private[channel]; ok && len(shares) > 0 {
				return shares[0].Ts, nil
			}
		}

		select {
		case <-pollCtx.Done():
			return "", r.timeoutError(fileID, lastErr)
		case <-timer.C:
			timer.Reset(r.pollInterval)
		}
	}
}

func pollingTerminalError(fileID, reason string, cause error) error {
	msg := fmt.Sprintf("%s waiting for ts (file_id=%s): %s", reason, fileID, doublePostWarning)
	if cause != nil {
		return fmt.Errorf("%s: %w", msg, cause)
	}
	return errors.New(msg)
}

func (r *realPoster) nonRetryableError(fileID string, err error) error {
	return pollingTerminalError(fileID, "non-retryable Slack error", err)
}

func (r *realPoster) timeoutError(fileID string, lastErr error) error {
	return pollingTerminalError(fileID, "timed out", lastErr)
}

func (r *realPoster) PostThreadReply(ctx context.Context, p ReplyParams) error {
	_, _, err := r.client.PostMessageContext(ctx, p.Channel,
		slackgo.MsgOptionTS(p.ThreadTS),
		slackgo.MsgOptionBlocks(p.Blocks...),
		slackgo.MsgOptionText(p.Text, false),
	)
	if err != nil {
		return fmt.Errorf("post thread reply: %w", err)
	}
	return nil
}
