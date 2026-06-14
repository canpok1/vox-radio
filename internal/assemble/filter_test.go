package assemble

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/canpok1/vox-radio/internal/config"
	"github.com/canpok1/vox-radio/internal/model"
)

func boolPtr(v bool) *bool { return &v }

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
				"chime": {File: "/assets/chime.wav", Volume: 0.8, Overlay: boolPtr(true)},
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
				"chime": {File: "/assets/chime.wav", Volume: 1.0, Overlay: boolPtr(true)},
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

	// All three speeches are same speaker (host), so no inter-clip pauses (continuation).
	// speech1(2s) → speech2(3s): BGM stop at 2000ms, BGM restart at 2000+3000=5000ms.
	// 2nd interval adelay should be 5000ms.
	if !strings.Contains(args.FilterComplex, "adelay=5000|5000") {
		t.Errorf("2nd BGM interval should have adelay=5000 (same-speaker continuation, no pauses), filter: %s", args.FilterComplex)
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

// TestBuildFFmpegArgs_SameSpeakerContinuation_NoSilence verifies that consecutive clips
// from the same speaker are joined without silence between them.
func TestBuildFFmpegArgs_SameSpeakerContinuation_NoSilence(t *testing.T) {
	ctx := BuildContext{
		Script: model.Script{
			Segments: []model.ScriptSegment{
				{Type: model.SegmentTypeSpeech, SpeakerRole: "host", Text: "A"},
				{Type: model.SegmentTypeSpeech, SpeakerRole: "host", Text: "B"},
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
		PauseSec: 0.3,
		OutPath:  "/out.mp3",
	}

	args, err := BuildFFmpegArgs(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Same-speaker consecutive clips must NOT have anullsrc silence between them.
	if strings.Contains(args.FilterComplex, "anullsrc") {
		t.Errorf("same-speaker continuation must not insert silence, filter: %s", args.FilterComplex)
	}
}

// TestBuildFFmpegArgs_DifferentSpeaker_HasDefaultPause verifies that clips from different
// speakers have the default pauseSec silence inserted between them.
func TestBuildFFmpegArgs_DifferentSpeaker_HasDefaultPause(t *testing.T) {
	ctx := BuildContext{
		Script: model.Script{
			Segments: []model.ScriptSegment{
				{Type: model.SegmentTypeSpeech, SpeakerRole: "host", Text: "A"},
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
		PauseSec: 0.3,
		OutPath:  "/out.mp3",
	}

	args, err := BuildFFmpegArgs(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(args.FilterComplex, "anullsrc") {
		t.Errorf("different-speaker clips must insert silence, filter: %s", args.FilterComplex)
	}
	if !strings.Contains(args.FilterComplex, "atrim=duration=0.300") {
		t.Errorf("different-speaker pause must be 0.300s, filter: %s", args.FilterComplex)
	}
}

// TestBuildFFmpegArgs_ExplicitPause_BreaksContinuation verifies that an explicit pause segment
// between same-speaker clips is NOT treated as continuation (silence is still inserted).
func TestBuildFFmpegArgs_ExplicitPause_BreaksContinuation(t *testing.T) {
	ctx := BuildContext{
		Script: model.Script{
			Segments: []model.ScriptSegment{
				{Type: model.SegmentTypeSpeech, SpeakerRole: "host", Text: "A"},
				{Type: model.SegmentTypePause, DurationSec: 1.0},
				{Type: model.SegmentTypeSpeech, SpeakerRole: "host", Text: "B"},
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
		PauseSec: 0.3,
		OutPath:  "/out.mp3",
	}

	args, err := BuildFFmpegArgs(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Explicit pause breaks continuation: both defaultPauseSec and explicit pause must appear.
	if !strings.Contains(args.FilterComplex, "atrim=duration=1.000") {
		t.Errorf("explicit pause duration 1.000s must appear, filter: %s", args.FilterComplex)
	}
	if !strings.Contains(args.FilterComplex, "atrim=duration=0.300") {
		t.Errorf("default pause 0.300s must appear (explicit pause breaks continuation), filter: %s", args.FilterComplex)
	}
}

// TestBuildFFmpegArgs_SameSpeakerContinuation_SEOffsetConsistent verifies that when two
// same-speaker clips are a continuation (no silence between them), the SE offset reflects
// the actual timeline (durationMs does not include the omitted pause).
func TestBuildFFmpegArgs_SameSpeakerContinuation_SEOffsetConsistent(t *testing.T) {
	// clip0(host,2s) → SE → clip1(host,3s): continuation, no inter-clip pause.
	// SE offset = clip0.duration = 2000ms (NOT 2000 + pauseSec).
	ctx := BuildContext{
		Script: model.Script{
			Segments: []model.ScriptSegment{
				{Type: model.SegmentTypeSpeech, SpeakerRole: "host", Text: "A"},
				{Type: model.SegmentTypeSE, AssetName: "chime"},
				{Type: model.SegmentTypeSpeech, SpeakerRole: "host", Text: "B"},
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
				"chime": {File: "/assets/chime.wav", Volume: 1.0, Overlay: boolPtr(true)},
			},
		},
		PauseSec: 0.3,
		OutPath:  "/out.mp3",
	}

	args, err := BuildFFmpegArgs(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// SE offset must be 2000ms: clip0 duration only, no pause (same-speaker continuation).
	if !strings.Contains(args.FilterComplex, "adelay=2000|2000") {
		t.Errorf("SE offset must be 2000ms (no pause for same-speaker continuation), filter: %s", args.FilterComplex)
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

// TestBuildFFmpegArgs_SE_SequentialDefault verifies that a SE with no Overlay setting (default=false)
// is placed sequentially in the concat chain, not overlaid via adelay/amix.
func TestBuildFFmpegArgs_SE_SequentialDefault(t *testing.T) {
	ctx := BuildContext{
		Script: model.Script{
			Segments: []model.ScriptSegment{
				{Type: model.SegmentTypeSpeech, SpeakerRole: "host", Text: "intro"},
				{Type: model.SegmentTypeSE, AssetName: "chime"},
				{Type: model.SegmentTypeSpeech, SpeakerRole: "guest", Text: "main"},
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
		SEDurations: map[string]float64{"chime": 1.5},
		PauseSec:    0.5,
		OutPath:     "/out.mp3",
	}

	args, err := BuildFFmpegArgs(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Sequential SE must appear in concat (volume filter applied), not as overlay.
	if strings.Contains(args.FilterComplex, "adelay") {
		t.Errorf("sequential SE must not use adelay, filter: %s", args.FilterComplex)
	}
	if strings.Contains(args.FilterComplex, "amix") {
		t.Errorf("sequential SE must not use amix, filter: %s", args.FilterComplex)
	}
	foundSE := false
	for _, inp := range args.Inputs {
		if inp.Path == "/assets/chime.wav" {
			foundSE = true
		}
	}
	if !foundSE {
		t.Errorf("sequential SE input /assets/chime.wav not found: %v", args.Inputs)
	}
	if !strings.Contains(args.FilterComplex, "concat") {
		t.Errorf("sequential SE should produce a concat filter: %s", args.FilterComplex)
	}
}

// TestBuildFFmpegArgs_SE_SequentialDurationAdvancesOverlaySEOffset verifies that
// after a sequential SE, any following overlay SE has its offsetMs shifted by the SE's duration.
func TestBuildFFmpegArgs_SE_SequentialDurationAdvancesOverlaySEOffset(t *testing.T) {
	// durationMs trace:
	//   clip0(host,2s) + pauseAfter(0.5s, different speaker) = 2500ms
	//   seq_se(1.5s)                                          = 4000ms
	//   clip1(guest,3s) + pauseAfter(0.5s, no next speech)   = 7500ms  ← overlay_se fires here
	ctx := BuildContext{
		Script: model.Script{
			Segments: []model.ScriptSegment{
				{Type: model.SegmentTypeSpeech, SpeakerRole: "host", Text: "A"},
				{Type: model.SegmentTypeSE, AssetName: "seq_se"},
				{Type: model.SegmentTypeSpeech, SpeakerRole: "guest", Text: "B"},
				{Type: model.SegmentTypeSE, AssetName: "overlay_se"},
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
				"seq_se":     {File: "/assets/seq.wav", Volume: 1.0},
				"overlay_se": {File: "/assets/overlay.wav", Volume: 1.0, Overlay: boolPtr(true)},
			},
		},
		SEDurations: map[string]float64{"seq_se": 1.5},
		PauseSec:    0.5,
		OutPath:     "/out.mp3",
	}

	args, err := BuildFFmpegArgs(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// overlay_se offset = 7500ms (includes clip1's trailing pauseAfter in durationMs)
	if !strings.Contains(args.FilterComplex, "adelay=7500|7500") {
		t.Errorf("overlay SE adelay should be 7500ms (after sequential SE duration), filter: %s", args.FilterComplex)
	}
}

func float64Ptr(v float64) *float64 { return &v }

// --- golden test ---

// TestBuildFFmpegArgs_Golden_FilterComplex verifies that the exact filter_complex output
// for a comprehensive scenario (jingle / same-speaker / BGM-duck / SE overlay / jingle)
// does not change across refactoring.
// Run with UPDATE_GOLDEN=1 to regenerate testdata/golden_filter_complex.txt.
func TestBuildFFmpegArgs_Golden_FilterComplex(t *testing.T) {
	ctx := BuildContext{
		Script: model.Script{
			Segments: []model.ScriptSegment{
				{Type: model.SegmentTypeJingle, AssetName: "op"},
				{Type: model.SegmentTypeSpeech, SpeakerRole: "host", Text: "A"},
				{Type: model.SegmentTypeSpeech, SpeakerRole: "host", Text: "B"},
				{Type: model.SegmentTypeBGM, AssetName: "talk_bgm"},
				{Type: model.SegmentTypeSpeech, SpeakerRole: "guest", Text: "C"},
				{Type: model.SegmentTypeSE, AssetName: "chime"},
				{Type: model.SegmentTypeSpeech, SpeakerRole: "host", Text: "D"},
				{Type: model.SegmentTypeJingle, AssetName: "ed"},
			},
		},
		Clips: model.ClipsMeta{
			Clips: []model.ClipMeta{
				{Index: 0, File: "clip_000.wav", DurationSec: 1.5},
				{Index: 1, File: "clip_001.wav", DurationSec: 2.0},
				{Index: 2, File: "clip_002.wav", DurationSec: 1.0},
				{Index: 3, File: "clip_003.wav", DurationSec: 3.0},
			},
		},
		ClipsDir: "/clips",
		Assets: config.AssetsConfig{
			BGM: map[string]config.BGMEntry{
				"talk_bgm": {File: "/assets/bgm.mp3", Volume: 0.3, Loop: true, DuckRatio: 4.0},
			},
			SE: map[string]config.SEEntry{
				"chime": {File: "/assets/chime.wav", Volume: 0.8, Overlay: boolPtr(true)},
			},
			Jingle: map[string]config.JingleEntry{
				"op": {File: "/assets/op.wav", FadeIn: 0.3},
				"ed": {File: "/assets/ed.wav", FadeOut: 1.0},
			},
		},
		PauseSec: 0.5,
		OutPath:  "/out.mp3",
	}

	args, err := BuildFFmpegArgs(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	goldenFile := "testdata/golden_filter_complex.txt"
	if os.Getenv("UPDATE_GOLDEN") == "1" {
		if err := os.MkdirAll("testdata", 0755); err != nil {
			t.Fatalf("failed to create testdata dir: %v", err)
		}
		if err := os.WriteFile(goldenFile, []byte(args.FilterComplex), 0644); err != nil {
			t.Fatalf("failed to write golden file: %v", err)
		}
		t.Logf("golden file updated: %s", goldenFile)
		return
	}

	want, err := os.ReadFile(goldenFile)
	if err != nil {
		t.Fatalf("golden file not found (run with UPDATE_GOLDEN=1 to create): %v", err)
	}
	if args.FilterComplex != string(want) {
		t.Errorf("filter_complex mismatch:\ngot:  %s\nwant: %s", args.FilterComplex, string(want))
	}
}

// --- buildDuckSplit unit tests ---

func TestBuildDuckSplit_NoBGMIntervals(t *testing.T) {
	b := &filterBuilder{}
	outLabel, duckLabel, _ := buildDuckSplit(b, nil, config.AssetsConfig{}, 0, "[input]")
	if outLabel != "[input]" {
		t.Errorf("outLabel: got %q, want %q", outLabel, "[input]")
	}
	if duckLabel != "" {
		t.Errorf("duckLabel: got %q, want empty", duckLabel)
	}
	if len(b.filters) != 0 {
		t.Errorf("no filters should be emitted, got %d", len(b.filters))
	}
}

func TestBuildDuckSplit_BGMWithNoDuckRatio(t *testing.T) {
	b := &filterBuilder{}
	intervals := []bgmInterval{{assetName: "bgm1", startMs: 0, endMs: -1}}
	assets := config.AssetsConfig{
		BGM: map[string]config.BGMEntry{
			"bgm1": {File: "/bgm.mp3", Volume: 0.3, DuckRatio: 0},
		},
	}
	outLabel, duckLabel, _ := buildDuckSplit(b, intervals, assets, 0, "[input]")
	if outLabel != "[input]" {
		t.Errorf("outLabel: got %q, want %q", outLabel, "[input]")
	}
	if duckLabel != "" {
		t.Errorf("duckLabel: got %q, want empty", duckLabel)
	}
	if len(b.filters) != 0 {
		t.Errorf("no filters should be emitted, got %d", len(b.filters))
	}
}

func TestBuildDuckSplit_BGMWithDuckRatio(t *testing.T) {
	b := &filterBuilder{}
	intervals := []bgmInterval{{assetName: "bgm1", startMs: 0, endMs: -1}}
	assets := config.AssetsConfig{
		BGM: map[string]config.BGMEntry{
			"bgm1": {File: "/bgm.mp3", Volume: 0.3, DuckRatio: 4.0},
		},
	}
	outLabel, duckLabel, duckRatio := buildDuckSplit(b, intervals, assets, 0, "[input]")
	if outLabel == "[input]" {
		t.Errorf("outLabel should change when ducking is applied")
	}
	if duckLabel == "" {
		t.Errorf("duckLabel should be set when ducking is applied")
	}
	if duckRatio != 4.0 {
		t.Errorf("duckRatio: got %v, want 4.0", duckRatio)
	}
	if len(b.filters) != 1 {
		t.Fatalf("expected 1 filter, got %d", len(b.filters))
	}
	if !strings.Contains(b.filters[0], "asplit=2") {
		t.Errorf("filter should contain asplit=2, got: %s", b.filters[0])
	}
}

// --- buildSEOverlay unit tests ---

func TestBuildSEOverlay_NoEvents(t *testing.T) {
	b := &filterBuilder{}
	outLabel := buildSEOverlay(b, nil, config.AssetsConfig{}, 0, "[input]")
	if outLabel != "[input]" {
		t.Errorf("outLabel: got %q, want %q", outLabel, "[input]")
	}
	if len(b.filters) != 0 {
		t.Errorf("no filters should be emitted, got %d", len(b.filters))
	}
}

func TestBuildSEOverlay_UnknownAsset(t *testing.T) {
	b := &filterBuilder{}
	events := []seEvent{{assetName: "unknown", offsetMs: 1000}}
	outLabel := buildSEOverlay(b, events, config.AssetsConfig{SE: map[string]config.SEEntry{}}, 0, "[input]")
	if outLabel != "[input]" {
		t.Errorf("outLabel should be unchanged for unknown asset, got %q", outLabel)
	}
}

func TestBuildSEOverlay_OneOverlayEvent(t *testing.T) {
	b := &filterBuilder{}
	events := []seEvent{{assetName: "chime", offsetMs: 2000}}
	assets := config.AssetsConfig{
		SE: map[string]config.SEEntry{
			"chime": {File: "/chime.wav", Volume: 0.8, Overlay: boolPtr(true)},
		},
	}
	outLabel := buildSEOverlay(b, events, assets, 0, "[input]")
	if outLabel == "[input]" {
		t.Errorf("outLabel should change after SE overlay")
	}
	found := false
	for _, f := range b.filters {
		if strings.Contains(f, "adelay=2000|2000") {
			found = true
		}
	}
	if !found {
		t.Errorf("adelay=2000|2000 not found in filters: %v", b.filters)
	}
}

// --- runBuilder.closeBGMInterval unit tests ---

func TestRunBuilderCloseBGMInterval_NoActiveBGM(t *testing.T) {
	rb := newRun()
	rb.closeBGMInterval(5000)
	if len(rb.bgmIntervals) != 0 {
		t.Errorf("no interval should be appended when no active BGM, got %d", len(rb.bgmIntervals))
	}
}

func TestRunBuilderCloseBGMInterval_ExplicitEnd(t *testing.T) {
	rb := newRun()
	rb.activeBGMStart = 1000
	rb.activeBGMName = "bgm1"
	rb.closeBGMInterval(5000)
	if len(rb.bgmIntervals) != 1 {
		t.Fatalf("expected 1 interval, got %d", len(rb.bgmIntervals))
	}
	iv := rb.bgmIntervals[0]
	if iv.startMs != 1000 || iv.endMs != 5000 || iv.assetName != "bgm1" {
		t.Errorf("interval mismatch: got %+v", iv)
	}
	if rb.activeBGMStart != -1 {
		t.Errorf("activeBGMStart should be reset to -1, got %d", rb.activeBGMStart)
	}
	if rb.activeBGMName != "" {
		t.Errorf("activeBGMName should be reset, got %q", rb.activeBGMName)
	}
}

func TestRunBuilderCloseBGMInterval_ToEnd(t *testing.T) {
	rb := newRun()
	rb.activeBGMStart = 500
	rb.activeBGMName = "bgm2"
	rb.closeBGMInterval(-1)
	if len(rb.bgmIntervals) != 1 {
		t.Fatalf("expected 1 interval, got %d", len(rb.bgmIntervals))
	}
	if rb.bgmIntervals[0].endMs != -1 {
		t.Errorf("endMs should be -1, got %d", rb.bgmIntervals[0].endMs)
	}
	if rb.activeBGMStart != -1 {
		t.Errorf("activeBGMStart should be reset to -1, got %d", rb.activeBGMStart)
	}
}

// TestBuildFFmpegArgs_BGMFadeInOut_A1 verifies that a BGM interval with explicit FadeIn/FadeOut
// gets afade filters applied before adelay.
func TestBuildFFmpegArgs_BGMFadeInOut_A1(t *testing.T) {
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
				"talk_bgm": {File: "/assets/bgm.mp3", Volume: 0.3, Loop: true, FadeIn: float64Ptr(0.5), FadeOut: float64Ptr(0.5)},
			},
		},
		PauseSec: 0.5,
		OutPath:  "/out.mp3",
	}

	args, err := BuildFFmpegArgs(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Fade-in filter must appear.
	if !strings.Contains(args.FilterComplex, "afade=t=in:d=0.500") {
		t.Errorf("filter_complex missing afade fade-in for BGM: %s", args.FilterComplex)
	}
	// Fade-out filter (areverse trick) must appear.
	if !strings.Contains(args.FilterComplex, "areverse,afade=t=in:d=0.500,areverse") {
		t.Errorf("filter_complex missing areverse fade-out for BGM: %s", args.FilterComplex)
	}
}

// TestBuildFFmpegArgs_BGMFadeDefault_A5 verifies that a BGM with no FadeIn/FadeOut fields
// defaults to 1.0 second fade-in and fade-out.
func TestBuildFFmpegArgs_BGMFadeDefault_A5(t *testing.T) {
	ctx := BuildContext{
		Script: model.Script{
			Segments: []model.ScriptSegment{
				{Type: model.SegmentTypeBGM, AssetName: "talk_bgm"},
				{Type: model.SegmentTypeSpeech, SpeakerRole: "host", Text: "with bgm"},
			},
		},
		Clips: model.ClipsMeta{
			Clips: []model.ClipMeta{
				{Index: 0, File: "clip_000.wav", DurationSec: 10.0},
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

	// Default 1.0-second fade-in must appear.
	if !strings.Contains(args.FilterComplex, "afade=t=in:d=1.000") {
		t.Errorf("filter_complex missing default 1.0s fade-in for BGM: %s", args.FilterComplex)
	}
	// Default 1.0-second fade-out (areverse trick) must appear.
	if !strings.Contains(args.FilterComplex, "areverse,afade=t=in:d=1.000,areverse") {
		t.Errorf("filter_complex missing default 1.0s fade-out for BGM: %s", args.FilterComplex)
	}
}

// TestBuildFFmpegArgs_BGMFadeDisabled_A5 verifies that explicit FadeIn=0 / FadeOut=0
// produces no afade filters for that BGM.
func TestBuildFFmpegArgs_BGMFadeDisabled_A5(t *testing.T) {
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
				"talk_bgm": {File: "/assets/bgm.mp3", Volume: 0.3, Loop: true, FadeIn: float64Ptr(0), FadeOut: float64Ptr(0)},
			},
		},
		PauseSec: 0.5,
		OutPath:  "/out.mp3",
	}

	args, err := BuildFFmpegArgs(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// No afade filter should appear for this BGM.
	if strings.Contains(args.FilterComplex, "afade") {
		t.Errorf("filter_complex must not contain afade when FadeIn=FadeOut=0: %s", args.FilterComplex)
	}
}

// TestBuildFFmpegArgs_BGMCrossfade_A2 verifies that adjacent BGM intervals produce a crossfade:
// the first interval is extended and both intervals get afade filters.
func TestBuildFFmpegArgs_BGMCrossfade_A2(t *testing.T) {
	// Script: BGM(a) starts, speech1(5s), BGM(b) switches (adjacent to end of a), speech2(5s)
	// BGM a: [0, 5500ms] (5s clip + 0.5s pauseSec), BGM b: [5500ms, end]
	ctx := BuildContext{
		Script: model.Script{
			Segments: []model.ScriptSegment{
				{Type: model.SegmentTypeBGM, AssetName: "bgm_a"},
				{Type: model.SegmentTypeSpeech, SpeakerRole: "host", Text: "with bgm a"},
				{Type: model.SegmentTypeBGM, AssetName: "bgm_b"},
				{Type: model.SegmentTypeSpeech, SpeakerRole: "guest", Text: "with bgm b"},
			},
		},
		Clips: model.ClipsMeta{
			Clips: []model.ClipMeta{
				{Index: 0, File: "clip_000.wav", DurationSec: 5.0},
				{Index: 1, File: "clip_001.wav", DurationSec: 5.0},
			},
		},
		ClipsDir: "/clips",
		Assets: config.AssetsConfig{
			BGM: map[string]config.BGMEntry{
				"bgm_a": {File: "/assets/bgm_a.mp3", Volume: 0.3, Loop: true, FadeIn: float64Ptr(0.5), FadeOut: float64Ptr(1.0)},
				"bgm_b": {File: "/assets/bgm_b.mp3", Volume: 0.3, Loop: true, FadeIn: float64Ptr(1.0), FadeOut: float64Ptr(0.5)},
			},
		},
		PauseSec: 0.5,
		OutPath:  "/out.mp3",
	}

	args, err := BuildFFmpegArgs(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Both BGM inputs must be present.
	foundA, foundB := false, false
	for _, inp := range args.Inputs {
		if inp.Path == "/assets/bgm_a.mp3" {
			foundA = true
		}
		if inp.Path == "/assets/bgm_b.mp3" {
			foundB = true
		}
	}
	if !foundA {
		t.Errorf("bgm_a not in inputs: %v", args.Inputs)
	}
	if !foundB {
		t.Errorf("bgm_b not in inputs: %v", args.Inputs)
	}

	// afade must appear for both intervals (fade-in of A, fade-out of A, fade-in of B, fade-out of B).
	afadeCount := strings.Count(args.FilterComplex, "afade=t=in")
	if afadeCount < 3 {
		// At minimum: fade-in of A (0.5s), fade-out of A into crossfade (1.0s), fade-in of B (1.0s), fade-out of B (0.5s) = 4
		// But with areverse trick, each fade-out has one afade call.
		t.Errorf("expected at least 3 afade filters for crossfade (got %d): %s", afadeCount, args.FilterComplex)
	}

	// The first BGM interval (bgm_a) must be EXTENDED: atrim duration > 5.5s (original BGM a duration)
	// overlapSec = min(FadeOut_A=1.0, FadeIn_B=1.0) = 1.0
	// extended atrim = 5.5 + 1.0 = 6.5s → atrim=duration=6.500
	if !strings.Contains(args.FilterComplex, "atrim=duration=6.500") {
		t.Errorf("bgm_a should be extended for crossfade (atrim=duration=6.500): %s", args.FilterComplex)
	}
}

// TestBuildFFmpegArgs_BGMShortClamp_A4 verifies that a very short BGM interval (shorter than
// FadeIn+FadeOut) does not break the filter generation.
func TestBuildFFmpegArgs_BGMShortClamp_A4(t *testing.T) {
	ctx := BuildContext{
		Script: model.Script{
			Segments: []model.ScriptSegment{
				{Type: model.SegmentTypeBGM, AssetName: "talk_bgm"},
				{Type: model.SegmentTypeSpeech, SpeakerRole: "host", Text: "short"},
				{Type: model.SegmentTypeBGM, AssetName: ""},
			},
		},
		Clips: model.ClipsMeta{
			Clips: []model.ClipMeta{
				{Index: 0, File: "clip_000.wav", DurationSec: 0.8},
			},
		},
		ClipsDir: "/clips",
		Assets: config.AssetsConfig{
			BGM: map[string]config.BGMEntry{
				// FadeIn=1.0 and FadeOut=1.0 both exceed half the BGM interval duration
				"talk_bgm": {File: "/assets/bgm.mp3", Volume: 0.3, Loop: true, FadeIn: float64Ptr(1.0), FadeOut: float64Ptr(1.0)},
			},
		},
		PauseSec: 0.5,
		OutPath:  "/out.mp3",
	}

	// Should not error (filter is generated without breaking).
	_, err := BuildFFmpegArgs(ctx)
	if err != nil {
		t.Errorf("unexpected error for short BGM with large fade: %v", err)
	}
}

// TestBuildFFmpegArgs_SilenceRemoveExplicitThreshold verifies that an explicit
// trim_silence_threshold is reflected in the silenceremove filter for jingle, SE sequential, and SE overlay.
func TestBuildFFmpegArgs_SilenceRemoveExplicitThreshold(t *testing.T) {
	threshold := -40.0
	overlayTrue := true
	seSeg := []model.ScriptSegment{
		{Type: model.SegmentTypeSpeech, SpeakerRole: "host", Text: "intro"},
		{Type: model.SegmentTypeSE, AssetName: "chime"},
		{Type: model.SegmentTypeSpeech, SpeakerRole: "host", Text: "main"},
	}
	seClips := []model.ClipMeta{
		{Index: 0, File: "clip_000.wav", DurationSec: 2.0},
		{Index: 1, File: "clip_001.wav", DurationSec: 3.0},
	}
	cases := []struct {
		name   string
		segs   []model.ScriptSegment
		clips  []model.ClipMeta
		assets config.AssetsConfig
	}{
		{
			name: "jingle",
			segs: []model.ScriptSegment{
				{Type: model.SegmentTypeJingle, AssetName: "op"},
				{Type: model.SegmentTypeSpeech, SpeakerRole: "host", Text: "hello"},
			},
			clips: []model.ClipMeta{{Index: 0, File: "clip_000.wav", DurationSec: 2.0}},
			assets: config.AssetsConfig{
				Jingle: map[string]config.JingleEntry{
					"op": {File: "/assets/op.wav", FadeIn: 0.3, TrimSilenceThreshold: &threshold},
				},
			},
		},
		{
			name:  "SE sequential",
			segs:  seSeg,
			clips: seClips,
			assets: config.AssetsConfig{
				SE: map[string]config.SEEntry{
					"chime": {File: "/assets/chime.wav", Volume: 0.8, TrimSilenceThreshold: &threshold},
				},
			},
		},
		{
			name:  "SE overlay",
			segs:  seSeg,
			clips: seClips,
			assets: config.AssetsConfig{
				SE: map[string]config.SEEntry{
					"chime": {File: "/assets/chime.wav", Volume: 0.8, TrimSilenceThreshold: &threshold, Overlay: &overlayTrue},
				},
			},
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			ctx := BuildContext{
				Script:   model.Script{Segments: c.segs},
				Clips:    model.ClipsMeta{Clips: c.clips},
				ClipsDir: "/clips",
				Assets:   c.assets,
				PauseSec: 0.5,
				OutPath:  "/out.mp3",
			}
			args, err := BuildFFmpegArgs(ctx)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !strings.Contains(args.FilterComplex, "start_threshold=-40dB") {
				t.Errorf("filter_complex missing start_threshold=-40dB: %s", args.FilterComplex)
			}
		})
	}
}

// --- metadata args tests ---

func newMinimalContext() BuildContext {
	return BuildContext{
		Script: model.Script{
			Segments: []model.ScriptSegment{
				{Type: model.SegmentTypeSpeech, SpeakerRole: "host", Text: "テスト"},
			},
		},
		Clips: model.ClipsMeta{
			Clips: []model.ClipMeta{{Index: 0, File: "clip_000.wav", DurationSec: 1.0}},
		},
		ClipsDir: "/clips",
		Assets:   config.AssetsConfig{},
		PauseSec: 0.3,
		OutPath:  "/out.mp3",
	}
}

func hasOutputArg(outputArgs []string, needle string) bool {
	return strings.Contains(strings.Join(outputArgs, " "), needle)
}

func TestBuildFFmpegArgs_MetadataArgs_AllSet(t *testing.T) {
	ctx := newMinimalContext()
	ctx.Program = config.ProgramConfig{
		Title:    "テストラジオ",
		Author:   "テスト制作",
		Timezone: "Asia/Tokyo",
	}
	ctx.Meta = model.EpisodeMeta{
		Number:      5,
		Title:       "今週の技術",
		GeneratedAt: time.Date(2026, 6, 14, 12, 0, 0, 0, time.UTC),
	}

	args, err := BuildFFmpegArgs(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	out := strings.Join(args.OutputArgs, " ")
	if !strings.Contains(out, "-id3v2_version 3") {
		t.Errorf("OutputArgs missing -id3v2_version 3: %v", args.OutputArgs)
	}
	if !strings.Contains(out, "album=テストラジオ") {
		t.Errorf("OutputArgs missing album=テストラジオ: %v", args.OutputArgs)
	}
	if !strings.Contains(out, "title=第5回 今週の技術") {
		t.Errorf("OutputArgs missing title=第5回 今週の技術: %v", args.OutputArgs)
	}
	if !strings.Contains(out, "artist=テスト制作") {
		t.Errorf("OutputArgs missing artist=テスト制作: %v", args.OutputArgs)
	}
	if !strings.Contains(out, "track=5") {
		t.Errorf("OutputArgs missing track=5: %v", args.OutputArgs)
	}
	if !strings.Contains(out, "date=2026-06-14") {
		t.Errorf("OutputArgs missing date=2026-06-14: %v", args.OutputArgs)
	}
}

func TestBuildFFmpegArgs_MetadataArgs_NoMetadata(t *testing.T) {
	ctx := newMinimalContext()
	// Program and Meta are zero values

	args, err := BuildFFmpegArgs(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	out := strings.Join(args.OutputArgs, " ")
	if strings.Contains(out, "-id3v2_version") {
		t.Errorf("OutputArgs should not contain -id3v2_version when no metadata: %v", args.OutputArgs)
	}
	if strings.Contains(out, "-metadata") {
		t.Errorf("OutputArgs should not contain -metadata when all fields empty: %v", args.OutputArgs)
	}
}

func TestBuildFFmpegArgs_MetadataArgs_EmptyTitle(t *testing.T) {
	ctx := newMinimalContext()
	ctx.Program = config.ProgramConfig{Author: "テスト制作"}
	ctx.Meta = model.EpisodeMeta{Number: 3}

	args, err := BuildFFmpegArgs(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	out := strings.Join(args.OutputArgs, " ")
	if strings.Contains(out, "album=") {
		t.Errorf("OutputArgs should not contain album= when title is empty: %v", args.OutputArgs)
	}
}

func TestBuildFFmpegArgs_MetadataArgs_ZeroEpisodeNumber(t *testing.T) {
	ctx := newMinimalContext()
	ctx.Program = config.ProgramConfig{Title: "番組名"}
	ctx.Meta = model.EpisodeMeta{Number: 0}

	args, err := BuildFFmpegArgs(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	out := strings.Join(args.OutputArgs, " ")
	if strings.Contains(out, "track=") {
		t.Errorf("OutputArgs should not contain track= when episode number is 0: %v", args.OutputArgs)
	}
}

func TestBuildFFmpegArgs_MetadataArgs_ZeroGeneratedAt(t *testing.T) {
	ctx := newMinimalContext()
	ctx.Program = config.ProgramConfig{Title: "番組名"}
	// Meta.GeneratedAt is zero value

	args, err := BuildFFmpegArgs(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	out := strings.Join(args.OutputArgs, " ")
	if strings.Contains(out, "date=") {
		t.Errorf("OutputArgs should not contain date= when GeneratedAt is zero: %v", args.OutputArgs)
	}
}

func TestBuildFFmpegArgs_MetadataArgs_DateUsesTimezone(t *testing.T) {
	ctx := newMinimalContext()
	ctx.Program = config.ProgramConfig{
		Title:    "番組名",
		Timezone: "Asia/Tokyo", // UTC+9
	}
	// UTC 15:00 = JST 翌日 00:00 (still same day in JST), but UTC 23:00 = JST 翌日 08:00
	ctx.Meta = model.EpisodeMeta{
		Number:      1,
		GeneratedAt: time.Date(2026, 6, 14, 23, 0, 0, 0, time.UTC), // 2026-06-15 08:00 JST
	}

	args, err := BuildFFmpegArgs(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	out := strings.Join(args.OutputArgs, " ")
	if !strings.Contains(out, "date=2026-06-15") {
		t.Errorf("OutputArgs: date should be 2026-06-15 (JST), got: %v", args.OutputArgs)
	}
}
