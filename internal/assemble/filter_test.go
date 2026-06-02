package assemble

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/canpok1/vox-radio/internal/config"
	"github.com/canpok1/vox-radio/internal/model"
)

// hasStreamLoop reports whether the input with the given path has -stream_loop -1 in its PreOptions.
func hasStreamLoop(inputs []FFmpegInput, path string) bool {
	for _, inp := range inputs {
		if inp.Path != path {
			continue
		}
		for i, opt := range inp.PreOptions {
			if opt == "-stream_loop" && i+1 < len(inp.PreOptions) && inp.PreOptions[i+1] == "-1" {
				return true
			}
		}
	}
	return false
}

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

	if len(args.Inputs) < 1 || args.Inputs[0].Path != filepath.Join("/clips", "clip_000.wav") {
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
		if inp.Path == "/assets/chime.wav" {
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

// TestBuildFFmpegArgs_BGMSegment_StartsAndStops verifies that a bgm segment with asset_name
// starts BGM for the run, and a bgm segment with empty asset_name stops it.
func TestBuildFFmpegArgs_BGMSegment_StartsAndStops(t *testing.T) {
	ctx := BuildContext{
		Script: model.Script{
			Segments: []model.ScriptSegment{
				{Type: model.SegmentTypeSpeech, SpeakerRole: "host", Text: "start"},
				{Type: model.SegmentTypeBGM, AssetName: "talk_bgm"},
				{Type: model.SegmentTypeSpeech, SpeakerRole: "host", Text: "with bgm"},
				{Type: model.SegmentTypeBGM, AssetName: ""},
				{Type: model.SegmentTypeSpeech, SpeakerRole: "host", Text: "no bgm"},
			},
		},
		Clips: model.ClipsMeta{
			Clips: []model.ClipMeta{
				{Index: 0, File: "clip_000.wav", DurationSec: 1.0},
				{Index: 1, File: "clip_001.wav", DurationSec: 2.0},
				{Index: 2, File: "clip_002.wav", DurationSec: 1.5},
			},
		},
		ClipsDir: "/clips",
		Assets: config.AssetsConfig{
			BGM: map[string]config.BGMEntry{
				"talk_bgm": {File: "/assets/bgm.mp3", Volume: 0.3, DuckRatio: 4.0, Loop: true},
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
		if inp.Path == "/assets/bgm.mp3" {
			foundBGM = true
		}
	}
	if !foundBGM {
		t.Errorf("BGM input /assets/bgm.mp3 not found in inputs: %v", args.Inputs)
	}
	if !hasStreamLoop(args.Inputs, "/assets/bgm.mp3") {
		t.Errorf("loop:true BGM should have -stream_loop -1 in PreOptions, inputs: %v", args.Inputs)
	}
	if strings.Contains(args.FilterComplex, "aloop") {
		t.Errorf("loop:true BGM must not use aloop filter, filter: %s", args.FilterComplex)
	}
	if !strings.Contains(args.FilterComplex, "sidechaincompress") {
		t.Errorf("filter_complex missing sidechaincompress for BGM ducking: %s", args.FilterComplex)
	}
}

// TestBuildFFmpegArgs_BGMDoesNotCrossJingle verifies that BGM stops at a jingle boundary
// and does not carry over to the next run.
func TestBuildFFmpegArgs_BGMDoesNotCrossJingle(t *testing.T) {
	ctx := BuildContext{
		Script: model.Script{
			Segments: []model.ScriptSegment{
				{Type: model.SegmentTypeBGM, AssetName: "talk_bgm"},
				{Type: model.SegmentTypeSpeech, SpeakerRole: "host", Text: "with bgm"},
				{Type: model.SegmentTypeJingle, AssetName: "eyecatch"},
				{Type: model.SegmentTypeSpeech, SpeakerRole: "host", Text: "no bgm"},
			},
		},
		Clips: model.ClipsMeta{
			Clips: []model.ClipMeta{
				{Index: 0, File: "clip_000.wav", DurationSec: 2.0},
				{Index: 1, File: "clip_001.wav", DurationSec: 2.0},
			},
		},
		ClipsDir: "/clips",
		Assets: config.AssetsConfig{
			BGM: map[string]config.BGMEntry{
				"talk_bgm": {File: "/assets/bgm.mp3", Volume: 0.3, DuckRatio: 0, Loop: true},
			},
			Jingle: map[string]config.JingleEntry{
				"eyecatch": {File: "/assets/eyecatch.wav"},
			},
		},
		PauseSec: 0.5,
		OutPath:  "/out.mp3",
	}

	args, err := BuildFFmpegArgs(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// BGM should appear (active in run 0)
	foundBGM := false
	for _, inp := range args.Inputs {
		if inp.Path == "/assets/bgm.mp3" {
			foundBGM = true
		}
	}
	if !foundBGM {
		t.Errorf("BGM input not found in inputs: %v", args.Inputs)
	}
	// Jingle must be in concat (serial)
	if !strings.Contains(args.FilterComplex, "concat") {
		t.Errorf("filter_complex missing concat for jingle: %s", args.FilterComplex)
	}
}

// TestBuildFFmpegArgs_JingleSegment_SerialConcat verifies that a jingle segment in the script
// is placed serially (concat), not overlaid (amix).
func TestBuildFFmpegArgs_JingleSegment_SerialConcat(t *testing.T) {
	ctx := BuildContext{
		Script: model.Script{
			Segments: []model.ScriptSegment{
				{Type: model.SegmentTypeSpeech, SpeakerRole: "host", Text: "before"},
				{Type: model.SegmentTypeJingle, AssetName: "eyecatch"},
				{Type: model.SegmentTypeSpeech, SpeakerRole: "host", Text: "after"},
			},
		},
		Clips: model.ClipsMeta{
			Clips: []model.ClipMeta{
				{Index: 0, File: "clip_000.wav", DurationSec: 2.0},
				{Index: 1, File: "clip_001.wav", DurationSec: 2.0},
			},
		},
		ClipsDir: "/clips",
		Assets: config.AssetsConfig{
			Jingle: map[string]config.JingleEntry{
				"eyecatch": {File: "/assets/eyecatch.wav", FadeIn: 0.3},
			},
		},
		PauseSec: 0.5,
		OutPath:  "/out.mp3",
	}

	args, err := BuildFFmpegArgs(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	foundJingle := false
	for _, inp := range args.Inputs {
		if inp.Path == "/assets/eyecatch.wav" {
			foundJingle = true
		}
	}
	if !foundJingle {
		t.Errorf("jingle input not found in inputs: %v", args.Inputs)
	}
	// Jingle is serial: must use concat
	if !strings.Contains(args.FilterComplex, "concat") {
		t.Errorf("filter_complex missing concat for serial jingle: %s", args.FilterComplex)
	}
	// Pause between run and jingle
	if !strings.Contains(args.FilterComplex, "anullsrc") {
		t.Errorf("filter_complex missing anullsrc for pause: %s", args.FilterComplex)
	}
	// Fade effect
	if !strings.Contains(args.FilterComplex, "afade") {
		t.Errorf("filter_complex missing afade for jingle fade: %s", args.FilterComplex)
	}
}

// TestBuildFFmpegArgs_JingleAtBeginning verifies that a jingle at the start (OP) is placed
// before the main content in serial concat.
func TestBuildFFmpegArgs_JingleAtBeginning(t *testing.T) {
	ctx := BuildContext{
		Script: model.Script{
			Segments: []model.ScriptSegment{
				{Type: model.SegmentTypeJingle, AssetName: "opening"},
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
		PauseSec: 0.5,
		OutPath:  "/out.mp3",
	}

	args, err := BuildFFmpegArgs(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	foundOP := false
	for _, inp := range args.Inputs {
		if inp.Path == "/assets/opening.wav" {
			foundOP = true
		}
	}
	if !foundOP {
		t.Errorf("opening jingle not in inputs: %v", args.Inputs)
	}
	if !strings.Contains(args.FilterComplex, "concat") {
		t.Errorf("filter_complex missing concat: %s", args.FilterComplex)
	}
}

// TestBuildFFmpegArgs_JingleAtEnd verifies that a jingle at the end (ED) is placed
// after the main content in serial concat.
func TestBuildFFmpegArgs_JingleAtEnd(t *testing.T) {
	ctx := BuildContext{
		Script: model.Script{
			Segments: []model.ScriptSegment{
				{Type: model.SegmentTypeSpeech, SpeakerRole: "host", Text: "bye"},
				{Type: model.SegmentTypeJingle, AssetName: "ending"},
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
		PauseSec: 0.5,
		OutPath:  "/out.mp3",
	}

	args, err := BuildFFmpegArgs(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	foundED := false
	for _, inp := range args.Inputs {
		if inp.Path == "/assets/ending.wav" {
			foundED = true
		}
	}
	if !foundED {
		t.Errorf("ending jingle not in inputs: %v", args.Inputs)
	}
	if !strings.Contains(args.FilterComplex, "afade") {
		t.Errorf("filter_complex missing afade for ED jingle: %s", args.FilterComplex)
	}
	if !strings.Contains(args.FilterComplex, "concat") {
		t.Errorf("filter_complex missing concat: %s", args.FilterComplex)
	}
}

// TestBuildFFmpegArgs_OPAndED verifies that OP and ED jingles from the script
// are both placed as serial segments with pauses.
func TestBuildFFmpegArgs_OPAndED(t *testing.T) {
	ctx := BuildContext{
		Script: model.Script{
			Segments: []model.ScriptSegment{
				{Type: model.SegmentTypeJingle, AssetName: "opening"},
				{Type: model.SegmentTypeSpeech, SpeakerRole: "host", Text: "hello"},
				{Type: model.SegmentTypeJingle, AssetName: "ending"},
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
		PauseSec: 0.5,
		OutPath:  "/out.mp3",
	}

	args, err := BuildFFmpegArgs(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	foundOP, foundED := false, false
	for _, inp := range args.Inputs {
		if inp.Path == "/assets/opening.wav" {
			foundOP = true
		}
		if inp.Path == "/assets/ending.wav" {
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
	if !strings.Contains(args.FilterComplex, "concat") {
		t.Errorf("filter_complex missing concat: %s", args.FilterComplex)
	}
}

// TestBuildFFmpegArgs_ConsecutiveJingles verifies that consecutive jingles work correctly.
func TestBuildFFmpegArgs_ConsecutiveJingles(t *testing.T) {
	ctx := BuildContext{
		Script: model.Script{
			Segments: []model.ScriptSegment{
				{Type: model.SegmentTypeSpeech, SpeakerRole: "host", Text: "A"},
				{Type: model.SegmentTypeJingle, AssetName: "j1"},
				{Type: model.SegmentTypeJingle, AssetName: "j2"},
				{Type: model.SegmentTypeSpeech, SpeakerRole: "host", Text: "B"},
			},
		},
		Clips: model.ClipsMeta{
			Clips: []model.ClipMeta{
				{Index: 0, File: "clip_000.wav", DurationSec: 1.0},
				{Index: 1, File: "clip_001.wav", DurationSec: 1.0},
			},
		},
		ClipsDir: "/clips",
		Assets: config.AssetsConfig{
			Jingle: map[string]config.JingleEntry{
				"j1": {File: "/assets/j1.wav"},
				"j2": {File: "/assets/j2.wav"},
			},
		},
		PauseSec: 0.5,
		OutPath:  "/out.mp3",
	}

	args, err := BuildFFmpegArgs(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	foundJ1, foundJ2 := false, false
	for _, inp := range args.Inputs {
		if inp.Path == "/assets/j1.wav" {
			foundJ1 = true
		}
		if inp.Path == "/assets/j2.wav" {
			foundJ2 = true
		}
	}
	if !foundJ1 {
		t.Errorf("j1 not in inputs: %v", args.Inputs)
	}
	if !foundJ2 {
		t.Errorf("j2 not in inputs: %v", args.Inputs)
	}
}

// TestBuildFFmpegArgs_JingleUnknownAssetSkipped verifies that a jingle segment with an
// unknown asset_name (not in Assets.Jingle) is silently skipped.
func TestBuildFFmpegArgs_JingleUnknownAssetSkipped(t *testing.T) {
	ctx := BuildContext{
		Script: model.Script{
			Segments: []model.ScriptSegment{
				{Type: model.SegmentTypeJingle, AssetName: "missing"},
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
	for _, inp := range args.Inputs {
		if strings.Contains(inp.Path, "jingle") || strings.Contains(inp.Path, "missing") {
			t.Errorf("unexpected jingle input when key is missing: %s", inp.Path)
		}
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

	chimeCount := 0
	for _, inp := range args.Inputs {
		if inp.Path == "/assets/chime.wav" {
			chimeCount++
		}
	}
	if chimeCount != 2 {
		t.Errorf("chime.wav input count: got %d, want 2 (one per SE usage)", chimeCount)
	}
}

// TestBuildFFmpegArgs_LoudnormAppliedOnce verifies that loudnorm is applied exactly once
// to the full assembled output (after all concat operations).
func TestBuildFFmpegArgs_LoudnormAppliedOnce(t *testing.T) {
	ctx := BuildContext{
		Script: model.Script{
			Segments: []model.ScriptSegment{
				{Type: model.SegmentTypeJingle, AssetName: "opening"},
				{Type: model.SegmentTypeSpeech, SpeakerRole: "host", Text: "hello"},
				{Type: model.SegmentTypeJingle, AssetName: "ending"},
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
				"opening": {File: "/assets/opening.wav"},
				"ending":  {File: "/assets/ending.wav"},
			},
		},
		PauseSec: 0.5,
		OutPath:  "/out.mp3",
	}

	args, err := BuildFFmpegArgs(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Count loudnorm occurrences in filter_complex
	loudnormCount := strings.Count(args.FilterComplex, "loudnorm")
	if loudnormCount != 1 {
		t.Errorf("loudnorm should appear exactly once, got %d times: %s", loudnormCount, args.FilterComplex)
	}
}

// TestBuildFFmpegArgs_FinalOutputIsAlimiter verifies that the final [out] stage uses
// alimiter (peak protection), not loudnorm.
func TestBuildFFmpegArgs_FinalOutputIsAlimiter(t *testing.T) {
	for _, name := range []string{"no jingle", "with jingle"} {
		t.Run(name, func(t *testing.T) {
			script := model.Script{
				Segments: []model.ScriptSegment{
					{Type: model.SegmentTypeSpeech, SpeakerRole: "host", Text: "hello"},
				},
			}
			assets := config.AssetsConfig{}
			if name == "with jingle" {
				script.Segments = append([]model.ScriptSegment{
					{Type: model.SegmentTypeJingle, AssetName: "opening"},
				}, script.Segments...)
				assets.Jingle = map[string]config.JingleEntry{
					"opening": {File: "/assets/opening.wav"},
				}
			}
			ctx := BuildContext{
				Script: script,
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
						t.Errorf("final [out] must not use loudnorm: %s", f)
					}
					if !strings.Contains(f, "alimiter") {
						t.Errorf("final [out] must use alimiter: %s", f)
					}
					return
				}
			}
			t.Error("[out] label not found in filter_complex")
		})
	}
}

// TestBuildFFmpegArgs_LoudnormAfterJingleConcat verifies that loudnorm is applied AFTER
// the jingle/run concat.
func TestBuildFFmpegArgs_LoudnormAfterJingleConcat(t *testing.T) {
	ctx := BuildContext{
		Script: model.Script{
			Segments: []model.ScriptSegment{
				{Type: model.SegmentTypeJingle, AssetName: "opening"},
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
				"opening": {File: "/assets/opening.wav"},
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
	concatIdx := -1
	for i, f := range filters {
		if loudnormIdx == -1 && strings.Contains(f, "loudnorm") {
			loudnormIdx = i
		}
		if concatIdx == -1 && strings.Contains(f, "concat") {
			concatIdx = i
		}
		if loudnormIdx != -1 && concatIdx != -1 {
			break
		}
	}
	if loudnormIdx == -1 {
		t.Fatal("loudnorm not found in filter_complex")
	}
	if concatIdx == -1 {
		t.Fatal("concat not found in filter_complex")
	}
	if loudnormIdx < concatIdx {
		t.Errorf("loudnorm (index %d) must appear AFTER concat (index %d)", loudnormIdx, concatIdx)
	}
}

// TestBuildFFmpegArgs_PauseSegment_AddsExtraSilence verifies that a pause segment
// injects extra silence into the speech timeline beyond the default segment_pause_sec gap.
func TestBuildFFmpegArgs_PauseSegment_AddsExtraSilence(t *testing.T) {
	ctx := BuildContext{
		Script: model.Script{
			Segments: []model.ScriptSegment{
				{Type: model.SegmentTypeSpeech, SpeakerRole: "host", Text: "A"},
				{Type: model.SegmentTypePause, DurationSec: 1.2},
				{Type: model.SegmentTypeSpeech, SpeakerRole: "guest", Text: "B"},
			},
		},
		Clips: model.ClipsMeta{
			Clips: []model.ClipMeta{
				{Index: 0, File: "clip_000.wav", DurationSec: 2.0},
				{Index: 1, File: "clip_001.wav", DurationSec: 3.0},
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

	// Should have two anullsrc entries: one for pauseSec(0.5s) and one for the explicit pause(1.2s)
	silenceCount := strings.Count(args.FilterComplex, "anullsrc")
	if silenceCount < 2 {
		t.Errorf("expected at least 2 anullsrc entries (pauseSec + explicit pause), got %d: %s", silenceCount, args.FilterComplex)
	}
	// Explicit pause duration must appear in filter
	if !strings.Contains(args.FilterComplex, "atrim=duration=1.200") {
		t.Errorf("filter_complex missing explicit pause duration 1.200: %s", args.FilterComplex)
	}
}

// TestBuildFFmpegArgs_SEAfterPause_HasShiftedOffset verifies that the SE placed
// after a pause segment has its adelay shifted by the pause duration_sec.
func TestBuildFFmpegArgs_SEAfterPause_HasShiftedOffset(t *testing.T) {
	// speech(2s) + pauseSec(0.5s) = 2500ms before pause segment
	// pause(1.2s) → durationMs becomes 3700ms
	// SE at that point should have adelay=3700
	ctx := BuildContext{
		Script: model.Script{
			Segments: []model.ScriptSegment{
				{Type: model.SegmentTypeSpeech, SpeakerRole: "host", Text: "A"},
				{Type: model.SegmentTypePause, DurationSec: 1.2},
				{Type: model.SegmentTypeSE, AssetName: "chime"},
				{Type: model.SegmentTypeSpeech, SpeakerRole: "guest", Text: "B"},
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
				"chime": {File: "/assets/chime.wav", Volume: 1.0},
			},
		},
		PauseSec: 0.5,
		OutPath:  "/out.mp3",
	}

	args, err := BuildFFmpegArgs(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// SE offset should be 3700ms (2000 + 500 + 1200)
	if !strings.Contains(args.FilterComplex, "adelay=3700|3700") {
		t.Errorf("SE adelay should be 3700 (shifted by pause), filter: %s", args.FilterComplex)
	}
}

// TestBuildFFmpegArgs_PauseSegment_ZeroDurationIgnored verifies that a pause segment
// with duration_sec <= 0 does not add silence to the speech timeline.
func TestBuildFFmpegArgs_PauseSegment_ZeroDurationIgnored(t *testing.T) {
	ctx := BuildContext{
		Script: model.Script{
			Segments: []model.ScriptSegment{
				{Type: model.SegmentTypeSpeech, SpeakerRole: "host", Text: "A"},
				{Type: model.SegmentTypePause, DurationSec: 0},
				{Type: model.SegmentTypeSpeech, SpeakerRole: "guest", Text: "B"},
			},
		},
		Clips: model.ClipsMeta{
			Clips: []model.ClipMeta{
				{Index: 0, File: "clip_000.wav", DurationSec: 2.0},
				{Index: 1, File: "clip_001.wav", DurationSec: 3.0},
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

	// Only one anullsrc (the default pauseSec between the two clips), no extra silence
	silenceCount := strings.Count(args.FilterComplex, "anullsrc")
	if silenceCount != 1 {
		t.Errorf("zero-duration pause should not add extra silence, got %d anullsrc: %s", silenceCount, args.FilterComplex)
	}
}

// TestBuildFFmpegArgs_PauseSegment_BGMContinues verifies that BGM continues through
// a pause segment without interruption (the BGM interval covers the pause duration).
func TestBuildFFmpegArgs_PauseSegment_BGMContinues(t *testing.T) {
	ctx := BuildContext{
		Script: model.Script{
			Segments: []model.ScriptSegment{
				{Type: model.SegmentTypeBGM, AssetName: "talk_bgm"},
				{Type: model.SegmentTypeSpeech, SpeakerRole: "host", Text: "A"},
				{Type: model.SegmentTypePause, DurationSec: 1.2},
				{Type: model.SegmentTypeSpeech, SpeakerRole: "guest", Text: "B"},
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
			BGM: map[string]config.BGMEntry{
				"talk_bgm": {File: "/assets/bgm.mp3", Volume: 0.3, Loop: true},
			},
		},
		PauseSec: 0.5,
		OutPath:  "/out.mp3",
	}

	args, err := BuildFFmpegArgs(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// BGM should be present and loop (continues through pause)
	foundBGM := false
	for _, inp := range args.Inputs {
		if inp.Path == "/assets/bgm.mp3" {
			foundBGM = true
		}
	}
	if !foundBGM {
		t.Errorf("BGM input not found in inputs: %v", args.Inputs)
	}
	if !hasStreamLoop(args.Inputs, "/assets/bgm.mp3") {
		t.Errorf("loop:true BGM should have -stream_loop -1 in PreOptions, inputs: %v", args.Inputs)
	}
	if strings.Contains(args.FilterComplex, "aloop") {
		t.Errorf("loop:true BGM must not use aloop filter, filter: %s", args.FilterComplex)
	}
	// BGM duration: 2s(clip0) + 0.5s(pauseSec) + 1.2s(pause) + 3s(clip1) + 0.5s(pauseSec) = 7.2s
	// atrim=duration=7.200 confirms BGM covers the explicit pause duration.
	if !strings.Contains(args.FilterComplex, "atrim=duration=7.200") {
		t.Errorf("BGM atrim duration should be 7.200 (covers pause), filter: %s", args.FilterComplex)
	}
}

// TestBuildFFmpegArgs_BGMLoop_StreamLoop verifies that loop:true BGM uses -stream_loop -1
// as a pre-input option instead of the aloop filter, enabling full-file looping.
func TestBuildFFmpegArgs_BGMLoop_StreamLoop(t *testing.T) {
	ctx := BuildContext{
		Script: model.Script{
			Segments: []model.ScriptSegment{
				{Type: model.SegmentTypeBGM, AssetName: "talk_bgm"},
				{Type: model.SegmentTypeSpeech, SpeakerRole: "host", Text: "with bgm"},
			},
		},
		Clips: model.ClipsMeta{
			Clips: []model.ClipMeta{
				{Index: 0, File: "clip_000.wav", DurationSec: 30.0},
			},
		},
		ClipsDir: "/clips",
		Assets: config.AssetsConfig{
			BGM: map[string]config.BGMEntry{
				"talk_bgm": {File: "/assets/bgm.mp3", Volume: 0.3, Loop: true},
			},
		},
		PauseSec: 0.5,
		OutPath:  "/out.mp3",
	}

	args, err := BuildFFmpegArgs(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// BGM must be present in inputs.
	foundBGM := false
	for _, inp := range args.Inputs {
		if inp.Path == "/assets/bgm.mp3" {
			foundBGM = true
		}
	}
	if !foundBGM {
		t.Errorf("BGM input /assets/bgm.mp3 not found in inputs: %v", args.Inputs)
	}

	// -stream_loop -1 must be set as PreOptions on the BGM input.
	if !hasStreamLoop(args.Inputs, "/assets/bgm.mp3") {
		t.Errorf("loop:true BGM should have -stream_loop -1 in PreOptions, inputs: %v", args.Inputs)
	}

	// aloop must NOT appear in filter_complex for loop:true BGM.
	if strings.Contains(args.FilterComplex, "aloop") {
		t.Errorf("loop:true BGM must not use aloop filter, filter: %s", args.FilterComplex)
	}
}

// TestBuildFFmpegArgs_BGMStopThenPlay_BothIntervalsInOutput verifies that when BGM stops
// and restarts within one run, both intervals appear in the filter output.
// This is a regression test for the bug where amix with duration=first truncated
// the second and later BGM intervals.
func TestBuildFFmpegArgs_BGMStopThenPlay_BothIntervalsInOutput(t *testing.T) {
	// Script: BGM(a), speech1, BGM(""), speech2, BGM(a), speech3
	// Interval 1: [0, ~2500ms], Interval 2: [~6000ms, end]
	ctx := BuildContext{
		Script: model.Script{
			Segments: []model.ScriptSegment{
				{Type: model.SegmentTypeBGM, AssetName: "talk_bgm"},
				{Type: model.SegmentTypeSpeech, SpeakerRole: "host", Text: "with bgm"},
				{Type: model.SegmentTypeBGM, AssetName: ""},
				{Type: model.SegmentTypeSpeech, SpeakerRole: "host", Text: "no bgm"},
				{Type: model.SegmentTypeBGM, AssetName: "talk_bgm"},
				{Type: model.SegmentTypeSpeech, SpeakerRole: "host", Text: "bgm again"},
			},
		},
		Clips: model.ClipsMeta{
			Clips: []model.ClipMeta{
				{Index: 0, File: "clip_000.wav", DurationSec: 2.0},
				{Index: 1, File: "clip_001.wav", DurationSec: 3.0},
				{Index: 2, File: "clip_002.wav", DurationSec: 2.0},
			},
		},
		ClipsDir: "/clips",
		Assets: config.AssetsConfig{
			BGM: map[string]config.BGMEntry{
				"talk_bgm": {File: "/assets/bgm.mp3", Volume: 0.3, Loop: true},
			},
		},
		PauseSec: 0.5,
		OutPath:  "/out.mp3",
	}

	args, err := BuildFFmpegArgs(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// speech1(2s) + pauseSec(0.5s) = 2500ms → BGM stop
	// speech2(3s) + pauseSec(0.5s) = 3500ms → BGM restart at 2500+3500=6000ms
	// 2nd interval adelay should be 6000ms
	if !strings.Contains(args.FilterComplex, "adelay=6000|6000") {
		t.Errorf("2nd BGM interval should have adelay=6000 (started after speech1+pause+speech2+pause), filter: %s", args.FilterComplex)
	}

	// The filter that merges the two BGM raw intervals (bgm0_raw + bgm1_raw) must use
	// duration=longest so neither interval is truncated.
	// (The later BGM×speech amix intentionally uses duration=first; only the interval merge is checked here.)
	filters := strings.Split(args.FilterComplex, ";")
	for _, f := range filters {
		if strings.Contains(f, "bgm0_raw") && strings.Contains(f, "bgm1_raw") {
			if !strings.Contains(f, "duration=longest") {
				t.Errorf("BGM interval amix must use duration=longest (not duration=first): %s", f)
			}
		}
	}
}

// TestBuildFFmpegArgs_BGMNoLoop_NoStreamLoop verifies that loop:false BGM does NOT
// get -stream_loop -1 (it should play once and stop).
func TestBuildFFmpegArgs_BGMNoLoop_NoStreamLoop(t *testing.T) {
	ctx := BuildContext{
		Script: model.Script{
			Segments: []model.ScriptSegment{
				{Type: model.SegmentTypeBGM, AssetName: "talk_bgm"},
				{Type: model.SegmentTypeSpeech, SpeakerRole: "host", Text: "with bgm"},
			},
		},
		Clips: model.ClipsMeta{
			Clips: []model.ClipMeta{
				{Index: 0, File: "clip_000.wav", DurationSec: 5.0},
			},
		},
		ClipsDir: "/clips",
		Assets: config.AssetsConfig{
			BGM: map[string]config.BGMEntry{
				"talk_bgm": {File: "/assets/bgm.mp3", Volume: 0.3, Loop: false},
			},
		},
		PauseSec: 0.5,
		OutPath:  "/out.mp3",
	}

	args, err := BuildFFmpegArgs(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	for _, inp := range args.Inputs {
		if inp.Path == "/assets/bgm.mp3" {
			for _, opt := range inp.PreOptions {
				if opt == "-stream_loop" {
					t.Errorf("loop:false BGM should not have -stream_loop, got PreOptions: %v", inp.PreOptions)
				}
			}
		}
	}
}

// TestBuildFFmpegArgs_JingleSilenceRemoveDefault verifies that silenceremove is applied
// to a jingle when TrimSilence is nil (default = true).
func TestBuildFFmpegArgs_JingleSilenceRemoveDefault(t *testing.T) {
	ctx := BuildContext{
		Script: model.Script{
			Segments: []model.ScriptSegment{
				{Type: model.SegmentTypeJingle, AssetName: "op"},
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
				"op": {File: "/assets/op.wav", FadeIn: 0.3},
			},
		},
		PauseSec: 0.5,
		OutPath:  "/out.mp3",
	}

	args, err := BuildFFmpegArgs(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(args.FilterComplex, "silenceremove") {
		t.Errorf("filter_complex missing silenceremove for jingle with default TrimSilence: %s", args.FilterComplex)
	}
}

// TestBuildFFmpegArgs_JingleSilenceRemoveDisabled verifies that silenceremove is NOT applied
// when TrimSilence is explicitly false.
func TestBuildFFmpegArgs_JingleSilenceRemoveDisabled(t *testing.T) {
	f := false
	ctx := BuildContext{
		Script: model.Script{
			Segments: []model.ScriptSegment{
				{Type: model.SegmentTypeJingle, AssetName: "op"},
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
				"op": {File: "/assets/op.wav", FadeIn: 0.3, TrimSilence: &f},
			},
		},
		PauseSec: 0.5,
		OutPath:  "/out.mp3",
	}

	args, err := BuildFFmpegArgs(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if strings.Contains(args.FilterComplex, "silenceremove") {
		t.Errorf("filter_complex must not contain silenceremove when TrimSilence=false: %s", args.FilterComplex)
	}
}

// TestBuildFFmpegArgs_SESilenceRemoveDefault verifies that silenceremove is applied
// to a SE when TrimSilence is nil (default = true).
func TestBuildFFmpegArgs_SESilenceRemoveDefault(t *testing.T) {
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
	if !strings.Contains(args.FilterComplex, "silenceremove") {
		t.Errorf("filter_complex missing silenceremove for SE with default TrimSilence: %s", args.FilterComplex)
	}
}

// TestBuildFFmpegArgs_SESilenceRemoveDisabled verifies that silenceremove is NOT applied
// when SE TrimSilence is explicitly false.
func TestBuildFFmpegArgs_SESilenceRemoveDisabled(t *testing.T) {
	f := false
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
				"chime": {File: "/assets/chime.wav", Volume: 0.8, TrimSilence: &f},
			},
		},
		PauseSec: 0.5,
		OutPath:  "/out.mp3",
	}

	args, err := BuildFFmpegArgs(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if strings.Contains(args.FilterComplex, "silenceremove") {
		t.Errorf("filter_complex must not contain silenceremove when SE TrimSilence=false: %s", args.FilterComplex)
	}
}
