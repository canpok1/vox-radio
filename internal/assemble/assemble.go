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
	"time"

	"github.com/canpok1/vox-radio/internal/config"
	"github.com/canpok1/vox-radio/internal/mediainfo"
	"github.com/canpok1/vox-radio/internal/model"
)

const defaultPauseSec = 0.3

// Result holds the output metrics for an assembled episode.
type Result struct {
	DurationSec float64
	Bytes       int64
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
func (a *Assembler) Run(ctx context.Context, script model.Script, clips model.ClipsMeta, clipsDir string, outPath string) (*Result, error) {
	logger := a.logger.With("step", "assemble")
	start := time.Now()

	logger.Info("開始")

	if err := os.MkdirAll(filepath.Dir(outPath), 0o755); err != nil {
		return nil, fmt.Errorf("create output dir: %w", err)
	}

	pauseSec := a.Program.SegmentPauseSec
	if pauseSec == 0 {
		pauseSec = defaultPauseSec
	}

	bctx := BuildContext{
		Script:        script,
		Clips:         clips,
		ClipsDir:      clipsDir,
		Assets:        a.AssetsConfig,
		PauseSec:      pauseSec,
		OutPath:       outPath,
		OpeningJingle: a.Program.OpeningJingle,
		EndingJingle:  a.Program.EndingJingle,
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

	logger.Info(fmt.Sprintf("完了 (duration=%.1fs, %.2fMB, %.1fs)", dur, float64(size)/(1024*1024), time.Since(start).Seconds()))

	return &Result{DurationSec: dur, Bytes: size}, nil
}

// buildCmdArgs converts FFmpegArgs into a flat argument slice for exec.Command.
func buildCmdArgs(ffArgs *FFmpegArgs) []string {
	var args []string
	for _, inp := range ffArgs.Inputs {
		args = append(args, "-i", inp)
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
