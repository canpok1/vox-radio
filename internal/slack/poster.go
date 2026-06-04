package slack

import (
	"context"
	"fmt"
	"os"

	slackgo "github.com/slack-go/slack"
)

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
	// UploadAudio uploads an mp3 file as the parent message and returns the file ID and thread ts.
	UploadAudio(ctx context.Context, u UploadParams) (fileID, ts string, err error)
	// PostThreadReply posts a thread reply with Block Kit blocks.
	PostThreadReply(ctx context.Context, p ReplyParams) error
}

// realPoster is the production implementation of Poster backed by slack-go.
type realPoster struct {
	client *slackgo.Client
}

// NewPoster creates a Poster backed by the real Slack API.
func NewPoster(token string) Poster {
	return &realPoster{client: slackgo.New(token)}
}

func (r *realPoster) UploadAudio(ctx context.Context, u UploadParams) (string, string, error) {
	f, err := os.Open(u.FilePath)
	if err != nil {
		return "", "", fmt.Errorf("open audio file: %w", err)
	}
	defer func() { _ = f.Close() }()

	info, err := f.Stat()
	if err != nil {
		return "", "", fmt.Errorf("stat audio file: %w", err)
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
		return "", "", fmt.Errorf("upload audio: %w", err)
	}

	fileInfo, _, _, err := r.client.GetFileInfoContext(ctx, summary.ID, 0, 0)
	if err != nil {
		return summary.ID, "", nil
	}

	ts := ""
	if shares, ok := fileInfo.Shares.Public[u.Channel]; ok && len(shares) > 0 {
		ts = shares[0].Ts
	}
	return summary.ID, ts, nil
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
