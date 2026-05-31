package assemble

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/canpok1/vox-radio/internal/config"
	"github.com/canpok1/vox-radio/internal/model"
)

const (
	// outputNormFilter normalizes the assembled output to EBU R128 before peak limiting.
	// Applied once to the full mix (speech + SE + BGM + jingles) after serial concat.
	outputNormFilter = "loudnorm=I=-16:TP=-1.5:LRA=11"
	// outputLimiterLimit is the linear equivalent of the TP ceiling used in outputNormFilter (10^(-1.5/20) ≈ 0.841).
	outputLimiterLimit = 0.841
)

// BuildContext holds all data needed to build the ffmpeg command.
type BuildContext struct {
	Script        model.Script
	Clips         model.ClipsMeta
	ClipsDir      string
	Assets        config.AssetsConfig
	PauseSec      float64
	OutPath       string
	OpeningJingle string // key into Assets.Jingle; empty = no OP jingle
	EndingJingle  string // key into Assets.Jingle; empty = no ED jingle
}

// FFmpegArgs holds the complete set of arguments for an ffmpeg call.
type FFmpegArgs struct {
	Inputs        []string // input file paths
	FilterComplex string   // -filter_complex value
	OutputArgs    []string // output options (map, codec, etc.)
	OutputPath    string
}

type filterBuilder struct {
	inputs  []string
	filters []string
	nextIdx int
}

func (b *filterBuilder) addInput(path string) int {
	idx := b.nextIdx
	b.inputs = append(b.inputs, path)
	b.nextIdx++
	return idx
}

func (b *filterBuilder) addFilter(f string) {
	b.filters = append(b.filters, f)
}

// addSilence emits an anullsrc silence segment with the given label.
func (b *filterBuilder) addSilence(durationSec float64, label string) {
	b.addFilter(fmt.Sprintf("anullsrc=cl=stereo:r=44100,atrim=duration=%.3f%s", durationSec, label))
}

type seEvent struct {
	seName   string
	offsetMs int
}

// BuildFFmpegArgs constructs the complete ffmpeg argument list for audio assembly.
// It builds a single filter_complex covering speech concat, SE placement, BGM ducking,
// OP/ED jingles (serial concat), and loudness normalization.
func BuildFFmpegArgs(bctx BuildContext) (*FFmpegArgs, error) {
	if len(bctx.Clips.Clips) == 0 {
		return nil, fmt.Errorf("no speech clips to assemble")
	}

	b := &filterBuilder{}

	// Add clip inputs first (indices 0..N-1)
	clipInputIdx := make([]int, len(bctx.Clips.Clips))
	for i, clip := range bctx.Clips.Clips {
		clipInputIdx[i] = b.addInput(filepath.Join(bctx.ClipsDir, clip.File))
	}

	// Build speech track (concat with silence between clips)
	speechLabel := buildSpeechConcat(b, bctx.Clips.Clips, clipInputIdx, bctx.PauseSec)
	currentLabel := speechLabel

	// If BGM is configured, split speech for sidechain ducking.
	hasBGM := len(bctx.Assets.BGM) > 0
	if hasBGM {
		b.addFilter(fmt.Sprintf("%sasplit=2[speech_mix][speech_duck]", currentLabel))
		currentLabel = "[speech_mix]"
	}

	// SE placement: add each SE as a separate input (even if the same file is used
	// multiple times) so that no stream label is reused in filter_complex.
	events := computeSEEvents(bctx.Script, bctx.Clips.Clips, bctx.PauseSec)
	for i, ev := range events {
		entry, ok := bctx.Assets.SE[ev.seName]
		if !ok {
			continue
		}
		seIdx := b.addInput(entry.File)
		delayedLabel := fmt.Sprintf("[se%d]", i)
		nextLabel := fmt.Sprintf("[after_se%d]", i)
		b.addFilter(fmt.Sprintf("[%d:a]volume=%.2f,adelay=%d|%d%s",
			seIdx, entry.Volume, ev.offsetMs, ev.offsetMs, delayedLabel))
		b.addFilter(fmt.Sprintf("%s%samix=inputs=2:duration=first:normalize=0%s",
			currentLabel, delayedLabel, nextLabel))
		currentLabel = nextLabel
	}

	// BGM: aloop + volume + sidechaincompress ducking
	if hasBGM {
		var bgmEntry config.BGMEntry
		for _, e := range bctx.Assets.BGM {
			bgmEntry = e
			break
		}
		bgmIdx := b.addInput(bgmEntry.File)
		bgmLabel := fmt.Sprintf("[bgm%d_vol]", bgmIdx)
		if bgmEntry.Loop {
			b.addFilter(fmt.Sprintf("[%d:a]aloop=loop=-1:size=999999,volume=%.2f%s",
				bgmIdx, bgmEntry.Volume, bgmLabel))
		} else {
			b.addFilter(fmt.Sprintf("[%d:a]volume=%.2f%s", bgmIdx, bgmEntry.Volume, bgmLabel))
		}
		if bgmEntry.DuckRatio > 0 {
			b.addFilter(fmt.Sprintf("%s[speech_duck]sidechaincompress=threshold=0.02:ratio=%.1f[bgm_ducked]",
				bgmLabel, bgmEntry.DuckRatio))
			b.addFilter(fmt.Sprintf("%s[bgm_ducked]amix=inputs=2:duration=first:normalize=0[after_bgm]",
				currentLabel))
		} else {
			b.addFilter(fmt.Sprintf("%s%samix=inputs=2:duration=first:normalize=0[after_bgm]",
				currentLabel, bgmLabel))
		}
		currentLabel = "[after_bgm]"
	}

	// Build serial jingle concat: [OP jingle][pause][main content][pause][ED jingle]
	// Each jingle is a distinct segment played before/after main content (not overlaid).
	currentLabel = buildJingleConcat(b, bctx, currentLabel)

	// loudnorm: applied once to the full assembled mix (speech + SE + BGM + jingles).
	b.addFilter(fmt.Sprintf("%s%s[norm_out]", currentLabel, outputNormFilter))

	// Peak limiter: prevents clipping after loudnorm. level=0 disables auto gain equalization.
	b.addFilter(fmt.Sprintf("[norm_out]alimiter=limit=%.3f:level=0[out]", outputLimiterLimit))

	return &FFmpegArgs{
		Inputs:        b.inputs,
		FilterComplex: strings.Join(b.filters, ";"),
		OutputArgs:    []string{"-map", "[out]", "-c:a", "libmp3lame", "-q:a", "2"},
		OutputPath:    bctx.OutPath,
	}, nil
}

// buildJingleConcat inserts OP/ED jingles around mainLabel using ffmpeg concat.
// Returns the label of the final stream (either a new [full_mix] label or mainLabel unchanged).
func buildJingleConcat(b *filterBuilder, bctx BuildContext, mainLabel string) string {
	var opEntry config.JingleEntry
	var hasOP bool
	if bctx.OpeningJingle != "" {
		opEntry, hasOP = bctx.Assets.Jingle[bctx.OpeningJingle]
	}
	var edEntry config.JingleEntry
	var hasED bool
	if bctx.EndingJingle != "" {
		edEntry, hasED = bctx.Assets.Jingle[bctx.EndingJingle]
	}
	if !hasOP && !hasED {
		return mainLabel
	}

	var parts []string

	if hasOP {
		opIdx := b.addInput(opEntry.File)
		opLabel := buildJingleFadeIn(b, opIdx, opEntry)
		b.addSilence(bctx.PauseSec, "[pause_op]")
		parts = append(parts, opLabel, "[pause_op]")
	}

	parts = append(parts, mainLabel)

	if hasED {
		b.addSilence(bctx.PauseSec, "[pause_ed]")
		edIdx := b.addInput(edEntry.File)
		edLabel := buildJingleFadeOut(b, edIdx, edEntry)
		parts = append(parts, "[pause_ed]", edLabel)
	}

	b.addFilter(fmt.Sprintf("%sconcat=n=%d:v=0:a=1[full_mix]", strings.Join(parts, ""), len(parts)))
	return "[full_mix]"
}

// buildJingleFadeIn applies fade-in to a jingle input and returns the resulting label.
func buildJingleFadeIn(b *filterBuilder, idx int, entry config.JingleEntry) string {
	rawLabel := fmt.Sprintf("[%d:a]", idx)
	return applyFadeIn(b, rawLabel, idx, entry.FadeIn)
}

// buildJingleFadeOut applies fade-out (and optional fade-in) to a jingle input and returns the resulting label.
func buildJingleFadeOut(b *filterBuilder, idx int, entry config.JingleEntry) string {
	label := fmt.Sprintf("[%d:a]", idx)
	if entry.FadeOut > 0 {
		// Reverse → fade in → reverse = fade out from end
		fadedLabel := fmt.Sprintf("[jingle%d_fo]", idx)
		b.addFilter(fmt.Sprintf("%sareverse,afade=t=in:d=%.3f,areverse%s", label, entry.FadeOut, fadedLabel))
		label = fadedLabel
	}
	return applyFadeIn(b, label, idx, entry.FadeIn)
}

// applyFadeIn applies an afade=t=in filter to currentLabel and returns the output label.
// Returns currentLabel unchanged when fadeSec <= 0.
func applyFadeIn(b *filterBuilder, currentLabel string, idx int, fadeSec float64) string {
	if fadeSec <= 0 {
		return currentLabel
	}
	outLabel := fmt.Sprintf("[jingle%d_fi]", idx)
	b.addFilter(fmt.Sprintf("%safade=t=in:d=%.3f%s", currentLabel, fadeSec, outLabel))
	return outLabel
}

// buildSpeechConcat generates filter entries for concatenating clips with silence between them.
func buildSpeechConcat(b *filterBuilder, clips []model.ClipMeta, inputIdx []int, pauseSec float64) string {
	if len(clips) == 1 {
		return fmt.Sprintf("[%d:a]", inputIdx[0])
	}

	var parts []string
	for i := range clips {
		parts = append(parts, fmt.Sprintf("[%d:a]", inputIdx[i]))
		if i < len(clips)-1 {
			b.addSilence(pauseSec, fmt.Sprintf("[p%d]", i))
			parts = append(parts, fmt.Sprintf("[p%d]", i))
		}
	}
	n := 2*len(clips) - 1
	b.addFilter(fmt.Sprintf("%sconcat=n=%d:v=0:a=1[speech]", strings.Join(parts, ""), n))
	return "[speech]"
}

// computeSEEvents processes the script to determine the timeline offset of each SE segment.
// The offset is measured in milliseconds from the start of the assembled audio.
func computeSEEvents(script model.Script, clips []model.ClipMeta, pauseSec float64) []seEvent {
	var events []seEvent
	clipIdx := 0
	offsetMs := 0.0

	for _, seg := range script.Segments {
		switch seg.Type {
		case model.SegmentTypeSpeech:
			if clipIdx < len(clips) {
				offsetMs += clips[clipIdx].DurationSec * 1000
				clipIdx++
			}
			offsetMs += pauseSec * 1000
		case model.SegmentTypeSE:
			events = append(events, seEvent{
				seName:   seg.SEName,
				offsetMs: int(offsetMs),
			})
		}
	}
	return events
}
