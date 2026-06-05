package slack

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/canpok1/vox-radio/internal/config"
	"github.com/canpok1/vox-radio/internal/fileio"
	"github.com/canpok1/vox-radio/internal/model"
)

// Options holds the inputs for Run.
type Options struct {
	ConfigPath   string
	ManifestPath string
	SpecPath     string
	DryRun       bool
	Out          io.Writer
}

// Run executes the slackpost workflow.
// poster is injected for testing; pass nil to use the real Poster (requires valid token).
func Run(opts Options, poster Poster) error {
	if opts.Out == nil {
		opts.Out = os.Stdout
	}

	cfg, err := config.LoadConfig(opts.ConfigPath)
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	token := os.Getenv(cfg.Slack.BotTokenEnv)
	if token == "" && !opts.DryRun {
		return fmt.Errorf("bot token env var %q is not set", cfg.Slack.BotTokenEnv)
	}

	var manifest model.Manifest
	if err := fileio.ReadJSON(opts.ManifestPath, &manifest); err != nil {
		return fmt.Errorf("load manifest: %w", err)
	}

	audioPath := filepath.Join(filepath.Dir(opts.ManifestPath), manifest.AudioFile)
	if _, err := os.Stat(audioPath); err != nil {
		return fmt.Errorf("audio file not found: %w", err)
	}

	spec, err := model.LoadSlackSpec(opts.SpecPath)
	if err != nil {
		return fmt.Errorf("load slack spec: %w", err)
	}
	if err := model.ValidateSlackSpec(spec); err != nil {
		return fmt.Errorf("validate slack spec: %w", err)
	}

	tmpl := spec.Slack.EffectiveMessageTemplate()
	header := BuildHeader(manifest, tmpl)
	blocks, fallback := BuildThreadBlocks(manifest, tmpl)
	audioTitle := BuildAudioTitle(manifest)

	if opts.DryRun {
		_, _ = fmt.Fprintf(opts.Out, "audio: %s\n", audioPath)
		_, _ = fmt.Fprintf(opts.Out, "header: %s\n", header)
		if len(blocks) > 0 {
			blocksJSON, _ := json.MarshalIndent(blocks, "", "  ")
			_, _ = fmt.Fprintf(opts.Out, "thread blocks:\n%s\n", blocksJSON)
		}
		return nil
	}

	if poster == nil {
		poster = NewPoster(token)
	}

	ctx := context.Background()

	fileID, ts, err := poster.UploadAudio(ctx, UploadParams{
		Channel:        spec.Slack.Channel,
		FilePath:       audioPath,
		Title:          audioTitle,
		Filename:       manifest.AudioFile,
		InitialComment: header,
	})
	if err != nil {
		return fmt.Errorf("upload audio: %w", err)
	}

	if len(blocks) > 0 {
		if ts == "" {
			return fmt.Errorf("cannot post thread reply: ts is empty (file_id=%s); audio is already uploaded — re-running will 二重投稿", fileID)
		}
		replyParams := ReplyParams{
			Channel:  spec.Slack.Channel,
			ThreadTS: ts,
			Blocks:   blocks,
			Text:     fallback,
		}
		if err := poster.PostThreadReply(ctx, replyParams); err != nil {
			return fmt.Errorf("post thread reply: %w", err)
		}
	}

	_, _ = fmt.Fprintf(opts.Out, "channel: %s\n", spec.Slack.Channel)
	_, _ = fmt.Fprintf(opts.Out, "file_id: %s\n", fileID)
	_, _ = fmt.Fprintf(opts.Out, "thread_ts: %s\n", ts)
	return nil
}
