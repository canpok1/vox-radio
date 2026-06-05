package assemble

import (
	"context"
	"fmt"
	"io"
	"math"
	"strings"

	"github.com/canpok1/vox-radio/internal/config"
)

const (
	defaultPreviewMaxLengthSec = 30.0
	dummyNarrationToneSec      = 2.0
	dummyNarrationSilSec       = 1.0
	dummyNarrationFreq         = 440
	dummyNarrationSampleRate   = 44100
)

// PreviewContext holds the parameters for a single-asset preview.
type PreviewContext struct {
	AssetType    string // "jingle", "se", or "bgm"
	AssetKey     string // key in the assets map
	Assets       config.AssetsConfig
	OutPath      string
	MaxLengthSec float64 // maximum output duration; <= 0 uses defaultPreviewMaxLengthSec
}

// Previewer executes a single-asset preview via ffmpeg.
type Previewer struct {
	runFFmpeg func(ctx context.Context, args []string, w io.Writer) error
}

// NewPreviewer creates a Previewer that calls ffmpeg.
func NewPreviewer() *Previewer {
	return &Previewer{runFFmpeg: runFFmpegCmd}
}

// Run builds ffmpeg args for the preview context and executes ffmpeg.
func (p *Previewer) Run(ctx context.Context, pctx PreviewContext, w io.Writer) error {
	ffArgs, err := BuildPreviewFFmpegArgs(pctx)
	if err != nil {
		return err
	}
	args := buildCmdArgs(ffArgs)
	if w != nil {
		_, _ = fmt.Fprintf(w, "--- ffmpeg command ---\nffmpeg %s\n--- ffmpeg output ---\n", strings.Join(args, " "))
	}
	return p.runFFmpeg(ctx, args, w)
}

// BuildPreviewFFmpegArgs constructs ffmpeg arguments for a single-asset preview.
// Unlike BuildFFmpegArgs, loudnorm and alimiter are NOT applied.
func BuildPreviewFFmpegArgs(ctx PreviewContext) (*FFmpegArgs, error) {
	maxSec := ctx.MaxLengthSec
	if maxSec <= 0 {
		maxSec = defaultPreviewMaxLengthSec
	}

	b := &filterBuilder{}

	var finalLabel string
	var trimApplied bool
	var err error

	switch ctx.AssetType {
	case "jingle":
		finalLabel, err = buildJinglePreview(b, ctx)
	case "se":
		finalLabel, err = buildSEPreview(b, ctx)
	case "bgm":
		finalLabel, trimApplied, err = buildBGMPreview(b, ctx, maxSec)
	default:
		return nil, fmt.Errorf("unknown asset type %q: must be jingle, se, or bgm", ctx.AssetType)
	}
	if err != nil {
		return nil, err
	}

	if !trimApplied {
		trimLabel := "[preview_out]"
		b.addFilter(fmt.Sprintf("%satrim=duration=%.3f%s", finalLabel, maxSec, trimLabel))
		finalLabel = trimLabel
	}

	return &FFmpegArgs{
		Inputs:        b.inputs,
		FilterComplex: strings.Join(b.filters, ";"),
		OutputArgs:    []string{"-map", finalLabel, "-c:a", "libmp3lame", "-q:a", "2"},
		OutputPath:    ctx.OutPath,
	}, nil
}

func buildJinglePreview(b *filterBuilder, ctx PreviewContext) (string, error) {
	entry, ok := ctx.Assets.Jingle[ctx.AssetKey]
	if !ok {
		return "", fmt.Errorf("jingle %q not found in assets", ctx.AssetKey)
	}
	idx := b.addInput(entry.File)
	key := "preview"
	label := applySilenceTrim(b, fmt.Sprintf("[%d:a]", idx), key, entry.EffectiveTrimSilence(), entry.EffectiveTrimSilenceThresholdDB())
	label = applyFadeIn(b, label, key, entry.FadeIn)
	label = applyFadeOut(b, label, key, entry.FadeOut)
	return label, nil
}

func buildSEPreview(b *filterBuilder, ctx PreviewContext) (string, error) {
	entry, ok := ctx.Assets.SE[ctx.AssetKey]
	if !ok {
		return "", fmt.Errorf("se %q not found in assets", ctx.AssetKey)
	}
	idx := b.addInput(entry.File)
	key := "preview"
	label := applySilenceTrim(b, fmt.Sprintf("[%d:a]", idx), key, entry.EffectiveTrimSilence(), entry.EffectiveTrimSilenceThresholdDB())
	volLabel := "[preview_vol]"
	b.addFilter(fmt.Sprintf("%svolume=%.2f%s", label, entry.Volume, volLabel))
	return volLabel, nil
}

// buildBGMPreview builds the filter chain for a BGM asset preview.
// Returns the final label, whether max-length trim was already applied (loop=true), and any error.
func buildBGMPreview(b *filterBuilder, ctx PreviewContext, maxSec float64) (string, bool, error) {
	entry, ok := ctx.Assets.BGM[ctx.AssetKey]
	if !ok {
		return "", false, fmt.Errorf("bgm %q not found in assets", ctx.AssetKey)
	}

	var bgmIdx int
	if entry.Loop {
		bgmIdx = b.addInput(entry.File, "-stream_loop", "-1")
	} else {
		bgmIdx = b.addInput(entry.File)
	}

	key := "preview"
	var label string
	if entry.Loop {
		// Apply volume and atrim together to stop the infinite loop at maxSec.
		trimLabel := "[preview_bgm_trim]"
		b.addFilter(fmt.Sprintf("[%d:a]volume=%.2f,atrim=duration=%.3f%s", bgmIdx, entry.Volume, maxSec, trimLabel))
		label = trimLabel
	} else {
		volLabel := "[preview_bgm_vol]"
		b.addFilter(fmt.Sprintf("[%d:a]volume=%.2f%s", bgmIdx, entry.Volume, volLabel))
		label = volLabel
	}

	label = applyFadeIn(b, label, key, entry.EffectiveFadeIn())
	label = applyFadeOut(b, label, key, entry.EffectiveFadeOut())

	if entry.DuckRatio > 0 {
		narrLabel := buildDummyNarration(b, maxSec)
		duckedLabel := "[preview_ducked]"
		b.addFilter(fmt.Sprintf("%s%ssidechaincompress=threshold=0.02:ratio=%.1f%s",
			label, narrLabel, entry.DuckRatio, duckedLabel))
		label = duckedLabel
	}

	return label, entry.Loop, nil
}

// buildDummyNarration generates a repeating tone/silence pattern as a sidechain signal for ducking.
// The pattern alternates dummyNarrationToneSec-second tones with dummyNarrationSilSec-second silence.
func buildDummyNarration(b *filterBuilder, durationSec float64) string {
	const cycleSec = dummyNarrationToneSec + dummyNarrationSilSec
	numCycles := int(math.Ceil(durationSec/cycleSec)) + 1

	parts := make([]string, 0, numCycles*2)
	for i := 0; i < numCycles; i++ {
		toneLabel := fmt.Sprintf("[dummy_tone%d]", i)
		silLabel := fmt.Sprintf("[dummy_sil%d]", i)
		b.addFilter(fmt.Sprintf(
			"sine=frequency=%d:sample_rate=%d,atrim=duration=%.3f%s",
			dummyNarrationFreq, dummyNarrationSampleRate, dummyNarrationToneSec, toneLabel,
		))
		b.addFilter(fmt.Sprintf(
			"anullsrc=cl=mono:r=%d,atrim=duration=%.3f%s",
			dummyNarrationSampleRate, dummyNarrationSilSec, silLabel,
		))
		parts = append(parts, toneLabel, silLabel)
	}

	narrRawLabel := "[dummy_narr_raw]"
	b.addFilter(fmt.Sprintf("%sconcat=n=%d:v=0:a=1%s", strings.Join(parts, ""), len(parts), narrRawLabel))

	narrLabel := "[dummy_narr]"
	b.addFilter(fmt.Sprintf("%satrim=duration=%.3f%s", narrRawLabel, durationSec, narrLabel))

	return narrLabel
}
