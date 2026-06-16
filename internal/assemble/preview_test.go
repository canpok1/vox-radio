package assemble

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
				File:   "/audio/bgm.mp3",
				Volume: 0.5,
				Loop:   true,
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

func TestBuildPreviewFFmpegArgs_DefaultMaxLengthSec_AppliesDefaultTrim(t *testing.T) {
	assets := config.AssetsConfig{
		SE: map[string]config.SEEntry{
			"chime": {File: "/audio/chime.wav", Volume: 0.5},
		},
	}
	ctx := PreviewContext{
		AssetType:    "se",
		AssetKey:     "chime",
		Assets:       assets,
		OutPath:      "/out.mp3",
		MaxLengthSec: 0, // use default (30s)
	}

	args, err := BuildPreviewFFmpegArgs(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(args.FilterComplex, "atrim=duration=30.000") {
		t.Errorf("expected atrim=duration=30.000 for default max length, got: %s", args.FilterComplex)
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
