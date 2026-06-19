package mix

import (
	"context"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/canpok1/vox-radio/internal/config"
	"github.com/canpok1/vox-radio/internal/testutil"
)

const testMaxLengthSec = 10.0

func TestBuildPreviewFFmpegArgs_Jingle_ContainsExpectedFilters(t *testing.T) {
	assets := config.AssetsConfig{
		Jingle: map[string]config.JingleEntry{
			"opening": {
				File:    "/audio/opening.mp3",
				FadeIn:  0.5,
				FadeOut: 0.5,
			},
		},
	}
	ctx := PreviewContext{
		AssetType:    "jingle",
		AssetKey:     "opening",
		Assets:       assets,
		OutPath:      "/out.mp3",
		MaxLengthSec: testMaxLengthSec,
	}

	args, err := BuildPreviewFFmpegArgs(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(args.Inputs) < 1 || args.Inputs[0].Path != "/audio/opening.mp3" {
		t.Errorf("expected input /audio/opening.mp3, got: %v", args.Inputs)
	}

	fc := args.FilterComplex
	if !strings.Contains(fc, "silenceremove") {
		t.Errorf("expected silenceremove in filter_complex (trim_silence defaults to true), got: %s", fc)
	}
	if !strings.Contains(fc, "afade=t=in") {
		t.Errorf("expected afade=t=in in filter_complex, got: %s", fc)
	}
	if !strings.Contains(fc, "areverse") {
		t.Errorf("expected areverse in filter_complex (fade_out), got: %s", fc)
	}
	if !strings.Contains(fc, "atrim=duration=") {
		t.Errorf("expected atrim=duration= in filter_complex, got: %s", fc)
	}
	if strings.Contains(fc, "loudnorm") {
		t.Errorf("filter_complex must not contain loudnorm, got: %s", fc)
	}
	if strings.Contains(fc, "alimiter") {
		t.Errorf("filter_complex must not contain alimiter, got: %s", fc)
	}

	if args.OutputPath != "/out.mp3" {
		t.Errorf("output path: got %s, want /out.mp3", args.OutputPath)
	}
	hasMP3Codec := false
	for i, arg := range args.OutputArgs {
		if arg == "-c:a" && i+1 < len(args.OutputArgs) && args.OutputArgs[i+1] == "libmp3lame" {
			hasMP3Codec = true
		}
	}
	if !hasMP3Codec {
		t.Errorf("expected -c:a libmp3lame in output args, got: %v", args.OutputArgs)
	}
}

func TestBuildPreviewFFmpegArgs_SE_ContainsExpectedFilters(t *testing.T) {
	assets := config.AssetsConfig{
		SE: map[string]config.SEEntry{
			"chime": {
				File:   "/audio/chime.wav",
				Volume: 0.8,
			},
		},
	}
	ctx := PreviewContext{
		AssetType:    "se",
		AssetKey:     "chime",
		Assets:       assets,
		OutPath:      "/out.mp3",
		MaxLengthSec: testMaxLengthSec,
	}

	args, err := BuildPreviewFFmpegArgs(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	fc := args.FilterComplex
	if !strings.Contains(fc, "silenceremove") {
		t.Errorf("expected silenceremove in filter_complex (trim_silence defaults to true), got: %s", fc)
	}
	if !strings.Contains(fc, "volume=0.80") {
		t.Errorf("expected volume=0.80 in filter_complex, got: %s", fc)
	}
	if strings.Contains(fc, "loudnorm") {
		t.Errorf("filter_complex must not contain loudnorm, got: %s", fc)
	}
}

func TestBuildPreviewFFmpegArgs_BGM_ContainsExpectedFilters(t *testing.T) {
	assets := config.AssetsConfig{
		BGM: map[string]config.BGMEntry{
			"talk": {
				File:    "/audio/talk.mp3",
				Volume:  0.3,
				FadeIn:  testutil.Ptr(1.0),
				FadeOut: testutil.Ptr(1.0),
			},
		},
	}
	ctx := PreviewContext{
		AssetType:    "bgm",
		AssetKey:     "talk",
		Assets:       assets,
		OutPath:      "/out.mp3",
		MaxLengthSec: testMaxLengthSec,
	}

	args, err := BuildPreviewFFmpegArgs(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	fc := args.FilterComplex
	if !strings.Contains(fc, "volume=0.30") {
		t.Errorf("expected volume=0.30 in filter_complex, got: %s", fc)
	}
	if !strings.Contains(fc, "afade=t=in") {
		t.Errorf("expected afade=t=in in filter_complex, got: %s", fc)
	}
	if !strings.Contains(fc, "areverse") {
		t.Errorf("expected areverse in filter_complex (fade_out), got: %s", fc)
	}
	if !strings.Contains(fc, "atrim=duration=") {
		t.Errorf("expected atrim=duration= in filter_complex, got: %s", fc)
	}
	if strings.Contains(fc, "loudnorm") {
		t.Errorf("filter_complex must not contain loudnorm, got: %s", fc)
	}
}

func TestBuildPreviewFFmpegArgs_BGM_LoopTrue_UsesStreamLoop(t *testing.T) {
	assets := config.AssetsConfig{
		BGM: map[string]config.BGMEntry{
			"bgm": {
				File:        "/audio/bgm.mp3",
				Volume:      0.5,
				Loop:        true,
				TrimSilence: testutil.Ptr(false), // legacy -stream_loop path (trim/gap disabled)
			},
		},
	}
	ctx := PreviewContext{
		AssetType:    "bgm",
		AssetKey:     "bgm",
		Assets:       assets,
		OutPath:      "/out.mp3",
		MaxLengthSec: testMaxLengthSec,
	}

	args, err := BuildPreviewFFmpegArgs(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !hasStreamLoop(args.Inputs, "/audio/bgm.mp3") {
		t.Errorf("expected -stream_loop -1 for loop=true BGM, inputs: %v", args.Inputs)
	}
	if !strings.Contains(args.FilterComplex, "atrim=duration=") {
		t.Errorf("expected atrim=duration= in filter_complex for loop BGM, got: %s", args.FilterComplex)
	}
}

// TestBuildPreviewFFmpegArgs_BGM_LoopDefaultTrim_UsesAloop verifies the preview mirrors the
// production loop mechanism: loop=true with default trim_silence trims silence and loops via
// aloop (not -stream_loop), and still caps the output at maxSec via atrim.
func TestBuildPreviewFFmpegArgs_BGM_LoopDefaultTrim_UsesAloop(t *testing.T) {
	assets := config.AssetsConfig{
		BGM: map[string]config.BGMEntry{
			"bgm": {File: "/audio/bgm.mp3", Volume: 0.5, Loop: true},
		},
	}
	ctx := PreviewContext{
		AssetType:    "bgm",
		AssetKey:     "bgm",
		Assets:       assets,
		OutPath:      "/out.mp3",
		MaxLengthSec: testMaxLengthSec,
	}
	args, err := BuildPreviewFFmpegArgs(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if hasStreamLoop(args.Inputs, "/audio/bgm.mp3") {
		t.Errorf("loop:true BGM with default trim must NOT use -stream_loop, inputs: %v", args.Inputs)
	}
	if !strings.Contains(args.FilterComplex, "silenceremove") {
		t.Errorf("expected silenceremove for default trim_silence, filter: %s", args.FilterComplex)
	}
	if !strings.Contains(args.FilterComplex, "aloop=loop=-1") {
		t.Errorf("expected aloop=loop=-1 for filter-graph looping, filter: %s", args.FilterComplex)
	}
	if !strings.Contains(args.FilterComplex, "atrim=duration=") {
		t.Errorf("expected atrim=duration= to cap preview length, filter: %s", args.FilterComplex)
	}
}

func TestBuildPreviewFFmpegArgs_BGM_DuckRatio_AddsSidechainCompress(t *testing.T) {
	assets := config.AssetsConfig{
		BGM: map[string]config.BGMEntry{
			"bgm": {
				File:      "/audio/bgm.mp3",
				Volume:    0.5,
				DuckRatio: 8.0,
			},
		},
	}
	ctx := PreviewContext{
		AssetType:    "bgm",
		AssetKey:     "bgm",
		Assets:       assets,
		OutPath:      "/out.mp3",
		MaxLengthSec: testMaxLengthSec,
	}

	args, err := BuildPreviewFFmpegArgs(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	fc := args.FilterComplex
	if !strings.Contains(fc, "sidechaincompress") {
		t.Errorf("expected sidechaincompress in filter_complex for duck_ratio>0, got: %s", fc)
	}
	if !strings.Contains(fc, "ratio=8.0") {
		t.Errorf("expected ratio=8.0 in sidechaincompress, got: %s", fc)
	}
	if !strings.Contains(fc, "threshold=0.02") {
		t.Errorf("expected threshold=0.02 in sidechaincompress, got: %s", fc)
	}
	if !strings.Contains(fc, "sine") {
		t.Errorf("expected sine in filter_complex for dummy narration, got: %s", fc)
	}
}

func TestBuildPreviewFFmpegArgs_BGM_NoDuckRatio_NoSidechain(t *testing.T) {
	assets := config.AssetsConfig{
		BGM: map[string]config.BGMEntry{
			"bgm": {
				File:      "/audio/bgm.mp3",
				Volume:    0.5,
				DuckRatio: 0.0,
			},
		},
	}
	ctx := PreviewContext{
		AssetType:    "bgm",
		AssetKey:     "bgm",
		Assets:       assets,
		OutPath:      "/out.mp3",
		MaxLengthSec: testMaxLengthSec,
	}

	args, err := BuildPreviewFFmpegArgs(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if strings.Contains(args.FilterComplex, "sidechaincompress") {
		t.Errorf("sidechaincompress must not be present when duck_ratio=0, got: %s", args.FilterComplex)
	}
}

func TestBuildPreviewFFmpegArgs_DefaultMaxLength_NoTrim(t *testing.T) {
	tests := []struct {
		name      string
		assetType string
		assetKey  string
		assets    config.AssetsConfig
	}{
		{
			name:      "jingle",
			assetType: "jingle",
			assetKey:  "opening",
			assets: config.AssetsConfig{
				Jingle: map[string]config.JingleEntry{"opening": {File: "/audio/opening.mp3"}},
			},
		},
		{
			name:      "se",
			assetType: "se",
			assetKey:  "chime",
			assets: config.AssetsConfig{
				SE: map[string]config.SEEntry{"chime": {File: "/audio/chime.wav", Volume: 0.5}},
			},
		},
		{
			name:      "bgm",
			assetType: "bgm",
			assetKey:  "talk",
			assets: config.AssetsConfig{
				BGM: map[string]config.BGMEntry{"talk": {File: "/audio/talk.mp3", Volume: 0.3}},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := PreviewContext{
				AssetType:    tt.assetType,
				AssetKey:     tt.assetKey,
				Assets:       tt.assets,
				OutPath:      "/out.mp3",
				MaxLengthSec: 0, // default: truncation disabled
			}

			args, err := BuildPreviewFFmpegArgs(ctx)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if strings.Contains(args.FilterComplex, "atrim=duration=") {
				t.Errorf("expected no atrim when truncation disabled, got: %s", args.FilterComplex)
			}
		})
	}
}

func TestBuildPreviewFFmpegArgs_DefaultMaxLength_LoopBGM_NoLoopNoTrim(t *testing.T) {
	assets := config.AssetsConfig{
		BGM: map[string]config.BGMEntry{
			"bgm": {File: "/audio/bgm.mp3", Volume: 0.5, Loop: true},
		},
	}
	ctx := PreviewContext{
		AssetType:    "bgm",
		AssetKey:     "bgm",
		Assets:       assets,
		OutPath:      "/out.mp3",
		MaxLengthSec: 0, // default: truncation disabled
	}

	args, err := BuildPreviewFFmpegArgs(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if hasStreamLoop(args.Inputs, "/audio/bgm.mp3") {
		t.Errorf("expected no -stream_loop for loop BGM when truncation disabled, inputs: %v", args.Inputs)
	}
	if strings.Contains(args.FilterComplex, "atrim=duration=") {
		t.Errorf("expected no atrim for loop BGM when truncation disabled, got: %s", args.FilterComplex)
	}
}

func TestBuildPreviewFFmpegArgs_DefaultMaxLength_Duck_UsesSourceDuration(t *testing.T) {
	assets := config.AssetsConfig{
		BGM: map[string]config.BGMEntry{
			"bgm": {File: "/audio/bgm.mp3", Volume: 0.5, DuckRatio: 8.0},
		},
	}
	ctx := PreviewContext{
		AssetType:         "bgm",
		AssetKey:          "bgm",
		Assets:            assets,
		OutPath:           "/out.mp3",
		MaxLengthSec:      0, // default: truncation disabled
		SourceDurationSec: 45.0,
	}

	args, err := BuildPreviewFFmpegArgs(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	fc := args.FilterComplex
	if !strings.Contains(fc, "sidechaincompress") {
		t.Errorf("expected sidechaincompress for duck_ratio>0, got: %s", fc)
	}
	// The dummy narration is trimmed to the source duration (45s).
	if !strings.Contains(fc, "atrim=duration=45.000") {
		t.Errorf("expected dummy narration trimmed to source duration 45.000, got: %s", fc)
	}
}

func TestPreviewer_Run_DefaultMaxLength_Duck_ProbesSourceDuration(t *testing.T) {
	dir := t.TempDir()
	outPath := dir + "/out.mp3"

	assets := config.AssetsConfig{
		BGM: map[string]config.BGMEntry{
			"bgm": {File: "/audio/bgm.mp3", Volume: 0.5, DuckRatio: 8.0},
		},
	}

	var probedPath string
	var capturedArgs []string
	p := &Previewer{
		runFFmpeg: func(_ context.Context, args []string, _ io.Writer) error {
			capturedArgs = args
			return nil
		},
		getDuration: func(path string) (float64, error) {
			probedPath = path
			return 45.0, nil
		},
	}

	pctx := PreviewContext{
		AssetType:    "bgm",
		AssetKey:     "bgm",
		Assets:       assets,
		OutPath:      outPath,
		MaxLengthSec: 0, // default: truncation disabled
	}

	if err := p.Run(context.Background(), pctx, nil); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if probedPath != "/audio/bgm.mp3" {
		t.Errorf("expected ffprobe on /audio/bgm.mp3, got: %q", probedPath)
	}
	joined := strings.Join(capturedArgs, " ")
	if !strings.Contains(joined, "atrim=duration=45.000") {
		t.Errorf("expected dummy narration trimmed to probed duration 45.000, got: %v", capturedArgs)
	}
}

func TestBuildPreviewFFmpegArgs_UnknownType_ReturnsError(t *testing.T) {
	ctx := PreviewContext{
		AssetType:    "unknown",
		AssetKey:     "foo",
		Assets:       config.AssetsConfig{},
		OutPath:      "/out.mp3",
		MaxLengthSec: testMaxLengthSec,
	}

	_, err := BuildPreviewFFmpegArgs(ctx)
	if err == nil {
		t.Error("expected error for unknown asset type, got nil")
	}
}

func TestBuildPreviewFFmpegArgs_UnknownKey_ReturnsError(t *testing.T) {
	tests := []struct {
		name      string
		assetType string
		assets    config.AssetsConfig
	}{
		{
			name:      "jingle",
			assetType: "jingle",
			assets:    config.AssetsConfig{Jingle: map[string]config.JingleEntry{}},
		},
		{
			name:      "se",
			assetType: "se",
			assets:    config.AssetsConfig{SE: map[string]config.SEEntry{}},
		},
		{
			name:      "bgm",
			assetType: "bgm",
			assets:    config.AssetsConfig{BGM: map[string]config.BGMEntry{}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := PreviewContext{
				AssetType:    tt.assetType,
				AssetKey:     "nonexistent",
				Assets:       tt.assets,
				OutPath:      "/out.mp3",
				MaxLengthSec: testMaxLengthSec,
			}
			_, err := BuildPreviewFFmpegArgs(ctx)
			if err == nil {
				t.Errorf("expected error for unknown key in %s, got nil", tt.assetType)
			}
		})
	}
}

func TestPreviewer_Run_InvokesFFmpegWithFilterComplex(t *testing.T) {
	dir := t.TempDir()
	outPath := dir + "/out.mp3"

	assets := config.AssetsConfig{
		SE: map[string]config.SEEntry{
			"chime": {File: "/audio/chime.wav", Volume: 0.8},
		},
	}

	var capturedArgs []string
	p := &Previewer{
		runFFmpeg: func(_ context.Context, args []string, _ io.Writer) error {
			capturedArgs = args
			return nil
		},
	}

	pctx := PreviewContext{
		AssetType:    "se",
		AssetKey:     "chime",
		Assets:       assets,
		OutPath:      outPath,
		MaxLengthSec: testMaxLengthSec,
	}

	err := p.Run(context.Background(), pctx, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	hasFilterComplex := false
	for _, arg := range capturedArgs {
		if arg == "-filter_complex" {
			hasFilterComplex = true
		}
	}
	if !hasFilterComplex {
		t.Errorf("expected -filter_complex in ffmpeg args, got: %v", capturedArgs)
	}

	if len(capturedArgs) == 0 || capturedArgs[len(capturedArgs)-1] != outPath {
		t.Errorf("expected %s as last ffmpeg arg, got: %v", outPath, capturedArgs)
	}
}

func TestPreviewer_Run_CreatesOutputDirectory(t *testing.T) {
	dir := t.TempDir()
	outPath := dir + "/subdir/out.mp3"

	assets := config.AssetsConfig{
		SE: map[string]config.SEEntry{
			"chime": {File: "/audio/chime.wav", Volume: 0.5},
		},
	}

	p := &Previewer{
		runFFmpeg: func(_ context.Context, _ []string, _ io.Writer) error { return nil },
	}

	pctx := PreviewContext{
		AssetType:    "se",
		AssetKey:     "chime",
		Assets:       assets,
		OutPath:      outPath,
		MaxLengthSec: testMaxLengthSec,
	}

	err := p.Run(context.Background(), pctx, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if _, statErr := os.Stat(dir + "/subdir"); os.IsNotExist(statErr) {
		t.Errorf("expected output directory to be created, but it does not exist")
	}
}
