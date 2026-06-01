package pipeline

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/canpok1/vox-radio/internal/assemble"
	"github.com/canpok1/vox-radio/internal/config"
	"github.com/canpok1/vox-radio/internal/fileio"
	"github.com/canpok1/vox-radio/internal/manifest"
	"github.com/canpok1/vox-radio/internal/model"
)

// ProgramSummarizer generates a summary of the episode from the final script.
type ProgramSummarizer interface {
	Summarize(ctx context.Context, scr model.Script) (string, error)
}

// CornerSummarizer generates a summary for a single corner from its script lines.
type CornerSummarizer interface {
	SummarizeCorner(ctx context.Context, corner model.CornerLines) (model.CornerSummary, error)
}

// Collector gathers articles per corner from configured sources.
type Collector interface {
	RunAll(ctx context.Context, corners []config.CornerConfig) (model.Articles, error)
}

// Rundowner selects articles and designs the talk flow for each corner.
type Rundowner interface {
	Run(ctx context.Context, corners []config.CornerConfig, articles model.Articles) (model.Rundown, error)
}

// Scripter generates a radio script from a rundown.
type Scripter interface {
	Generate(ctx context.Context, program config.ProgramConfig, rundown model.Rundown, corners []config.CornerConfig, chars map[string]config.CharacterConfig) (model.Script, error)
}

// Synther synthesizes voice clips from a script.
type Synther interface {
	Run(ctx context.Context, scr model.Script, outDir string) (*model.ClipsMeta, error)
}

// Assembler produces an MP3 episode from clips and a script.
type Assembler interface {
	Run(ctx context.Context, scr model.Script, clips model.ClipsMeta, clipsDir, outPath string) (*assemble.Result, error)
}

// Options configures a single pipeline run.
type Options struct {
	OutDir      string
	GeneratedAt time.Time // zero value means time.Now().UTC()
}

// Runner orchestrates the full collect→rundown→script→synth→assemble→manifest pipeline.
type Runner struct {
	Profile           *config.Profile
	Config            *config.Config
	Collector         Collector
	Rundowner         Rundowner
	Scripter          Scripter
	Synther           Synther
	Assembler         Assembler
	ProgramSummarizer ProgramSummarizer // optional; if nil, program summary is omitted from manifest
	CornerSummarizer  CornerSummarizer  // optional; if nil, corner summaries are omitted from manifest
	Logger            *slog.Logger      // optional; if nil, slog.Default() is used
}

// Run executes the full pipeline, writing intermediate files to <outDir>/intermediate/.
func (r *Runner) Run(ctx context.Context, opts Options) error {
	outDir := opts.OutDir

	if err := fileio.EnsureOutputDirs(outDir); err != nil {
		return fmt.Errorf("create output dirs: %w", err)
	}

	articles, err := r.Collector.RunAll(ctx, r.Profile.Corners)
	if err != nil {
		return fmt.Errorf("collect: %w", err)
	}
	if err := fileio.WriteJSON(fileio.ArticlesPath(outDir), articles); err != nil {
		return err
	}

	rundown, err := r.Rundowner.Run(ctx, r.Profile.Corners, articles)
	if err != nil {
		return fmt.Errorf("rundown: %w", err)
	}
	if err := fileio.WriteJSON(fileio.RundownPath(outDir), rundown); err != nil {
		return err
	}

	var chars map[string]config.CharacterConfig
	if r.Config != nil {
		chars = r.Config.Characters
	}

	scr, err := r.Scripter.Generate(ctx, r.Profile.Program, rundown, r.Profile.Corners, chars)
	if err != nil {
		return fmt.Errorf("script: %w", err)
	}
	if err := fileio.WriteJSON(fileio.ScriptPath(outDir), scr); err != nil {
		return err
	}

	clips, err := r.Synther.Run(ctx, scr, fileio.ClipsDir(outDir))
	if err != nil {
		return fmt.Errorf("synth: %w", err)
	}
	if clips == nil {
		clips = &model.ClipsMeta{Clips: make([]model.ClipMeta, 0)}
	}

	if _, err := r.Assembler.Run(ctx, scr, *clips, fileio.ClipsDir(outDir), fileio.EpisodePath(outDir)); err != nil {
		return fmt.Errorf("assemble: %w", err)
	}

	generatedAt := opts.GeneratedAt
	if generatedAt.IsZero() {
		generatedAt = time.Now().UTC()
	}

	var programSummary string
	if r.ProgramSummarizer != nil {
		programSummary, err = r.ProgramSummarizer.Summarize(ctx, scr)
		if err != nil {
			return fmt.Errorf("summarize program: %w", err)
		}
	}

	var cornerSummaries map[string]model.CornerSummary
	if r.CornerSummarizer != nil {
		var scriptLines model.ScriptLines
		if err := fileio.ReadJSON(fileio.LinesPath(outDir), &scriptLines); err != nil {
			return fmt.Errorf("read script lines for corner summarization: %w", err)
		}
		cornerSummaries = make(map[string]model.CornerSummary, len(scriptLines.Corners))
		for _, cl := range scriptLines.Corners {
			cs, err := r.CornerSummarizer.SummarizeCorner(ctx, cl)
			if err != nil {
				return fmt.Errorf("summarize corner %s: %w", cl.Title, err)
			}
			cornerSummaries[cl.Title] = cs
		}
	}

	logger := r.Logger
	if logger == nil {
		logger = slog.Default()
	}
	manifestLogger := logger.With("step", "manifest")
	manifestLogger.Info("開始")
	manifestStart := time.Now()

	m := manifest.Build(r.Profile.Program, r.Profile.Corners, rundown, fileio.FileEpisode, generatedAt, programSummary, cornerSummaries)
	if err := fileio.WriteJSON(fileio.ManifestPath(outDir), m); err != nil {
		return fmt.Errorf("write manifest: %w", err)
	}

	manifestLogger.Info(fmt.Sprintf("完了 (%.1fs)", time.Since(manifestStart).Seconds()))

	return nil
}
