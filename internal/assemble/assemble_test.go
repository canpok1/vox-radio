package assemble

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/canpok1/vox-radio/internal/config"
	"github.com/canpok1/vox-radio/internal/model"
)

func newTestAssembler(ffmpegErr error, duration float64, size int64) *Assembler {
	return &Assembler{
		AssetsConfig: config.AssetsConfig{},
		Program:      config.ProgramConfig{SegmentPauseSec: 0.5},
		runFFmpeg:    func(_ context.Context, _ []string, _ io.Writer) error { return ffmpegErr },
		getDuration:  func(_ string) (float64, error) { return duration, nil },
		getFileSize:  func(_ string) (int64, error) { return size, nil },
		logger:       slog.Default(),
	}
}

func TestAssembler_Run_ReturnsResult(t *testing.T) {
	var capturedArgs []string
	a := &Assembler{
		AssetsConfig: config.AssetsConfig{},
		Program:      config.ProgramConfig{SegmentPauseSec: 0.5},
		runFFmpeg: func(_ context.Context, args []string, _ io.Writer) error {
			capturedArgs = args
			return nil
		},
		getDuration: func(_ string) (float64, error) { return 60.0, nil },
		getFileSize: func(_ string) (int64, error) { return 1024, nil },
		logger:      slog.Default(),
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
		Program:      config.ProgramConfig{SegmentPauseSec: 0}, // zero → default
		runFFmpeg: func(_ context.Context, args []string, _ io.Writer) error {
			capturedArgs = args
			return nil
		},
		getDuration: func(_ string) (float64, error) { return 1.0, nil },
		getFileSize: func(_ string) (int64, error) { return 100, nil },
		logger:      slog.Default(),
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
	if len(capturedArgs) == 0 {
		t.Error("ffmpeg was not called")
	}
}

func TestAssembler_Run_LogsStartAndComplete(t *testing.T) {
	a := newTestAssembler(nil, 30.0, 1024*1024)

	var buf strings.Builder
	logger := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelInfo}))
	a.logger = logger

	script := model.Script{
		Segments: []model.ScriptSegment{
			{Type: model.SegmentTypeSpeech, Text: "テスト"},
		},
	}
	clips := model.ClipsMeta{
		Clips: []model.ClipMeta{
			{Index: 0, File: "clip_000.wav", DurationSec: 2.0},
		},
	}

	dir := t.TempDir()
	if _, err := a.Run(context.Background(), script, clips, dir, filepath.Join(dir, "out.mp3")); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	logs := buf.String()
	if !strings.Contains(logs, "開始") {
		t.Errorf("should log start: %q", logs)
	}
	if !strings.Contains(logs, "完了") {
		t.Errorf("should log complete: %q", logs)
	}
}

func TestAssembler_Run_FFmpegOutputGoesToWriter(t *testing.T) {
	var ffmpegOutput strings.Builder
	a := &Assembler{
		AssetsConfig: config.AssetsConfig{},
		Program:      config.ProgramConfig{SegmentPauseSec: 0.5},
		runFFmpeg: func(_ context.Context, _ []string, w io.Writer) error {
			if w != nil {
				_, _ = io.WriteString(w, "ffmpeg output here")
			}
			return nil
		},
		getDuration:  func(_ string) (float64, error) { return 30.0, nil },
		getFileSize:  func(_ string) (int64, error) { return 512, nil },
		ffmpegWriter: &ffmpegOutput,
		logger:       slog.Default(),
	}

	script := model.Script{
		Segments: []model.ScriptSegment{
			{Type: model.SegmentTypeSpeech, Text: "テスト"},
		},
	}
	clips := model.ClipsMeta{
		Clips: []model.ClipMeta{
			{Index: 0, File: "clip_000.wav", DurationSec: 2.0},
		},
	}

	dir := t.TempDir()
	if _, err := a.Run(context.Background(), script, clips, dir, filepath.Join(dir, "out.mp3")); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(ffmpegOutput.String(), "ffmpeg output here") {
		t.Errorf("ffmpeg output should go to writer: %q", ffmpegOutput.String())
	}
}

// TestAssembler_Run_RealYAMLKeys verifies that asset keys from the actual YAML profile
// are correctly wired through to the ffmpeg command, preventing silent disable recurrence.
// This is the integration test that catches the bug described in Issue #98
// where YAML keys ("opening"/"ending") didn't match the code's expected keys ("op"/"ed").
func TestAssembler_Run_RealYAMLKeys(t *testing.T) {
	// Load the testdata profile to get real YAML asset keys.
	profile, err := config.LoadProfile("../../internal/config/testdata/profile.yaml")
	if err != nil {
		t.Fatalf("load profile: %v", err)
	}

	var capturedArgs []string
	a := &Assembler{
		AssetsConfig: profile.Assets,
		Program:      profile.Program,
		runFFmpeg: func(_ context.Context, args []string, _ io.Writer) error {
			capturedArgs = args
			return nil
		},
		getDuration: func(_ string) (float64, error) { return 30.0, nil },
		getFileSize: func(_ string) (int64, error) { return 1024, nil },
		logger:      slog.Default(),
	}

	// Build a script that uses the exact YAML keys for jingle and bgm.
	openingKey := profile.Program.OpeningJingle
	endingKey := profile.Program.EndingJingle
	if openingKey == "" || endingKey == "" {
		t.Skip("profile has no opening/ending jingle configured")
	}

	script := model.Script{
		Segments: []model.ScriptSegment{
			{Type: model.SegmentTypeSpeech, SpeakerRole: "host", Text: "テスト"},
		},
	}
	clips := model.ClipsMeta{
		Clips: []model.ClipMeta{
			{Index: 0, File: "clip_000.wav", DurationSec: 2.0},
		},
	}

	dir := t.TempDir()
	_, err = a.Run(context.Background(), script, clips, dir, filepath.Join(dir, "out.mp3"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify that the opening jingle file appears in the ffmpeg arguments.
	// If the YAML key doesn't match what the code looks up, the jingle would be silently skipped.
	if openingEntry, ok := profile.Assets.Jingle[openingKey]; ok {
		foundOpening := false
		for _, arg := range capturedArgs {
			if arg == openingEntry.File {
				foundOpening = true
			}
		}
		if !foundOpening {
			t.Errorf("opening jingle file %q (key=%q) not found in ffmpeg args: %v",
				openingEntry.File, openingKey, capturedArgs)
		}
	}

	if endingEntry, ok := profile.Assets.Jingle[endingKey]; ok {
		foundEnding := false
		for _, arg := range capturedArgs {
			if arg == endingEntry.File {
				foundEnding = true
			}
		}
		if !foundEnding {
			t.Errorf("ending jingle file %q (key=%q) not found in ffmpeg args: %v",
				endingEntry.File, endingKey, capturedArgs)
		}
	}
}
