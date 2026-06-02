package assemble

import (
	"fmt"
	"path/filepath"
	"slices"
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
	// silenceTrimThreshold is the amplitude below which audio is treated as silence
	// when stripping leading/trailing silence from jingle/SE inputs.
	silenceTrimThreshold = "-50dB"
)

// BuildContext holds all data needed to build the ffmpeg command.
// Jingles are placed as SegmentTypeJingle in Script; OP/ED jingles are injected
// into Script by the Assembler before calling BuildFFmpegArgs.
type BuildContext struct {
	Script   model.Script
	Clips    model.ClipsMeta
	ClipsDir string
	Assets   config.AssetsConfig
	PauseSec float64
	OutPath  string
}

// FFmpegInput holds a single ffmpeg input file with optional pre-input options.
type FFmpegInput struct {
	Path       string   // input file path
	PreOptions []string // options placed before -i (e.g., "-stream_loop", "-1")
}

// FFmpegArgs holds the complete set of arguments for an ffmpeg call.
type FFmpegArgs struct {
	Inputs        []FFmpegInput // input files with optional pre-options
	FilterComplex string        // -filter_complex value
	OutputArgs    []string      // output options (map, codec, etc.)
	OutputPath    string
}

type filterBuilder struct {
	inputs  []FFmpegInput
	filters []string
	nextIdx int
}

func (b *filterBuilder) addInput(path string, preOpts ...string) int {
	idx := b.nextIdx
	b.inputs = append(b.inputs, FFmpegInput{Path: path, PreOptions: preOpts})
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
	assetName string
	offsetMs  int
}

// speechItem represents an element in the speech timeline: either a clip or an explicit pause.
type speechItem struct {
	clipIndex   int     // >= 0 for a clip, -1 for an explicit pause
	durationSec float64 // used when clipIndex == -1
}

// bgmInterval represents a BGM active period within a run.
type bgmInterval struct {
	startMs   int
	endMs     int // -1 = to end of run
	assetName string
}

// runData holds all audio segments for a single "run" (the content between jingles).
type runData struct {
	speechItems  []speechItem
	seEvents     []seEvent
	bgmIntervals []bgmInterval
	durationMs   int
}

// BuildFFmpegArgs constructs the complete ffmpeg argument list for audio assembly.
// The script is scanned for jingle segments which act as run boundaries.
// Each run (speech + SE + BGM) is assembled independently, then all runs and jingles
// are concatenated in order. loudnorm is applied once to the full output.
func BuildFFmpegArgs(bctx BuildContext) (*FFmpegArgs, error) {
	if len(bctx.Clips.Clips) == 0 {
		return nil, fmt.Errorf("no speech clips to assemble")
	}

	runs, jingleAssets := collectRuns(bctx.Script, bctx.Clips.Clips, bctx.PauseSec)

	b := &filterBuilder{}

	// Register all clip inputs.
	clipInputIdx := make([]int, len(bctx.Clips.Clips))
	for i, clip := range bctx.Clips.Clips {
		clipInputIdx[i] = b.addInput(filepath.Join(bctx.ClipsDir, clip.File))
	}

	// Build each run and collect their output labels.
	runLabels := make([]string, len(runs))
	for i, run := range runs {
		if !hasClips(run.speechItems) {
			runLabels[i] = ""
			continue
		}
		label := buildRun(b, run, clipInputIdx, bctx.Assets, bctx.PauseSec, i)
		runLabels[i] = label
	}

	// Build jingle inputs and collect their labels.
	jingleLabels := make([]string, len(jingleAssets))
	for i, assetName := range jingleAssets {
		entry, ok := bctx.Assets.Jingle[assetName]
		if !ok {
			jingleLabels[i] = ""
			continue
		}
		idx := b.addInput(entry.File)
		label := buildJingleFadeIn(b, idx, entry)
		label = applyFadeOut(b, label, idx, entry)
		jingleLabels[i] = label
	}

	// Build the final sequence of parts for full concat.
	// Structure: [run_0][j_0][run_1][j_1]...[run_N]
	// Empty runs and missing jingles are skipped.
	var parts []string
	appendPart := func(label string) {
		if len(parts) > 0 {
			pauseLabel := fmt.Sprintf("[pause_concat%d]", len(parts))
			b.addSilence(bctx.PauseSec, pauseLabel)
			parts = append(parts, pauseLabel)
		}
		parts = append(parts, label)
	}

	for i, runLabel := range runLabels {
		if runLabel != "" {
			appendPart(runLabel)
		}
		if i < len(jingleLabels) && jingleLabels[i] != "" {
			appendPart(jingleLabels[i])
		}
	}

	if len(parts) == 0 {
		return nil, fmt.Errorf("no audio parts to assemble")
	}

	var currentLabel string
	if len(parts) == 1 {
		currentLabel = parts[0]
	} else {
		b.addFilter(fmt.Sprintf("%sconcat=n=%d:v=0:a=1[full_concat]", strings.Join(parts, ""), len(parts)))
		currentLabel = "[full_concat]"
	}

	// loudnorm: applied once to the full assembled output.
	b.addFilter(fmt.Sprintf("%s%s[norm_out]", currentLabel, outputNormFilter))

	// Peak limiter: prevents clipping after loudnorm.
	b.addFilter(fmt.Sprintf("[norm_out]alimiter=limit=%.3f:level=0[out]", outputLimiterLimit))

	return &FFmpegArgs{
		Inputs:        b.inputs,
		FilterComplex: strings.Join(b.filters, ";"),
		OutputArgs:    []string{"-map", "[out]", "-c:a", "libmp3lame", "-q:a", "2"},
		OutputPath:    bctx.OutPath,
	}, nil
}

// hasClips returns true if the speech timeline contains at least one clip item.
func hasClips(items []speechItem) bool {
	return slices.ContainsFunc(items, func(it speechItem) bool { return it.clipIndex >= 0 })
}

// collectRuns scans the script and splits it into runs separated by jingle segments.
// Returns the list of runs and the list of jingle asset names between them.
func collectRuns(script model.Script, clips []model.ClipMeta, pauseSec float64) ([]runData, []string) {
	clipIdx := 0
	var jingles []string
	var runs []runData

	current := newRun()

	for _, seg := range script.Segments {
		switch seg.Type {
		case model.SegmentTypeSpeech:
			if clipIdx < len(clips) {
				current.speechItems = append(current.speechItems, speechItem{clipIndex: clipIdx})
				current.durationMs += int(clips[clipIdx].DurationSec * 1000)
				clipIdx++
			}
			current.durationMs += int(pauseSec * 1000)

		case model.SegmentTypePause:
			if seg.DurationSec > 0 {
				current.speechItems = append(current.speechItems, speechItem{clipIndex: -1, durationSec: seg.DurationSec})
				current.durationMs += int(seg.DurationSec * 1000)
			}

		case model.SegmentTypeSE:
			current.seEvents = append(current.seEvents, seEvent{
				assetName: seg.AssetName,
				offsetMs:  current.durationMs,
			})

		case model.SegmentTypeBGM:
			// Finalize active BGM interval if any.
			if current.activeBGMStart >= 0 {
				current.bgmIntervals = append(current.bgmIntervals, bgmInterval{
					startMs:   current.activeBGMStart,
					endMs:     current.durationMs,
					assetName: current.activeBGMName,
				})
			}
			if seg.AssetName != "" {
				current.activeBGMStart = current.durationMs
				current.activeBGMName = seg.AssetName
			} else {
				current.activeBGMStart = -1
				current.activeBGMName = ""
			}

		case model.SegmentTypeJingle:
			// Finalize active BGM (doesn't cross jingle boundary).
			if current.activeBGMStart >= 0 {
				current.bgmIntervals = append(current.bgmIntervals, bgmInterval{
					startMs:   current.activeBGMStart,
					endMs:     -1,
					assetName: current.activeBGMName,
				})
			}
			runs = append(runs, current.runData)
			jingles = append(jingles, seg.AssetName)
			current = newRun()
		}
	}

	// Finalize active BGM for the last run.
	if current.activeBGMStart >= 0 {
		current.bgmIntervals = append(current.bgmIntervals, bgmInterval{
			startMs:   current.activeBGMStart,
			endMs:     -1,
			assetName: current.activeBGMName,
		})
	}
	runs = append(runs, current.runData)

	return runs, jingles
}

// runBuilder is a temporary helper to track state while building a run.
type runBuilder struct {
	runData
	activeBGMStart int
	activeBGMName  string
}

func newRun() runBuilder {
	return runBuilder{activeBGMStart: -1}
}

// buildRun constructs the filter_complex entries for a single run and returns the output label.
func buildRun(b *filterBuilder, run runData, clipInputIdx []int, assets config.AssetsConfig, pauseSec float64, runIdx int) string {
	speechLabel := buildSpeechConcat(b, run.speechItems, clipInputIdx, pauseSec, runIdx)
	currentLabel := speechLabel

	// Single pass over bgmIntervals to determine hasBGM, hasDucking, and firstDuckRatio.
	hasBGM, hasDucking := false, false
	firstDuckRatio := 0.0
	for _, interval := range run.bgmIntervals {
		if e, ok := assets.BGM[interval.assetName]; ok {
			hasBGM = true
			if e.DuckRatio > 0 && !hasDucking {
				hasDucking = true
				firstDuckRatio = e.DuckRatio
			}
		}
	}

	// Split speech for sidechain ducking if any BGM has duck_ratio > 0.
	duckLabel := ""
	if hasDucking {
		mixLabel := fmt.Sprintf("[run%d_speech_mix]", runIdx)
		duckLabel = fmt.Sprintf("[run%d_speech_duck]", runIdx)
		b.addFilter(fmt.Sprintf("%sasplit=2%s%s", currentLabel, mixLabel, duckLabel))
		currentLabel = mixLabel
	}

	// SE overlay.
	for i, ev := range run.seEvents {
		entry, ok := assets.SE[ev.assetName]
		if !ok {
			continue
		}
		seIdx := b.addInput(entry.File)
		seKey := fmt.Sprintf("run%d_se%d", runIdx, i)
		seRawLabel := fmt.Sprintf("[%d:a]", seIdx)
		seLabel := applySilenceTrim(b, seRawLabel, seKey, entry.EffectiveTrimSilence())
		delayedLabel := fmt.Sprintf("[%s]", seKey)
		nextLabel := fmt.Sprintf("[run%d_after_se%d]", runIdx, i)
		b.addFilter(fmt.Sprintf("%svolume=%.2f,adelay=%d|%d%s",
			seLabel, entry.Volume, ev.offsetMs, ev.offsetMs, delayedLabel))
		b.addFilter(fmt.Sprintf("%s%samix=inputs=2:duration=first:normalize=0%s",
			currentLabel, delayedLabel, nextLabel))
		currentLabel = nextLabel
	}

	// BGM intervals overlay.
	if hasBGM {
		bgmParts := make([]string, 0, len(run.bgmIntervals))
		for i, interval := range run.bgmIntervals {
			entry, ok := assets.BGM[interval.assetName]
			if !ok {
				continue
			}
			var bgmIdx int
			if entry.Loop {
				bgmIdx = b.addInput(entry.File, "-stream_loop", "-1")
			} else {
				bgmIdx = b.addInput(entry.File)
			}
			intervalLabel := fmt.Sprintf("[run%d_bgm%d_raw]", runIdx, i)
			endMs := interval.endMs
			if endMs < 0 {
				endMs = run.durationMs
			}
			durationSec := float64(endMs-interval.startMs) / 1000.0
			b.addFilter(fmt.Sprintf("[%d:a]volume=%.2f,atrim=duration=%.3f,adelay=%d|%d%s",
				bgmIdx, entry.Volume, durationSec, interval.startMs, interval.startMs, intervalLabel))
			bgmParts = append(bgmParts, intervalLabel)
		}

		if len(bgmParts) > 0 {
			var bgmFullLabel string
			if len(bgmParts) == 1 {
				bgmFullLabel = bgmParts[0]
			} else {
				bgmFullLabel = fmt.Sprintf("[run%d_bgm_full]", runIdx)
				b.addFilter(fmt.Sprintf("%samix=inputs=%d:duration=longest:normalize=0%s",
					strings.Join(bgmParts, ""), len(bgmParts), bgmFullLabel))
			}

			// Apply sidechain ducking if duck ratio > 0.
			if duckLabel != "" {
				bgmDuckedLabel := fmt.Sprintf("[run%d_bgm_ducked]", runIdx)
				b.addFilter(fmt.Sprintf("%s%ssidechaincompress=threshold=0.02:ratio=%.1f%s",
					bgmFullLabel, duckLabel, firstDuckRatio, bgmDuckedLabel))
				bgmFullLabel = bgmDuckedLabel
			}

			afterBGMLabel := fmt.Sprintf("[run%d_after_bgm]", runIdx)
			b.addFilter(fmt.Sprintf("%s%samix=inputs=2:duration=first:normalize=0%s",
				currentLabel, bgmFullLabel, afterBGMLabel))
			currentLabel = afterBGMLabel
		}
	}

	return currentLabel
}

// buildSpeechConcat generates filter entries for the speech timeline of a run.
// Items can be clip references or explicit pauses; default pauseSec is inserted between consecutive clips.
func buildSpeechConcat(b *filterBuilder, items []speechItem, clipInputIdx []int, pauseSec float64, runIdx int) string {
	// Find the last clip's position (reverse scan) to know where to stop adding inter-clip pauses.
	lastClipPos := -1
	for i := len(items) - 1; i >= 0; i-- {
		if items[i].clipIndex >= 0 {
			lastClipPos = i
			break
		}
	}

	parts := make([]string, 0, 2*len(items))
	silenceIdx := 0
	for i, item := range items {
		if item.clipIndex >= 0 {
			parts = append(parts, fmt.Sprintf("[%d:a]", clipInputIdx[item.clipIndex]))
			if i != lastClipPos {
				label := fmt.Sprintf("[run%d_p%d]", runIdx, silenceIdx)
				b.addSilence(pauseSec, label)
				parts = append(parts, label)
				silenceIdx++
			}
		} else {
			label := fmt.Sprintf("[run%d_p%d]", runIdx, silenceIdx)
			b.addSilence(item.durationSec, label)
			parts = append(parts, label)
			silenceIdx++
		}
	}

	n := len(parts)
	if n == 1 {
		return parts[0]
	}
	concatLabel := fmt.Sprintf("[run%d_speech]", runIdx)
	b.addFilter(fmt.Sprintf("%sconcat=n=%d:v=0:a=1%s", strings.Join(parts, ""), n, concatLabel))
	return concatLabel
}

// applySilenceTrim strips leading and trailing silence from currentLabel when enabled.
// Trailing silence is removed via the areverse trick (same pattern as applyFadeOut).
// Returns currentLabel unchanged when enabled is false.
func applySilenceTrim(b *filterBuilder, currentLabel string, key string, enabled bool) string {
	if !enabled {
		return currentLabel
	}
	outLabel := fmt.Sprintf("[%s_st]", key)
	b.addFilter(fmt.Sprintf(
		"%ssilenceremove=start_periods=1:start_threshold=%s,areverse,silenceremove=start_periods=1:start_threshold=%s,areverse%s",
		currentLabel, silenceTrimThreshold, silenceTrimThreshold, outLabel))
	return outLabel
}

// buildJingleFadeIn applies silence trim then fade-in to a jingle input and returns the resulting label.
func buildJingleFadeIn(b *filterBuilder, idx int, entry config.JingleEntry) string {
	label := applySilenceTrim(b, fmt.Sprintf("[%d:a]", idx), fmt.Sprintf("jingle%d", idx), entry.EffectiveTrimSilence())
	return applyFadeIn(b, label, idx, entry.FadeIn)
}

// applyFadeOut applies fade-out to the current label and returns the resulting label.
// Returns currentLabel unchanged when entry.FadeOut <= 0.
func applyFadeOut(b *filterBuilder, currentLabel string, idx int, entry config.JingleEntry) string {
	if entry.FadeOut <= 0 {
		return currentLabel
	}
	fadedLabel := fmt.Sprintf("[jingle%d_fo]", idx)
	b.addFilter(fmt.Sprintf("%sareverse,afade=t=in:d=%.3f,areverse%s", currentLabel, entry.FadeOut, fadedLabel))
	return fadedLabel
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
