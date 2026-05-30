package assemble

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/canpok1/vox-radio/internal/config"
	"github.com/canpok1/vox-radio/internal/model"
)

func newTestAssembler(ffmpegErr error, duration float64, size int64) *Assembler {
	return &Assembler{
		AssetsConfig: config.AssetsConfig{},
		ShowConfig:   model.ShowConfig{SegmentPauseSec: 0.5},
		runFFmpeg:    func(_ context.Context, _ []string) error { return ffmpegErr },
		getDuration:  func(_ string) (float64, error) { return duration, nil },
		getFileSize:  func(_ string) (int64, error) { return size, nil },
	}
}

func TestAssembler_Run_ReturnsResult(t *testing.T) {
	var capturedArgs []string
	a := &Assembler{
		AssetsConfig: config.AssetsConfig{},
		ShowConfig:   model.ShowConfig{SegmentPauseSec: 0.5},
		runFFmpeg: func(_ context.Context, args []string) error {
			capturedArgs = args
			return nil
		},
		getDuration: func(_ string) (float64, error) { return 60.0, nil },
		getFileSize: func(_ string) (int64, error) { return 1024, nil },
	}

	script := model.Script{
		Segments: []model.ScriptSegment{
			{Type: model.SegmentTypeSpeech, Text: "hello"},
		},
	}
	clips := model.ClipsMeta{
		Clips: []model.ClipMeta{
			{Index: 0, File: "clip_000.wav", DurationSec: 2.0},
		},
	}

	dir := t.TempDir()
	outPath := filepath.Join(dir, "out.mp3")
	result, err := a.Run(context.Background(), script, clips, dir, outPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.DurationSec != 60.0 {
		t.Errorf("duration: got %.1f, want 60.0", result.DurationSec)
	}
	if result.Bytes != 1024 {
		t.Errorf("bytes: got %d, want 1024", result.Bytes)
	}

	if len(capturedArgs) == 0 {
		t.Error("ffmpeg was not called")
	}
	foundOut := false
	for _, arg := range capturedArgs {
		if arg == outPath {
			foundOut = true
		}
	}
	if !foundOut {
		t.Errorf("output path not found in ffmpeg args: %v", capturedArgs)
	}
}

func TestAssembler_Run_FFmpegError(t *testing.T) {
	a := newTestAssembler(errors.New("ffmpeg failed"), 0, 0)

	script := model.Script{
		Segments: []model.ScriptSegment{
			{Type: model.SegmentTypeSpeech, Text: "hello"},
		},
	}
	clips := model.ClipsMeta{
		Clips: []model.ClipMeta{
			{Index: 0, File: "clip_000.wav", DurationSec: 2.0},
		},
	}

	dir := t.TempDir()
	_, err := a.Run(context.Background(), script, clips, dir, filepath.Join(dir, "out.mp3"))
	if err == nil {
		t.Error("expected error, got nil")
	}
}

func TestAssembler_Run_NoClips_Error(t *testing.T) {
	a := newTestAssembler(nil, 0, 0)

	script := model.Script{}
	clips := model.ClipsMeta{Clips: make([]model.ClipMeta, 0)}

	dir := t.TempDir()
	_, err := a.Run(context.Background(), script, clips, dir, filepath.Join(dir, "out.mp3"))
	if err == nil {
		t.Error("expected error for no clips, got nil")
	}
}

func TestAssembler_Run_CreatesOutputDir(t *testing.T) {
	a := newTestAssembler(nil, 30.0, 512)

	script := model.Script{
		Segments: []model.ScriptSegment{
			{Type: model.SegmentTypeSpeech, Text: "hello"},
		},
	}
	clips := model.ClipsMeta{
		Clips: []model.ClipMeta{
			{Index: 0, File: "clip_000.wav", DurationSec: 2.0},
		},
	}

	outDir := filepath.Join(t.TempDir(), "nested", "output")
	outPath := filepath.Join(outDir, "episode.mp3")
	_, err := a.Run(context.Background(), script, clips, t.TempDir(), outPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if _, statErr := os.Stat(outDir); os.IsNotExist(statErr) {
		t.Errorf("output dir not created: %s", outDir)
	}
}

func TestAssembler_Run_DefaultPause(t *testing.T) {
	var capturedArgs []string
	a := &Assembler{
		AssetsConfig: config.AssetsConfig{},
		ShowConfig:   model.ShowConfig{SegmentPauseSec: 0}, // zero → default
		runFFmpeg: func(_ context.Context, args []string) error {
			capturedArgs = args
			return nil
		},
		getDuration: func(_ string) (float64, error) { return 1.0, nil },
		getFileSize: func(_ string) (int64, error) { return 100, nil },
	}

	script := model.Script{
		Segments: []model.ScriptSegment{
			{Type: model.SegmentTypeSpeech, Text: "A"},
			{Type: model.SegmentTypeSpeech, Text: "B"},
		},
	}
	clips := model.ClipsMeta{
		Clips: []model.ClipMeta{
			{Index: 0, File: "clip_000.wav", DurationSec: 1.0},
			{Index: 1, File: "clip_001.wav", DurationSec: 1.0},
		},
	}

	dir := t.TempDir()
	_, err := a.Run(context.Background(), script, clips, dir, filepath.Join(dir, "out.mp3"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Ensure ffmpeg was called (default pause used)
	if len(capturedArgs) == 0 {
		t.Error("ffmpeg was not called")
	}
}
