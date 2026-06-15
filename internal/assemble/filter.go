package assemble

import (
	"fmt"
	"math"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"time"

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
// Jingles are placed as SegmentTypeJingle in Script; OP/ED jingles are injected
// into Script by the Assembler before calling BuildFFmpegArgs.
type BuildContext struct {
	Script      model.Script
	Clips       model.ClipsMeta
	ClipsDir    string
	Assets      config.AssetsConfig
	PauseSec    float64
	OutPath     string
	SEDurations map[string]float64 // duration in seconds per SE asset name (for sequential SE)
	Program     config.ProgramConfig
	Meta        model.EpisodeMeta
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

type speechItemKind int

const (
	speechItemKindClip  speechItemKind = iota
	speechItemKindPause                // explicit pause segment
	speechItemKindSeqSE                // sequential SE (plays to completion before next item)
)

// speechItem represents an element in the speech timeline: a clip, an explicit pause, or a sequential SE.
type speechItem struct {
	kind          speechItemKind
	clipIndex     int     // used when kind == speechItemKindClip; >= 0
	durationSec   float64 // used when kind == speechItemKindPause
	pauseAfterSec float64 // used when kind == speechItemKindClip: silence after clip
	seAssetName   string  // used when kind == speechItemKindSeqSE
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

	runs, jingleAssets := collectRuns(bctx.Script, bctx.Clips.Clips, bctx.PauseSec, bctx.Assets, bctx.SEDurations)

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
		label := buildRun(b, run, clipInputIdx, bctx.Assets, i)
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
		label = applyFadeOut(b, label, fmt.Sprintf("jingle%d", idx), entry.FadeOut)
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

	outputArgs := append([]string{"-map", "[out]", "-c:a", "libmp3lame"}, audioQualityArgs(bctx.Program.EffectiveAudioQuality())...)
	if metaArgs := buildMetadataArgs(bctx); len(metaArgs) > 0 {
		outputArgs = append(outputArgs, "-id3v2_version", "3")
		outputArgs = append(outputArgs, metaArgs...)
	}

	return &FFmpegArgs{
		Inputs:        b.inputs,
		FilterComplex: strings.Join(b.filters, ";"),
		OutputArgs:    outputArgs,
		OutputPath:    bctx.OutPath,
	}, nil
}

// audioQualityArgs returns the ffmpeg VBR quality arguments for the given preset.
// preset must be one of "high", "standard", "low" (values from config.DefaultAudioQuality etc.).
// Unknown values fall back to "standard" (-q:a 2).
func audioQualityArgs(preset string) []string {
	switch preset {
	case "high":
		return []string{"-q:a", "0"}
	case "low":
		return []string{"-q:a", "5"}
	default:
		return []string{"-q:a", "2"}
	}
}

// buildMetadataArgs constructs ffmpeg -metadata key=value pairs for ID3 tagging.
// Empty values and zero numbers/times result in the corresponding tag being omitted.
func buildMetadataArgs(bctx BuildContext) []string {
	program := bctx.Program
	meta := bctx.Meta
	var args []string
	if program.Title != "" {
		args = append(args, "-metadata", "album="+program.Title)
	}
	if title := model.EpisodeDisplayTitle(meta.Number, meta.Title, program.Title); title != "" {
		args = append(args, "-metadata", "title="+title)
	}
	if program.Author != "" {
		args = append(args, "-metadata", "artist="+program.Author)
	}
	if meta.Number > 0 {
		args = append(args, "-metadata", "track="+strconv.Itoa(meta.Number))
	}
	if !meta.GeneratedAt.IsZero() {
		loc, err := program.Location()
		if err != nil {
			loc = time.UTC
		}
		args = append(args, "-metadata", "date="+meta.GeneratedAt.In(loc).Format("2006-01-02"))
	}
	return args
}

// hasClips returns true if the speech timeline contains at least one clip item.
func hasClips(items []speechItem) bool {
	return slices.ContainsFunc(items, func(it speechItem) bool { return it.kind == speechItemKindClip })
}

// nextClipSpeakerRole scans segments forward from index from+1 and returns the speaker_role
// of the next speech segment, skipping SE/BGM segments. Returns ("", false) when a pause or
// jingle segment is encountered first, or when the end of segments is reached.
func nextClipSpeakerRole(segments []model.ScriptSegment, from int) (string, bool) {
	for j := from + 1; j < len(segments); j++ {
		switch segments[j].Type {
		case model.SegmentTypeSpeech:
			return segments[j].SpeakerRole, true
		case model.SegmentTypePause, model.SegmentTypeJingle:
			return "", false
		}
	}
	return "", false
}

// collectRuns scans the script and splits it into runs separated by jingle segments.
// Returns the list of runs and the list of jingle asset names between them.
func collectRuns(script model.Script, clips []model.ClipMeta, pauseSec float64, assets config.AssetsConfig, seDurations map[string]float64) ([]runData, []string) {
	clipIdx := 0
	var jingles []string
	var runs []runData

	current := newRun()

	for i, seg := range script.Segments {
		switch seg.Type {
		case model.SegmentTypeSpeech:
			if clipIdx < len(clips) {
				nextRole, hasNext := nextClipSpeakerRole(script.Segments, i)
				pauseAfter := pauseSec
				if hasNext && nextRole == seg.SpeakerRole {
					pauseAfter = 0
				}
				current.speechItems = append(current.speechItems, speechItem{
					kind:          speechItemKindClip,
					clipIndex:     clipIdx,
					pauseAfterSec: pauseAfter,
				})
				current.durationMs += int(clips[clipIdx].DurationSec*1000) + int(pauseAfter*1000)
				clipIdx++
			}

		case model.SegmentTypePause:
			if seg.DurationSec > 0 {
				current.speechItems = append(current.speechItems, speechItem{kind: speechItemKindPause, durationSec: seg.DurationSec})
				current.durationMs += int(seg.DurationSec * 1000)
			}

		case model.SegmentTypeSE:
			if seEntry, ok := assets.SE[seg.AssetName]; ok && !seEntry.EffectiveOverlay() {
				// Sequential SE: concatenate into speech timeline and advance durationMs.
				// trim_silence may reduce actual play length below raw file length; this
				// causes durationMs to be slightly over-estimated, which is acceptable.
				current.speechItems = append(current.speechItems, speechItem{
					kind:        speechItemKindSeqSE,
					seAssetName: seg.AssetName,
				})
				current.durationMs += int(seDurations[seg.AssetName] * 1000)
			} else {
				current.seEvents = append(current.seEvents, seEvent{
					assetName: seg.AssetName,
					offsetMs:  current.durationMs,
				})
			}

		case model.SegmentTypeBGM:
			current.closeBGMInterval(current.durationMs)
			if seg.AssetName != "" {
				current.activeBGMStart = current.durationMs
				current.activeBGMName = seg.AssetName
			}

		case model.SegmentTypeJingle:
			// BGM does not cross jingle boundaries.
			current.closeBGMInterval(-1)
			runs = append(runs, current.runData)
			jingles = append(jingles, seg.AssetName)
			current = newRun()
		}
	}

	// Finalize active BGM for the last run.
	current.closeBGMInterval(-1)
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

// closeBGMInterval finalizes the active BGM interval if any, appending it to bgmIntervals.
// endMs=-1 means the interval extends to the end of the run.
// After the call activeBGMStart is reset to -1.
func (rb *runBuilder) closeBGMInterval(endMs int) {
	if rb.activeBGMStart < 0 {
		return
	}
	rb.bgmIntervals = append(rb.bgmIntervals, bgmInterval{
		startMs:   rb.activeBGMStart,
		endMs:     endMs,
		assetName: rb.activeBGMName,
	})
	rb.activeBGMStart = -1
	rb.activeBGMName = ""
}

// buildRun constructs the filter_complex entries for a single run and returns the output label.
// Processing is split into four phases: speech concat → duck split → SE overlay → BGM overlay.
func buildRun(b *filterBuilder, run runData, clipInputIdx []int, assets config.AssetsConfig, runIdx int) string {
	currentLabel := buildSpeechConcat(b, run.speechItems, clipInputIdx, assets, runIdx)
	currentLabel, duckLabel, duckRatio := buildDuckSplit(b, run.bgmIntervals, assets, runIdx, currentLabel)
	currentLabel = buildSEOverlay(b, run.seEvents, assets, runIdx, currentLabel)
	return buildBGMOverlay(b, run, assets, runIdx, currentLabel, duckLabel, duckRatio)
}

// buildDuckSplit checks whether any BGM interval has duck_ratio > 0.
// If so, it emits an asplit=2 filter on currentLabel and returns the new mix label,
// a sidechain duck label, and the duck ratio. Otherwise currentLabel, "", and 0 are returned.
func buildDuckSplit(b *filterBuilder, bgmIntervals []bgmInterval, assets config.AssetsConfig, runIdx int, currentLabel string) (string, string, float64) {
	for _, interval := range bgmIntervals {
		if e, ok := assets.BGM[interval.assetName]; ok && e.DuckRatio > 0 {
			mixLabel := fmt.Sprintf("[run%d_speech_mix]", runIdx)
			duckLabel := fmt.Sprintf("[run%d_speech_duck]", runIdx)
			b.addFilter(fmt.Sprintf("%sasplit=2%s%s", currentLabel, mixLabel, duckLabel))
			return mixLabel, duckLabel, e.DuckRatio
		}
	}
	return currentLabel, "", 0
}

// buildSEOverlay overlays SE events onto currentLabel and returns the updated label.
func buildSEOverlay(b *filterBuilder, seEvents []seEvent, assets config.AssetsConfig, runIdx int, currentLabel string) string {
	for i, ev := range seEvents {
		entry, ok := assets.SE[ev.assetName]
		if !ok {
			continue
		}
		seIdx := b.addInput(entry.File)
		seKey := fmt.Sprintf("run%d_se%d", runIdx, i)
		seLabel := applySilenceTrim(b, fmt.Sprintf("[%d:a]", seIdx), seKey, entry.EffectiveTrimSilence(), entry.EffectiveTrimSilenceThresholdDB())
		delayedLabel := fmt.Sprintf("[%s]", seKey)
		nextLabel := fmt.Sprintf("[run%d_after_se%d]", runIdx, i)
		b.addFilter(fmt.Sprintf("%svolume=%.2f,adelay=%d|%d%s",
			seLabel, entry.Volume, ev.offsetMs, ev.offsetMs, delayedLabel))
		b.addFilter(fmt.Sprintf("%s%samix=inputs=2:duration=first:normalize=0%s",
			currentLabel, delayedLabel, nextLabel))
		currentLabel = nextLabel
	}
	return currentLabel
}

// computeBGMCrossfade returns per-interval crossfade extension seconds and fade-in overrides
// for adjacent BGM pairs. crossfadeExtSec[i] is the extra duration added to interval i so
// it overlaps with interval i+1; crossfadeFadeInSec[i+1] overrides interval i+1's fade-in.
func computeBGMCrossfade(intervals []bgmInterval, assets config.AssetsConfig, durationMs int) (crossfadeExtSec, crossfadeFadeInSec []float64) {
	crossfadeExtSec = make([]float64, len(intervals))
	crossfadeFadeInSec = make([]float64, len(intervals))
	for i := 0; i < len(intervals)-1; i++ {
		curr := intervals[i]
		next := intervals[i+1]
		currEntry, currOk := assets.BGM[curr.assetName]
		nextEntry, nextOk := assets.BGM[next.assetName]
		if !currOk || !nextOk {
			continue
		}
		currEndMs := curr.endMs
		if currEndMs < 0 {
			currEndMs = durationMs
		}
		if next.startMs == currEndMs {
			overlapSec := math.Min(currEntry.EffectiveFadeOut(), nextEntry.EffectiveFadeIn())
			crossfadeExtSec[i] = overlapSec
			crossfadeFadeInSec[i+1] = overlapSec
		}
	}
	return crossfadeExtSec, crossfadeFadeInSec
}

// buildBGMOverlay mixes BGM intervals (with crossfade and optional sidechain ducking) into currentLabel.
// duckLabel and duckRatio are produced by buildDuckSplit; duckLabel=="" means no ducking.
func buildBGMOverlay(b *filterBuilder, run runData, assets config.AssetsConfig, runIdx int, currentLabel, duckLabel string, duckRatio float64) string {
	bgmParts := buildBGMIntervalParts(b, run, assets, runIdx)
	if len(bgmParts) == 0 {
		return currentLabel
	}

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
			bgmFullLabel, duckLabel, duckRatio, bgmDuckedLabel))
		bgmFullLabel = bgmDuckedLabel
	}

	afterBGMLabel := fmt.Sprintf("[run%d_after_bgm]", runIdx)
	b.addFilter(fmt.Sprintf("%s%samix=inputs=2:duration=first:normalize=0%s",
		currentLabel, bgmFullLabel, afterBGMLabel))
	return afterBGMLabel
}

// buildBGMIntervalParts builds filter entries for each BGM interval in the run,
// returning the list of interval output labels for subsequent mixing.
// Returns an empty slice when no known BGM asset is active.
func buildBGMIntervalParts(b *filterBuilder, run runData, assets config.AssetsConfig, runIdx int) []string {
	crossfadeExtSec, crossfadeFadeInSec := computeBGMCrossfade(run.bgmIntervals, assets, run.durationMs)
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
		durationSec := float64(endMs-interval.startMs)/1000.0 + crossfadeExtSec[i]

		// Determine fade-in/out durations; clamp to half duration to prevent overlapping fades.
		fadeIn := crossfadeFadeInSec[i]
		if fadeIn == 0 {
			fadeIn = entry.EffectiveFadeIn()
		}
		fadeOut := crossfadeExtSec[i]
		if fadeOut == 0 {
			fadeOut = entry.EffectiveFadeOut()
		}
		half := durationSec / 2
		fadeIn = min(fadeIn, half)
		fadeOut = min(fadeOut, half)

		key := fmt.Sprintf("run%d_bgm%d", runIdx, i)
		trimLabel := fmt.Sprintf("[%s_trim]", key)
		b.addFilter(fmt.Sprintf("[%d:a]volume=%.2f,atrim=duration=%.3f%s", bgmIdx, entry.Volume, durationSec, trimLabel))
		label := applyFadeIn(b, trimLabel, key, fadeIn)
		label = applyFadeOut(b, label, key, fadeOut)
		b.addFilter(fmt.Sprintf("%sadelay=%d|%d%s", label, interval.startMs, interval.startMs, intervalLabel))
		bgmParts = append(bgmParts, intervalLabel)
	}
	return bgmParts
}

// buildSpeechConcat generates filter entries for the speech timeline of a run.
// Items can be clip references, explicit pauses, or sequential SE.
// item.pauseAfterSec determines the silence inserted after each clip
// (0 = continuation with same speaker, >0 = default pause duration).
// The trailing pause after the last clip is omitted only when the clip is the final item.
func buildSpeechConcat(b *filterBuilder, items []speechItem, clipInputIdx []int, assets config.AssetsConfig, runIdx int) string {
	parts := make([]string, 0, 2*len(items))
	silenceIdx := 0
	seqSEIdx := 0
	for i, item := range items {
		switch item.kind {
		case speechItemKindClip:
			parts = append(parts, fmt.Sprintf("[%d:a]", clipInputIdx[item.clipIndex]))
			if i < len(items)-1 && item.pauseAfterSec > 0 {
				label := fmt.Sprintf("[run%d_p%d]", runIdx, silenceIdx)
				b.addSilence(item.pauseAfterSec, label)
				parts = append(parts, label)
				silenceIdx++
			}
		case speechItemKindPause:
			label := fmt.Sprintf("[run%d_p%d]", runIdx, silenceIdx)
			b.addSilence(item.durationSec, label)
			parts = append(parts, label)
			silenceIdx++
		case speechItemKindSeqSE:
			entry, ok := assets.SE[item.seAssetName]
			if !ok {
				continue
			}
			seIdx := b.addInput(entry.File)
			seKey := fmt.Sprintf("run%d_seqse%d", runIdx, seqSEIdx)
			seLabel := applySilenceTrim(b, fmt.Sprintf("[%d:a]", seIdx), seKey, entry.EffectiveTrimSilence(), entry.EffectiveTrimSilenceThresholdDB())
			seVolLabel := fmt.Sprintf("[%s_vol]", seKey)
			b.addFilter(fmt.Sprintf("%svolume=%.2f%s", seLabel, entry.Volume, seVolLabel))
			parts = append(parts, seVolLabel)
			seqSEIdx++
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

// computeCornerDurations calculates the estimated playback duration per corner.
// Speech durations come from clips metadata; pauseAfter (defaultPauseSec) is added after each
// clip unless the next speech segment has the same speaker role within the same corner.
// Explicit pause segments and jingle durations (if provided) are attributed by CornerID.
// SE overlay and BGM segments are excluded as they play concurrently.
func computeCornerDurations(clips []model.ClipMeta, script model.Script, pauseSec float64, jingleDurations map[string]float64) map[string]float64 {
	durations := make(map[string]float64)
	clipIdx := 0
	for i, seg := range script.Segments {
		switch seg.Type {
		case model.SegmentTypeSpeech:
			if clipIdx < len(clips) {
				d := clips[clipIdx].DurationSec
				nextRole, hasNext := nextClipSpeakerRole(script.Segments, i)
				if !hasNext || nextRole != seg.SpeakerRole {
					d += pauseSec
				}
				durations[seg.CornerID] += d
				clipIdx++
			}
		case model.SegmentTypePause:
			durations[seg.CornerID] += seg.DurationSec
		case model.SegmentTypeJingle:
			if jingleDurations != nil {
				durations[seg.CornerID] += jingleDurations[seg.AssetName]
			}
		}
	}
	return durations
}

// applySilenceTrim strips leading and trailing silence from currentLabel when enabled.
// Trailing silence is removed via the areverse trick (same pattern as applyFadeOut).
// Returns currentLabel unchanged when enabled is false.
func applySilenceTrim(b *filterBuilder, currentLabel string, key string, enabled bool, thresholdDB float64) string {
	if !enabled {
		return currentLabel
	}
	outLabel := fmt.Sprintf("[%s_st]", key)
	b.addFilter(fmt.Sprintf(
		"%ssilenceremove=start_periods=1:start_threshold=%gdB,areverse,silenceremove=start_periods=1:start_threshold=%gdB,areverse%s",
		currentLabel, thresholdDB, thresholdDB, outLabel))
	return outLabel
}

// buildJingleFadeIn applies silence trim then fade-in to a jingle input and returns the resulting label.
func buildJingleFadeIn(b *filterBuilder, idx int, entry config.JingleEntry) string {
	key := fmt.Sprintf("jingle%d", idx)
	label := applySilenceTrim(b, fmt.Sprintf("[%d:a]", idx), key, entry.EffectiveTrimSilence(), entry.EffectiveTrimSilenceThresholdDB())
	return applyFadeIn(b, label, key, entry.FadeIn)
}

// applyFadeOut applies fade-out (areverse/afade/areverse) to currentLabel and returns the resulting label.
// key is used to name the intermediate output label. Returns currentLabel unchanged when fadeSec <= 0.
func applyFadeOut(b *filterBuilder, currentLabel string, key string, fadeSec float64) string {
	if fadeSec <= 0 {
		return currentLabel
	}
	fadedLabel := fmt.Sprintf("[%s_fo]", key)
	b.addFilter(fmt.Sprintf("%sareverse,afade=t=in:d=%.3f,areverse%s", currentLabel, fadeSec, fadedLabel))
	return fadedLabel
}

// applyFadeIn applies an afade=t=in filter to currentLabel and returns the output label.
// key is used to name the intermediate output label. Returns currentLabel unchanged when fadeSec <= 0.
func applyFadeIn(b *filterBuilder, currentLabel string, key string, fadeSec float64) string {
	if fadeSec <= 0 {
		return currentLabel
	}
	outLabel := fmt.Sprintf("[%s_fi]", key)
	b.addFilter(fmt.Sprintf("%safade=t=in:d=%.3f%s", currentLabel, fadeSec, outLabel))
	return outLabel
}
