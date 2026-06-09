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
	StatePath    string // optional: override state file path (default: derived from ManifestPath)
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
	channel := spec.Slack.Channel

	statePath := opts.StatePath
	if statePath == "" {
		statePath = DefaultStatePath(opts.ManifestPath)
	}

	// Build a base state from the current manifest; update fields progressively as each
	// phase completes and write a checkpoint so re-runs can resume from where they left off.
	state := PostState{
		AudioFile:     manifest.AudioFile,
		EpisodeNumber: manifest.EpisodeNumber,
		Channel:       channel,
	}
	needUpload := true

	if loaded, err := loadState(statePath); err == nil && loaded.AudioFile == manifest.AudioFile && loaded.EpisodeNumber == manifest.EpisodeNumber {
		if loaded.Replied {
			writeResult(opts.Out, loaded.Channel, loaded.FileID, loaded.ThreadTS)
			return nil
		}
		if loaded.FileID != "" {
			state = *loaded
			needUpload = false
		}
	}

	if needUpload {
		state.FileID, err = poster.UploadAudio(ctx, UploadParams{
			Channel:        channel,
			FilePath:       audioPath,
			Title:          audioTitle,
			Filename:       manifest.AudioFile,
			InitialComment: header,
		})
		if err != nil {
			return fmt.Errorf("upload audio: %w", err)
		}
		_ = saveState(statePath, state)
	}

	if len(blocks) > 0 && state.ThreadTS == "" {
		state.ThreadTS, err = poster.ResolveThreadTS(ctx, state.FileID, channel)
		if err != nil {
			return err
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
