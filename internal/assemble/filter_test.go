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
		if inp == "/assets/bgm.mp3" {
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
		if inp == "/assets/eyecatch.wav" {
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
		if inp == "/assets/opening.wav" {
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
		if inp == "/assets/ending.wav" {
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
		if inp == "/assets/j1.wav" {
			foundJ1 = true
		}
		if inp == "/assets/j2.wav" {
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
		if strings.Contains(inp, "jingle") || strings.Contains(inp, "missing") {
			t.Errorf("unexpected jingle input when key is missing: %s", inp)
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
		if inp == "/assets/chime.wav" {
			chimeCount++
		}
	}
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
	if events[0].assetName != "chime" {
		t.Errorf("asset name: got %s, want chime", events[0].assetName)
	}
	// After clip_000 (2.0s) + pause (0.5s) = 2500ms
	wantMs := int((2.0 + 0.5) * 1000)
	if events[0].offsetMs != wantMs {
		t.Errorf("SE offset: got %d ms, want %d ms", events[0].offsetMs, wantMs)
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
