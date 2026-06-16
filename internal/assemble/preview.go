package assemble

import (
	"context"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
	"strings"

	"github.com/canpok1/vox-radio/internal/config"
	"github.com/canpok1/vox-radio/internal/mediainfo"
)

const (
	dummyNarrationToneSec    = 2.0
	dummyNarrationSilSec     = 1.0
	dummyNarrationFreq       = 440
	dummyNarrationSampleRate = 44100
)

// PreviewContext holds the parameters for a single-asset preview.
type PreviewContext struct {
	AssetType    string // "jingle", "se", or "bgm"
	AssetKey     string // key in the assets map
	Assets       config.AssetsConfig
	OutPath      string
	MaxLengthSec float64 // maximum output duration in seconds; <= 0 disables truncation (outputs full length)
	// SourceDurationSec is the resolved duration of the source asset in seconds.
	// It is only used to size the dummy narration for ducking when truncation is
	// disabled (MaxLengthSec <= 0). Previewer.Run fills it via ffprobe when needed.
	SourceDurationSec float64
}

// Previewer executes a single-asset preview via ffmpeg.
type Previewer struct {
	runFFmpeg   func(ctx context.Context, args []string, w io.Writer) error
	getDuration func(path string) (float64, error)
}

// NewPreviewer creates a Previewer that calls ffmpeg.
func NewPreviewer() *Previewer {
	return &Previewer{runFFmpeg: runFFmpegCmd, getDuration: mediainfo.Duration}
}

// Run builds ffmpeg args for the preview context and executes ffmpeg.
func (p *Previewer) Run(ctx context.Context, pctx PreviewContext, w io.Writer) error {
	if err := os.MkdirAll(filepath.Dir(pctx.OutPath), 0o755); err != nil {
		return fmt.Errorf("create output dir: %w", err)
	}

	// When truncation is disabled, a ducking preview needs the source duration
	// to size the dummy narration over the full length of the BGM.
	if pctx.MaxLengthSec <= 0 && pctx.AssetType == "bgm" {
		if entry, ok := pctx.Assets.BGM[pctx.AssetKey]; ok && entry.DuckRatio > 0 {
			dur, err := p.getDuration(entry.File)
			if err != nil {
				return fmt.Errorf("probe bgm duration for ducking preview: %w", err)
			}
			pctx.SourceDurationSec = dur
		}
	}

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
	truncate := maxSec > 0

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

	if truncate && !trimApplied {
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
// Returns the final label, whether max-length trim was already applied, and any error.
// When truncation is disabled (maxSec <= 0), loop is ignored so the source plays
// once at its natural length instead of looping forever.
func buildBGMPreview(b *filterBuilder, ctx PreviewContext, maxSec float64) (string, bool, error) {
	entry, ok := ctx.Assets.BGM[ctx.AssetKey]
	if !ok {
		return "", false, fmt.Errorf("bgm %q not found in assets", ctx.AssetKey)
	}

	truncate := maxSec > 0
	loop := entry.Loop && truncate

	var bgmIdx int
	if loop {
		bgmIdx = b.addInput(entry.File, "-stream_loop", "-1")
	} else {
		bgmIdx = b.addInput(entry.File)
	}

	key := "preview"
	volLabel := "[preview_bgm_vol]"
	// For looping playback, chain atrim to stop the infinite loop at maxSec.
	atrimSuffix := ""
	if loop {
		atrimSuffix = fmt.Sprintf(",atrim=duration=%.3f", maxSec)
	}
	b.addFilter(fmt.Sprintf("[%d:a]volume=%.2f%s%s", bgmIdx, entry.Volume, atrimSuffix, volLabel))
	label := volLabel

	label = applyFadeIn(b, label, key, entry.EffectiveFadeIn())
	label = applyFadeOut(b, label, key, entry.EffectiveFadeOut())

	if entry.DuckRatio > 0 {
		// Size the dummy narration to the output length: maxSec when truncating,
		// otherwise the full source duration resolved by Previewer.Run.
		narrSec := maxSec
		if !truncate {
			narrSec = ctx.SourceDurationSec
		}
		narrLabel := buildDummyNarration(b, narrSec)
		duckedLabel := "[preview_ducked]"
		b.addFilter(fmt.Sprintf("%s%ssidechaincompress=threshold=0.02:ratio=%.1f%s",
			label, narrLabel, entry.DuckRatio, duckedLabel))
		label = duckedLabel
	}

	return label, loop, nil
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
