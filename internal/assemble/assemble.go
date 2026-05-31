package assemble

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

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
	runFFmpeg    func(ctx context.Context, args []string) error
	getDuration  func(path string) (float64, error)
	getFileSize  func(path string) (int64, error)
}

// New creates a new Assembler that calls ffmpeg and ffprobe.
func New(assetsConfig config.AssetsConfig, program config.ProgramConfig) *Assembler {
	return &Assembler{
		AssetsConfig: assetsConfig,
		Program:      program,
		runFFmpeg:    runFFmpegCmd,
		getDuration:  mediainfo.Duration,
		getFileSize:  mediainfo.FileSize,
	}
}

// Run assembles the given clips and script into an mp3 at outPath.
// It returns the duration and file size of the resulting mp3.
func (a *Assembler) Run(ctx context.Context, script model.Script, clips model.ClipsMeta, clipsDir string, outPath string) (*Result, error) {
	if err := os.MkdirAll(filepath.Dir(outPath), 0o755); err != nil {
		return nil, fmt.Errorf("create output dir: %w", err)
	}

	pauseSec := a.Program.SegmentPauseSec
	if pauseSec == 0 {
		pauseSec = defaultPauseSec
	}

	bctx := BuildContext{
		Script:   script,
		Clips:    clips,
		ClipsDir: clipsDir,
		Assets:   a.AssetsConfig,
		PauseSec: pauseSec,
		OutPath:  outPath,
	}

	ffArgs, err := BuildFFmpegArgs(bctx)
	if err != nil {
		return nil, fmt.Errorf("build ffmpeg args: %w", err)
	}

	cmdArgs := buildCmdArgs(ffArgs)
	if err := a.runFFmpeg(ctx, cmdArgs); err != nil {
		return nil, fmt.Errorf("ffmpeg: %w", err)
	}

	dur, err := a.getDuration(outPath)
	if err != nil {
		return nil, fmt.Errorf("get duration: %w", err)
	}

	size, err := a.getFileSize(outPath)
	if err != nil {
		return nil, fmt.Errorf("get file size: %w", err)
	}

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

func runFFmpegCmd(ctx context.Context, args []string) error {
	cmd := exec.CommandContext(ctx, "ffmpeg", args...)
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
