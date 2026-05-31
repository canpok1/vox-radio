package assemble

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/canpok1/vox-radio/internal/config"
	"github.com/canpok1/vox-radio/internal/model"
)

func TestBuildFFmpegArgs_NoClips_Error(t *testing.T) {
	ctx := BuildContext{
		Script:   model.Script{},
		Clips:    model.ClipsMeta{Clips: make([]model.ClipMeta, 0)},
		ClipsDir: "/clips",
		Assets:   config.AssetsConfig{},
		PauseSec: 0.5,
		OutPath:  "/out.mp3",
	}

	_, err := BuildFFmpegArgs(ctx)
	if err == nil {
		t.Error("expected error for no clips, got nil")
	}
}

func TestBuildFFmpegArgs_SingleClip(t *testing.T) {
	ctx := BuildContext{
		Script: model.Script{
			Segments: []model.ScriptSegment{
				{Type: model.SegmentTypeSpeech, SpeakerRole: "host", Text: "hello"},
			},
		},
		Clips: model.ClipsMeta{
			Clips: []model.ClipMeta{
				{Index: 0, File: "clip_000.wav", DurationSec: 2.0},
			},
		},
		ClipsDir: "/clips",
		Assets:   config.AssetsConfig{},
		PauseSec: 0.5,
		OutPath:  "/out.mp3",
	}

	args, err := BuildFFmpegArgs(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(args.Inputs) < 1 || args.Inputs[0] != filepath.Join("/clips", "clip_000.wav") {
		t.Errorf("clip input not found, inputs: %v", args.Inputs)
	}
	if !strings.Contains(args.FilterComplex, "[0:a]") {
		t.Errorf("filter_complex missing [0:a]: %s", args.FilterComplex)
	}
	if !strings.Contains(args.FilterComplex, "loudnorm") {
		t.Errorf("filter_complex missing loudnorm: %s", args.FilterComplex)
	}
	if args.OutputPath != "/out.mp3" {
		t.Errorf("output path: got %s, want /out.mp3", args.OutputPath)
	}
	foundCodec := false
	for _, a := range args.OutputArgs {
		if a == "libmp3lame" {
			foundCodec = true
		}
	}
	if !foundCodec {
		t.Errorf("libmp3lame not in output args: %v", args.OutputArgs)
	}
}

func TestBuildFFmpegArgs_MultipleClipsWithPauses(t *testing.T) {
	ctx := BuildContext{
		Script: model.Script{
			Segments: []model.ScriptSegment{
				{Type: model.SegmentTypeSpeech, SpeakerRole: "host", Text: "A"},
				{Type: model.SegmentTypeSpeech, SpeakerRole: "guest", Text: "B"},
				{Type: model.SegmentTypeSpeech, SpeakerRole: "host", Text: "C"},
			},
		},
		Clips: model.ClipsMeta{
			Clips: []model.ClipMeta{
				{Index: 0, File: "clip_000.wav", DurationSec: 1.0},
				{Index: 1, File: "clip_001.wav", DurationSec: 1.5},
				{Index: 2, File: "clip_002.wav", DurationSec: 2.0},
			},
		},
		ClipsDir: "/clips",
		Assets:   config.AssetsConfig{},
		PauseSec: 0.5,
		OutPath:  "/out.mp3",
	}

	args, err := BuildFFmpegArgs(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(args.Inputs) < 3 {
		t.Errorf("expected at least 3 inputs, got %d: %v", len(args.Inputs), args.Inputs)
	}
	if !strings.Contains(args.FilterComplex, "anullsrc") {
		t.Errorf("filter_complex missing anullsrc for pauses: %s", args.FilterComplex)
	}
	if !strings.Contains(args.FilterComplex, "concat") {
		t.Errorf("filter_complex missing concat: %s", args.FilterComplex)
	}
	// atrim の duration 指定は `duration=` でなければならない。
	// `d=` は atrim フィルタに存在しないオプションで ffmpeg が失敗する。
	if strings.Contains(args.FilterComplex, "atrim=d=") {
		t.Errorf("filter_complex uses invalid atrim option `d=` (must be `duration=`): %s", args.FilterComplex)
	}
	if !strings.Contains(args.FilterComplex, "atrim=duration=") {
		t.Errorf("filter_complex missing valid `atrim=duration=` for pauses: %s", args.FilterComplex)
	}
}

func TestBuildFFmpegArgs_SESegment(t *testing.T) {
	ctx := BuildContext{
		Script: model.Script{
			Segments: []model.ScriptSegment{
				{Type: model.SegmentTypeSpeech, SpeakerRole: "host", Text: "intro"},
				{Type: model.SegmentTypeSE, AssetName: "chime"},
				{Type: model.SegmentTypeSpeech, SpeakerRole: "host", Text: "main"},
			},
		},
		Clips: model.ClipsMeta{
			Clips: []model.ClipMeta{
				{Index: 0, File: "clip_000.wav", DurationSec: 2.0},
				{Index: 1, File: "clip_001.wav", DurationSec: 3.0},
			},
		},
		ClipsDir: "/clips",
		Assets: config.AssetsConfig{
			SE: map[string]config.SEEntry{
				"chime": {File: "/assets/chime.wav", Volume: 0.8},
			},
		},
		PauseSec: 0.5,
		OutPath:  "/out.mp3",
	}

	args, err := BuildFFmpegArgs(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	foundSE := false
	for _, inp := range args.Inputs {
		if inp == "/assets/chime.wav" {
			foundSE = true
		}
	}
	if !foundSE {
		t.Errorf("SE input /assets/chime.wav not found in inputs: %v", args.Inputs)
	}
	if !strings.Contains(args.FilterComplex, "adelay") {
		t.Errorf("filter_complex missing adelay for SE: %s", args.FilterComplex)
	}
	if !strings.Contains(args.FilterComplex, "amix") {
		t.Errorf("filter_complex missing amix for SE: %s", args.FilterComplex)
	}
}

func TestBuildFFmpegArgs_UnknownSEIgnored(t *testing.T) {
	ctx := BuildContext{
		Script: model.Script{
			Segments: []model.ScriptSegment{
				{Type: model.SegmentTypeSpeech, SpeakerRole: "host", Text: "hi"},
				{Type: model.SegmentTypeSE, AssetName: "unknown_se"},
			},
		},
		Clips: model.ClipsMeta{
			Clips: []model.ClipMeta{
				{Index: 0, File: "clip_000.wav", DurationSec: 1.0},
			},
		},
		ClipsDir: "/clips",
		Assets:   config.AssetsConfig{SE: map[string]config.SEEntry{}},
		PauseSec: 0.5,
		OutPath:  "/out.mp3",
	}

	_, err := BuildFFmpegArgs(ctx)
	if err != nil {
		t.Errorf("unexpected error for unknown SE: %v", err)
	}
}

func TestBuildFFmpegArgs_BGM(t *testing.T) {
	ctx := BuildContext{
		Script: model.Script{
			Segments: []model.ScriptSegment{
				{Type: model.SegmentTypeSpeech, SpeakerRole: "host", Text: "hello"},
			},
		},
		Clips: model.ClipsMeta{
			Clips: []model.ClipMeta{
				{Index: 0, File: "clip_000.wav", DurationSec: 2.0},
			},
		},
		ClipsDir: "/clips",
		Assets: config.AssetsConfig{
			BGM: map[string]config.BGMEntry{
				"main": {File: "/assets/bgm.mp3", Volume: 0.3, DuckRatio: 4.0, Loop: true},
			},
		},
		PauseSec: 0.5,
		OutPath:  "/out.mp3",
	}

	args, err := BuildFFmpegArgs(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	foundBGM := false
	for _, inp := range args.Inputs {
		if inp == "/assets/bgm.mp3" {
			foundBGM = true
		}
	}
	if !foundBGM {
		t.Errorf("BGM input /assets/bgm.mp3 not found in inputs: %v", args.Inputs)
	}
	if !strings.Contains(args.FilterComplex, "aloop") {
		t.Errorf("filter_complex missing aloop for BGM: %s", args.FilterComplex)
	}
	if !strings.Contains(args.FilterComplex, "sidechaincompress") {
		t.Errorf("filter_complex missing sidechaincompress for BGM: %s", args.FilterComplex)
	}
}

// TestBuildFFmpegArgs_OPJingle verifies that the opening jingle is prepended as a serial
// segment (concat) before the main content, not overlaid via amix.
func TestBuildFFmpegArgs_OPJingle(t *testing.T) {
	ctx := BuildContext{
		Script: model.Script{
			Segments: []model.ScriptSegment{
				{Type: model.SegmentTypeSpeech, SpeakerRole: "host", Text: "hello"},
			},
		},
		Clips: model.ClipsMeta{
			Clips: []model.ClipMeta{
				{Index: 0, File: "clip_000.wav", DurationSec: 2.0},
			},
		},
		ClipsDir: "/clips",
		Assets: config.AssetsConfig{
			Jingle: map[string]config.JingleEntry{
				"opening": {File: "/assets/opening.wav", FadeIn: 0.5},
			},
		},
		OpeningJingle: "opening",
		PauseSec:      0.5,
		OutPath:       "/out.mp3",
	}

	args, err := BuildFFmpegArgs(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	foundOP := false
	for _, inp := range args.Inputs {
		if inp == "/assets/opening.wav" {
			foundOP = true
		}
	}
	if !foundOP {
		t.Errorf("OP jingle input not found in inputs: %v", args.Inputs)
	}
	if !strings.Contains(args.FilterComplex, "afade") {
		t.Errorf("filter_complex missing afade for OP jingle: %s", args.FilterComplex)
	}
	// Jingle must be serial (concat), not overlaid (amix=duration=longest)
	if !strings.Contains(args.FilterComplex, "concat") {
		t.Errorf("filter_complex missing concat for serial jingle: %s", args.FilterComplex)
	}
}

// TestBuildFFmpegArgs_EDJingle verifies that the ending jingle is appended as a serial
// segment (concat) after the main content, not overlaid via amix.
func TestBuildFFmpegArgs_EDJingle(t *testing.T) {
	ctx := BuildContext{
		Script: model.Script{
			Segments: []model.ScriptSegment{
				{Type: model.SegmentTypeSpeech, SpeakerRole: "host", Text: "bye"},
			},
		},
		Clips: model.ClipsMeta{
			Clips: []model.ClipMeta{
				{Index: 0, File: "clip_000.wav", DurationSec: 2.0},
			},
		},
		ClipsDir: "/clips",
		Assets: config.AssetsConfig{
			Jingle: map[string]config.JingleEntry{
				"ending": {File: "/assets/ending.wav", FadeIn: 0.3, FadeOut: 1.5},
			},
		},
		EndingJingle: "ending",
		PauseSec:     0.5,
		OutPath:      "/out.mp3",
	}

	args, err := BuildFFmpegArgs(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	foundED := false
	for _, inp := range args.Inputs {
		if inp == "/assets/ending.wav" {
			foundED = true
		}
	}
	if !foundED {
		t.Errorf("ED jingle input not found in inputs: %v", args.Inputs)
	}
	if !strings.Contains(args.FilterComplex, "afade") {
		t.Errorf("filter_complex missing afade for ED jingle: %s", args.FilterComplex)
	}
	// Jingle must be serial (concat), not overlaid (amix=duration=longest)
	if !strings.Contains(args.FilterComplex, "concat") {
		t.Errorf("filter_complex missing concat for serial jingle: %s", args.FilterComplex)
	}
}

// TestBuildFFmpegArgs_BothJingles_WithPauses verifies that OP+ED jingles are both inserted
// with pause gaps between jingle and main content.
func TestBuildFFmpegArgs_BothJingles_WithPauses(t *testing.T) {
	ctx := BuildContext{
		Script: model.Script{
			Segments: []model.ScriptSegment{
				{Type: model.SegmentTypeSpeech, SpeakerRole: "host", Text: "hello"},
			},
		},
		Clips: model.ClipsMeta{
			Clips: []model.ClipMeta{
				{Index: 0, File: "clip_000.wav", DurationSec: 2.0},
			},
		},
		ClipsDir: "/clips",
		Assets: config.AssetsConfig{
			Jingle: map[string]config.JingleEntry{
				"opening": {File: "/assets/opening.wav", FadeIn: 0.5},
				"ending":  {File: "/assets/ending.wav", FadeOut: 1.0},
			},
		},
		OpeningJingle: "opening",
		EndingJingle:  "ending",
		PauseSec:      0.5,
		OutPath:       "/out.mp3",
	}

	args, err := BuildFFmpegArgs(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	foundOP, foundED := false, false
	for _, inp := range args.Inputs {
		if inp == "/assets/opening.wav" {
			foundOP = true
		}
		if inp == "/assets/ending.wav" {
			foundED = true
		}
	}
	if !foundOP {
		t.Errorf("OP jingle not in inputs: %v", args.Inputs)
	}
	if !foundED {
		t.Errorf("ED jingle not in inputs: %v", args.Inputs)
	}
	// Pauses between jingle and main content
	if !strings.Contains(args.FilterComplex, "anullsrc") {
		t.Errorf("filter_complex missing anullsrc (pause) for jingle gaps: %s", args.FilterComplex)
	}
	// Both jingles in serial concat
	if !strings.Contains(args.FilterComplex, "concat") {
		t.Errorf("filter_complex missing concat: %s", args.FilterComplex)
	}
}

// TestBuildFFmpegArgs_JingleKeyMissing_SilentSkip verifies that configuring OpeningJingle
// with a key that does not exist in Assets.Jingle does not cause an error — it is silently skipped.
func TestBuildFFmpegArgs_JingleKeyMissing_SilentSkip(t *testing.T) {
	ctx := BuildContext{
		Script: model.Script{
			Segments: []model.ScriptSegment{
				{Type: model.SegmentTypeSpeech, SpeakerRole: "host", Text: "hello"},
			},
		},
		Clips: model.ClipsMeta{
			Clips: []model.ClipMeta{
				{Index: 0, File: "clip_000.wav", DurationSec: 2.0},
			},
		},
		ClipsDir:      "/clips",
		Assets:        config.AssetsConfig{},
		OpeningJingle: "missing_key",
		PauseSec:      0.5,
		OutPath:       "/out.mp3",
	}

	args, err := BuildFFmpegArgs(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// No jingle file should appear in inputs
	for _, inp := range args.Inputs {
		if strings.Contains(inp, "jingle") {
			t.Errorf("unexpected jingle input when key is missing: %s", inp)
		}
	}
}

func TestComputeSEEvents_PositionsAfterSpeech(t *testing.T) {
	script := model.Script{
		Segments: []model.ScriptSegment{
			{Type: model.SegmentTypeSpeech, SpeakerRole: "host", Text: "first"},
			{Type: model.SegmentTypeSE, AssetName: "chime"},
			{Type: model.SegmentTypeSpeech, SpeakerRole: "host", Text: "second"},
		},
	}
	clips := []model.ClipMeta{
		{Index: 0, File: "clip_000.wav", DurationSec: 2.0},
		{Index: 1, File: "clip_001.wav", DurationSec: 3.0},
	}

	events := computeSEEvents(script, clips, 0.5)

	if len(events) != 1 {
		t.Fatalf("expected 1 SE event, got %d", len(events))
	}
	if events[0].seName != "chime" {
		t.Errorf("se name: got %s, want chime", events[0].seName)
	}
	// After clip_000 (2.0s) + pause (0.5s) = 2500ms
	wantMs := int((2.0 + 0.5) * 1000)
	if events[0].offsetMs != wantMs {
		t.Errorf("SE offset: got %d ms, want %d ms", events[0].offsetMs, wantMs)
	}
}

func TestBuildFFmpegArgs_DuplicateSEUsesDistinctInputs(t *testing.T) {
	ctx := BuildContext{
		Script: model.Script{
			Segments: []model.ScriptSegment{
				{Type: model.SegmentTypeSpeech, SpeakerRole: "host", Text: "intro"},
				{Type: model.SegmentTypeSE, AssetName: "chime"},
				{Type: model.SegmentTypeSpeech, SpeakerRole: "host", Text: "middle"},
				{Type: model.SegmentTypeSE, AssetName: "chime"},
				{Type: model.SegmentTypeSpeech, SpeakerRole: "host", Text: "end"},
			},
		},
		Clips: model.ClipsMeta{
			Clips: []model.ClipMeta{
				{Index: 0, File: "clip_000.wav", DurationSec: 1.0},
				{Index: 1, File: "clip_001.wav", DurationSec: 1.0},
				{Index: 2, File: "clip_002.wav", DurationSec: 1.0},
			},
		},
		ClipsDir: "/clips",
		Assets: config.AssetsConfig{
			SE: map[string]config.SEEntry{
				"chime": {File: "/assets/chime.wav", Volume: 0.8},
			},
		},
		PauseSec: 0.5,
		OutPath:  "/out.mp3",
	}

	args, err := BuildFFmpegArgs(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Count how many times chime.wav appears as input
	chimeCount := 0
	for _, inp := range args.Inputs {
		if inp == "/assets/chime.wav" {
			chimeCount++
		}
	}
	// Each SE usage must be a distinct input to avoid stream label reuse in filter_complex
	if chimeCount != 2 {
		t.Errorf("chime.wav input count: got %d, want 2 (one per SE usage)", chimeCount)
	}
}

func TestComputeSEEvents_NoSE(t *testing.T) {
	script := model.Script{
		Segments: []model.ScriptSegment{
			{Type: model.SegmentTypeSpeech, Text: "only speech"},
		},
	}
	clips := []model.ClipMeta{
		{Index: 0, File: "clip_000.wav", DurationSec: 1.0},
	}

	events := computeSEEvents(script, clips, 0.3)
	if len(events) != 0 {
		t.Errorf("expected 0 SE events, got %d", len(events))
	}
}

// TestBuildFFmpegArgs_LoudnormAfterMainContent verifies that loudnorm is applied AFTER
// the full main content (speech+BGM) is assembled, so it normalizes the whole output once.
func TestBuildFFmpegArgs_LoudnormAfterMainContent(t *testing.T) {
	ctx := BuildContext{
		Script: model.Script{
			Segments: []model.ScriptSegment{
				{Type: model.SegmentTypeSpeech, SpeakerRole: "host", Text: "hello"},
			},
		},
		Clips: model.ClipsMeta{
			Clips: []model.ClipMeta{
				{Index: 0, File: "clip_000.wav", DurationSec: 2.0},
			},
		},
		ClipsDir: "/clips",
		Assets: config.AssetsConfig{
			BGM: map[string]config.BGMEntry{
				"main": {File: "/assets/bgm.mp3", Volume: 0.3, DuckRatio: 4.0, Loop: true},
			},
		},
		PauseSec: 0.5,
		OutPath:  "/out.mp3",
	}

	args, err := BuildFFmpegArgs(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	filters := strings.Split(args.FilterComplex, ";")
	loudnormIdx := -1
	bgmFilterIdx := -1
	for i, f := range filters {
		if loudnormIdx == -1 && strings.Contains(f, "loudnorm") {
			loudnormIdx = i
		}
		if bgmFilterIdx == -1 && (strings.Contains(f, "aloop") || strings.Contains(f, "sidechaincompress")) {
			bgmFilterIdx = i
		}
		if loudnormIdx != -1 && bgmFilterIdx != -1 {
			break
		}
	}
	if loudnormIdx == -1 {
		t.Fatal("loudnorm not found in filter_complex")
	}
	if bgmFilterIdx == -1 {
		t.Fatal("BGM filter (aloop or sidechaincompress) not found in filter_complex")
	}
	if loudnormIdx <= bgmFilterIdx {
		t.Errorf("loudnorm (filter index %d) must appear AFTER BGM mix filter (index %d); filter_complex:\n%s",
			loudnormIdx, bgmFilterIdx, args.FilterComplex)
	}
}

// TestBuildFFmpegArgs_FinalOutputIsAlimiter verifies that the final [out] stage uses
// alimiter (peak protection), not a dynamic loudnorm that would cause BGM pumping.
func TestBuildFFmpegArgs_FinalOutputIsAlimiter(t *testing.T) {
	for _, name := range []string{"no BGM", "with BGM"} {
		t.Run(name, func(t *testing.T) {
			assets := config.AssetsConfig{}
			if name == "with BGM" {
				assets.BGM = map[string]config.BGMEntry{
					"main": {File: "/assets/bgm.mp3", Volume: 0.3, Loop: true},
				}
			}
			ctx := BuildContext{
				Script: model.Script{
					Segments: []model.ScriptSegment{
						{Type: model.SegmentTypeSpeech, SpeakerRole: "host", Text: "hello"},
					},
				},
				Clips: model.ClipsMeta{
					Clips: []model.ClipMeta{
						{Index: 0, File: "clip_000.wav", DurationSec: 2.0},
					},
				},
				ClipsDir: "/clips",
				Assets:   assets,
				PauseSec: 0.5,
				OutPath:  "/out.mp3",
			}

			args, err := BuildFFmpegArgs(ctx)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			filters := strings.Split(args.FilterComplex, ";")
			for _, f := range filters {
				if strings.Contains(f, "[out]") {
					if strings.Contains(f, "loudnorm") {
						t.Errorf("final [out] must not use loudnorm (causes BGM pumping): %s", f)
					}
					if !strings.Contains(f, "alimiter") {
						t.Errorf("final [out] must use alimiter for peak protection: %s", f)
					}
					return
				}
			}
			t.Error("[out] label not found in filter_complex")
		})
	}
}
