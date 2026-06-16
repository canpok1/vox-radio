package assemble

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/canpok1/vox-radio/internal/config"
	"github.com/canpok1/vox-radio/internal/logging"
	"github.com/canpok1/vox-radio/internal/mediainfo"
	"github.com/canpok1/vox-radio/internal/model"
)

const defaultPauseSec = 0.3

// Result holds the output metrics for an assembled episode.
type Result struct {
	DurationSec     float64
	Bytes           int64
	CornerDurations map[string]float64
}

// Assembler assembles speech clips and assets into a final mp3.
type Assembler struct {
	AssetsConfig config.AssetsConfig
	Program      config.ProgramConfig
	runFFmpeg    func(ctx context.Context, args []string, w io.Writer) error
	getDuration  func(path string) (float64, error)
	getFileSize  func(path string) (int64, error)
	logger       *slog.Logger
	ffmpegWriter io.Writer
}

// Option configures an Assembler.
type Option func(*Assembler)

// WithLogger sets the logger used for progress messages.
func WithLogger(l *slog.Logger) Option {
	return func(a *Assembler) { a.logger = l }
}

// WithFFmpegWriter sets the writer where ffmpeg stdout/stderr is captured.
// If nil (the default), ffmpeg output is discarded.
func WithFFmpegWriter(w io.Writer) Option {
	return func(a *Assembler) { a.ffmpegWriter = w }
}

// New creates a new Assembler that calls ffmpeg and ffprobe.
func New(assetsConfig config.AssetsConfig, program config.ProgramConfig, opts ...Option) *Assembler {
	a := &Assembler{
		AssetsConfig: assetsConfig,
		Program:      program,
		runFFmpeg:    runFFmpegCmd,
		getDuration:  mediainfo.Duration,
		getFileSize:  mediainfo.FileSize,
		logger:       slog.Default(),
	}
	for _, opt := range opts {
		opt(a)
	}
	return a
}

// Run assembles the given clips and script into an mp3 at outPath.
// It returns the duration and file size of the resulting mp3.
func (a *Assembler) Run(ctx context.Context, script model.Script, clips model.ClipsMeta, clipsDir string, outPath string, meta model.EpisodeMeta) (*Result, error) {
	logger := a.logger.With("step", "assemble")
	done := logging.StartStep(ctx, logger, "開始")

	if err := os.MkdirAll(filepath.Dir(outPath), 0o755); err != nil {
		return nil, fmt.Errorf("create output dir: %w", err)
	}

	seDurations, err := a.collectSEDurations(script)
	if err != nil {
		return nil, err
	}

	jingleDurations, err := a.collectJingleDurations(script)
	if err != nil {
		return nil, err
	}

	bctx := BuildContext{
		Script:      script,
		Clips:       clips,
		ClipsDir:    clipsDir,
		Assets:      a.AssetsConfig,
		PauseSec:    defaultPauseSec,
		OutPath:     outPath,
		SEDurations: seDurations,
		Program:     a.Program,
		Meta:        meta,
	}

	ffArgs, err := BuildFFmpegArgs(bctx)
	if err != nil {
		return nil, fmt.Errorf("build ffmpeg args: %w", err)
	}

	cmdArgs := buildCmdArgs(ffArgs)

	if a.ffmpegWriter != nil {
		_, _ = fmt.Fprintf(a.ffmpegWriter, "--- ffmpeg command ---\nffmpeg %s\n--- ffmpeg output ---\n", strings.Join(cmdArgs, " "))
	}

	if err := a.runFFmpeg(ctx, cmdArgs, a.ffmpegWriter); err != nil {
		return nil, fmt.Errorf("ffmpeg: %w", err)
	}

	if a.ffmpegWriter != nil {
		_, _ = fmt.Fprintln(a.ffmpegWriter, "--- end ffmpeg output ---")
	}

	dur, err := a.getDuration(outPath)
	if err != nil {
		return nil, fmt.Errorf("get duration: %w", err)
	}

	size, err := a.getFileSize(outPath)
	if err != nil {
		return nil, fmt.Errorf("get file size: %w", err)
	}

	done(fmt.Sprintf("duration=%.1fs, %.2fMB", dur, float64(size)/(1024*1024)))

	seSequentialDurations := make(map[string]float64, len(seDurations))
	for name, dur := range seDurations {
		if entry, ok := a.AssetsConfig.SE[name]; ok && !entry.EffectiveOverlay() {
			seSequentialDurations[name] = dur
		}
	}
	cornerDurations := computeCornerDurations(clips.Clips, script, defaultPauseSec, jingleDurations, seSequentialDurations)

	return &Result{DurationSec: dur, Bytes: size, CornerDurations: cornerDurations}, nil
}

// collectAssetDurations fetches the playback duration for each unique asset of segType
// that appears in the script. getFile returns the file path for a given asset name (ok=false
// means the asset is not in the config and should be skipped). errLabel is used in errors.
func (a *Assembler) collectAssetDurations(script model.Script, segType model.SegmentType, getFile func(name string) (string, bool), errLabel string) (map[string]float64, error) {
	seen := make(map[string]struct{})
	for _, seg := range script.Segments {
		if seg.Type == segType && seg.AssetName != "" {
			seen[seg.AssetName] = struct{}{}
		}
	}
	durations := make(map[string]float64)
	for name := range seen {
		file, ok := getFile(name)
		if !ok {
			continue
		}
		dur, err := a.getDuration(file)
		if err != nil {
			return nil, fmt.Errorf("get %s duration for %q: %w", errLabel, name, err)
		}
		durations[name] = dur
	}
	return durations, nil
}

func (a *Assembler) collectJingleDurations(script model.Script) (map[string]float64, error) {
	return a.collectAssetDurations(script, model.SegmentTypeJingle, func(name string) (string, bool) {
		e, ok := a.AssetsConfig.Jingle[name]
		return e.File, ok
	}, "jingle")
}

func (a *Assembler) collectSEDurations(script model.Script) (map[string]float64, error) {
	return a.collectAssetDurations(script, model.SegmentTypeSE, func(name string) (string, bool) {
		e, ok := a.AssetsConfig.SE[name]
		return e.File, ok
	}, "SE")
}

// buildCmdArgs converts FFmpegArgs into a flat argument slice for exec.Command.
func buildCmdArgs(ffArgs *FFmpegArgs) []string {
	var args []string
	for _, inp := range ffArgs.Inputs {
		args = append(args, inp.PreOptions...)
		args = append(args, "-i", inp.Path)
	}
	args = append(args, "-filter_complex", ffArgs.FilterComplex)
	args = append(args, ffArgs.OutputArgs...)
	args = append(args, "-y", ffArgs.OutputPath)
	return args
}

func runFFmpegCmd(ctx context.Context, args []string, w io.Writer) error {
	cmd := exec.CommandContext(ctx, "ffmpeg", args...)
	if w != nil {
		cmd.Stderr = w
		cmd.Stdout = w
	}
	return cmd.Run()
}
