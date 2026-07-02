package slack

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"os"

	"github.com/canpok1/vox-radio/internal/model"
)

// 投稿処理で必要になる Slack Bot スコープ。
const (
	scopeFilesWrite = "files:write" // mp3 アップロード
	scopeFilesRead  = "files:read"  // アップロード完了確認・親メッセージ ts 取得（files.info）
	scopeChatWrite  = "chat:write"  // スレッド返信
)

// Options holds the inputs for Run.
type Options struct {
	Manifest  model.Manifest
	AudioPath string
	Spec      SlackSpec
	Token     string
	Channel   string
	APIURL    string
	StatePath string
	DryRun    bool
	Out       io.Writer
	Logger    *slog.Logger
}

// Run executes the slackpost workflow.
// poster is injected for testing; pass nil to use the real Poster (requires valid token).
func Run(opts Options, poster Poster) error {
	if opts.Out == nil {
		opts.Out = os.Stdout
	}

	logger := opts.Logger
	if logger == nil {
		logger = slog.New(slog.NewTextHandler(io.Discard, nil))
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
	channel := opts.Channel
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
			logger.Info("投稿は完了済みのためスキップします", "state", statePath)
			writeResult(opts.Out, loaded.Channel, loaded.FileID, loaded.ThreadTS)
			return nil
		}
		if loaded.FileID != "" {
			logger.Info("アップロード済みの状態から再開します", "file_id", loaded.FileID)
			state = *loaded
			needUpload = false
		}
	}

	// 副作用（アップロード）の前に、これから実際に呼び出す API に必要なスコープだけを
	// 検証し、権限不足を早期・明確に通知する。再開時は残りの工程に不要なスコープで弾かない。
	var required []string
	if needUpload {
		required = append(required, scopeFilesWrite)
	}
	if len(blocks) > 0 {
		required = append(required, scopeChatWrite)
		if state.ThreadTS == "" {
			required = append(required, scopeFilesRead)
		}
	}
	if err := poster.VerifyScopes(ctx, required); err != nil {
		logger.Error("Slack スコープ検証に失敗しました", "required", required, "err", err)
		return fmt.Errorf("スコープ検証: %w", err)
	}
	logger.Info("Slack スコープ検証に成功しました", "required", required)

	var runErr error
	if needUpload {
		logger.Info("mp3 をアップロードします", "channel", channel, "audio", opts.AudioPath)
		state.FileID, runErr = poster.UploadAudio(ctx, UploadParams{
			Channel:        channel,
			FilePath:       opts.AudioPath,
			Title:          audioTitle,
			Filename:       opts.Manifest.AudioFile,
			InitialComment: header,
		})
		if runErr != nil {
			logger.Error("mp3 のアップロードに失敗しました", "err", runErr)
			return fmt.Errorf("upload audio: %w", runErr)
		}
		logger.Info("mp3 のアップロードに成功しました", "file_id", state.FileID)
		_ = saveState(statePath, state)
	}

	if len(blocks) > 0 && state.ThreadTS == "" {
		logger.Info("スレッド ts を解決します", "file_id", state.FileID)
		state.ThreadTS, runErr = poster.ResolveThreadTS(ctx, state.FileID, channel)
		if runErr != nil {
			logger.Error("スレッド ts の解決に失敗しました", "file_id", state.FileID, "err", runErr)
			return runErr
		}
		_ = saveState(statePath, state)
	}

	if len(blocks) > 0 {
		logger.Info("スレッド返信を投稿します", "thread_ts", state.ThreadTS)
		if err := poster.PostThreadReply(ctx, ReplyParams{
			Channel:  channel,
			ThreadTS: state.ThreadTS,
			Blocks:   blocks,
			Text:     fallback,
		}); err != nil {
			logger.Error("スレッド返信の投稿に失敗しました", "err", err)
			return fmt.Errorf("post thread reply: %w", err)
		}
	}

	state.Replied = true
	_ = saveState(statePath, state)

	logger.Info("Slack への投稿が完了しました", "channel", channel, "file_id", state.FileID, "thread_ts", state.ThreadTS)
	writeResult(opts.Out, channel, state.FileID, state.ThreadTS)
	return nil
}

func writeResult(w io.Writer, channel, fileID, ts string) {
	_, _ = fmt.Fprintf(w, "channel: %s\n", channel)
	_, _ = fmt.Fprintf(w, "file_id: %s\n", fileID)
	_, _ = fmt.Fprintf(w, "thread_ts: %s\n", ts)
}
