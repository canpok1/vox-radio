package slack

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/canpok1/vox-radio/internal/model"
)

// Options holds the inputs for Run.
type Options struct {
	Manifest  model.Manifest
	AudioPath string
	Spec      SlackSpec
	Token     string
	APIURL    string
	StatePath string
	DryRun    bool
	Out       io.Writer
}

// Run executes the slackpost workflow.
// poster is injected for testing; pass nil to use the real Poster (requires valid token).
func Run(opts Options, poster Poster) error {
	if opts.Out == nil {
		opts.Out = os.Stdout
	}

	templates, err := opts.Spec.Slack.LoadTemplates(opts.Spec.BaseDir)
	if err != nil {
		return fmt.Errorf("load templates: %w", err)
	}

	header, err := BuildParent(opts.Manifest, templates.Parent)
	if err != nil {
		return fmt.Errorf("render parent template: %w", err)
	}

	threadText, err := BuildThread(opts.Manifest, templates.Thread)
	if err != nil {
		return fmt.Errorf("render thread template: %w", err)
	}
	blocks := SplitIntoSectionBlocks(threadText)

	fallback, err := BuildFallback(opts.Manifest, templates.Fallback)
	if err != nil {
		return fmt.Errorf("render fallback template: %w", err)
	}

	audioTitle := BuildAudioTitle(opts.Manifest)

	if opts.DryRun {
		_, _ = fmt.Fprintf(opts.Out, "audio: %s\n", opts.AudioPath)
		_, _ = fmt.Fprintf(opts.Out, "header: %s\n", header)
		if len(blocks) > 0 {
			blocksJSON, _ := json.MarshalIndent(blocks, "", "  ")
			_, _ = fmt.Fprintf(opts.Out, "thread blocks:\n%s\n", blocksJSON)
		}
		return nil
	}

	if _, err := os.Stat(opts.AudioPath); err != nil {
		return fmt.Errorf("audio file not found: %w", err)
	}

	if poster == nil {
		poster = NewPoster(opts.Token, opts.APIURL)
	}

	ctx := context.Background()
	channel := opts.Spec.Slack.Channel
	statePath := opts.StatePath

	// Build a base state from the current manifest; update fields progressively as each
	// phase completes and write a checkpoint so re-runs can resume from where they left off.
	state := PostState{
		AudioFile:     opts.Manifest.AudioFile,
		EpisodeNumber: opts.Manifest.EpisodeNumber,
		Channel:       channel,
	}
	needUpload := true

	if loaded, err := loadState(statePath); err == nil && loaded.Matches(opts.Manifest.AudioFile, opts.Manifest.EpisodeNumber) {
		if loaded.Replied {
			writeResult(opts.Out, loaded.Channel, loaded.FileID, loaded.ThreadTS)
			return nil
		}
		if loaded.FileID != "" {
			state = *loaded
			needUpload = false
		}
	}

	var runErr error
	if needUpload {
		state.FileID, runErr = poster.UploadAudio(ctx, UploadParams{
			Channel:        channel,
			FilePath:       opts.AudioPath,
			Title:          audioTitle,
			Filename:       opts.Manifest.AudioFile,
			InitialComment: header,
		})
		if runErr != nil {
			return fmt.Errorf("upload audio: %w", runErr)
		}
		_ = saveState(statePath, state)
	}

	if len(blocks) > 0 && state.ThreadTS == "" {
		state.ThreadTS, runErr = poster.ResolveThreadTS(ctx, state.FileID, channel)
		if runErr != nil {
			return runErr
		}
		_ = saveState(statePath, state)
	}

	if len(blocks) > 0 {
		if err := poster.PostThreadReply(ctx, ReplyParams{
			Channel:  channel,
			ThreadTS: state.ThreadTS,
			Blocks:   blocks,
			Text:     fallback,
		}); err != nil {
			return fmt.Errorf("post thread reply: %w", err)
		}
	}

	state.Replied = true
	_ = saveState(statePath, state)

	writeResult(opts.Out, channel, state.FileID, state.ThreadTS)
	return nil
}

func writeResult(w io.Writer, channel, fileID, ts string) {
	_, _ = fmt.Fprintf(w, "channel: %s\n", channel)
	_, _ = fmt.Fprintf(w, "file_id: %s\n", fileID)
	_, _ = fmt.Fprintf(w, "thread_ts: %s\n", ts)
}
