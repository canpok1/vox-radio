package pipeline

import (
	"context"
	"fmt"
	"time"

	"github.com/canpok1/vox-radio/internal/config"
	"github.com/canpok1/vox-radio/internal/fileio"
	"github.com/canpok1/vox-radio/internal/manifest"
	"github.com/canpok1/vox-radio/internal/model"
)

// ProgramSummarizer generates a summary of the episode from the write-step output lines.
type ProgramSummarizer interface {
	Summarize(ctx context.Context, lines model.ScriptLines) (model.ProgramSummary, error)
}

// CornerSummarizer generates a summary for a single corner from its script lines.
type CornerSummarizer interface {
	SummarizeCorner(ctx context.Context, corner model.CornerLines, summaryLength int) (model.CornerSummary, error)
}

// Gatherer gathers articles per corner from configured sources.
type Gatherer interface {
	RunAll(ctx context.Context, corners []config.CornerConfig, excludedDedupKeys []string) (model.Articles, error)
}

// Rundowner selects articles and designs the talk flow for each corner.
type Rundowner interface {
	Run(ctx context.Context, corners []config.CornerConfig, articles model.Articles, casts []model.RundownCast) (model.Rundown, error)
}

// Scripter generates a radio script from a rundown.
type Scripter interface {
	Generate(ctx context.Context, program config.ProgramConfig, rundown model.Rundown, corners []config.CornerConfig, chars map[string]config.CharacterConfig) (model.Script, model.ScriptLines, *model.ProofreadResult, error)
}

// Synther synthesizes voice clips from a script.
type Synther interface {
	Run(ctx context.Context, scr model.Script, outDir string) (*model.ClipsMeta, error)
}

// Mixer produces an MP3 episode from clips and a script.
// Returns per-corner estimated durations (seconds) keyed by CornerID.
type Mixer interface {
	Run(ctx context.Context, scr model.Script, clips model.ClipsMeta, clipsDir, outPath string, meta model.EpisodeMeta) (map[string]float64, error)
}

// Options configures a single pipeline run.
type Options struct {
	OutDir        string
	GeneratedAt   time.Time           // zero value means time.Now().UTC()
	EpisodeNumber int                 // 0 means unknown (omitted from manifest)
	Casts         []model.RundownCast // confirmed cast for this episode; nil treated as empty
}

// Runner orchestrates the full gather→rundown→script→synth→summary→mix→manifest pipeline.
type Runner struct {
	Spec              *config.EpisodeSpec
	Config            *config.Config
	Gatherer          Gatherer
	Rundowner         Rundowner
	Scripter          Scripter
	Synther           Synther
	Mixer             Mixer
	ProgramSummarizer ProgramSummarizer // optional; if nil, program summary is omitted from manifest
	CornerSummarizer  CornerSummarizer  // optional; if nil, corner summaries are omitted from manifest
	ExcludedDedupKeys []string          // DedupKeys to exclude from feed collection (past-used articles)
}

// writeTimeline builds a model.Timeline from per-corner durations and writes it to 06_timeline.json.
func writeTimeline(layout fileio.EpisodeLayout, corners []config.CornerConfig, cornerDurations map[string]float64) error {
	timings := make([]model.CornerTiming, 0, len(corners))
	for _, c := range corners {
		timings = append(timings, model.CornerTiming{
			ID:          c.ID,
			DurationSec: cornerDurations[c.ID],
		})
	}
	return fileio.WriteJSON(layout.Timeline(), model.Timeline{Corners: timings})
}

// Run executes the full pipeline, writing intermediate files to <outDir>/intermediate/.
// Order: gather → rundown → script → synth → summary → mix → manifest.
// Summary runs before mix so that EpisodeTitle can be embedded in ID3 tags.
func (r *Runner) Run(ctx context.Context, opts Options) error {
	layout := fileio.EpisodeLayout{
		OutDir:        opts.OutDir,
		ProgramID:     r.Spec.Program.ID,
		EpisodeNumber: opts.EpisodeNumber,
	}

	if err := layout.EnsureDirs(); err != nil {
		return fmt.Errorf("create output dirs: %w", err)
	}

	articles, err := r.Gatherer.RunAll(ctx, r.Spec.Corners, r.ExcludedDedupKeys)
	if err != nil {
		return fmt.Errorf("gather: %w", err)
	}
	if err := fileio.WriteJSON(layout.Articles(), articles); err != nil {
		return err
	}

	rundown, err := r.Rundowner.Run(ctx, r.Spec.Corners, articles, model.NonNil(opts.Casts))
	if err != nil {
		return fmt.Errorf("rundown: %w", err)
	}
	if err := fileio.WriteJSON(layout.Rundown(), rundown); err != nil {
		return err
	}

	var chars map[string]config.CharacterConfig
	if r.Config != nil {
		chars = r.Config.Characters
	}

	scr, scriptLines, pr, err := r.Scripter.Generate(ctx, r.Spec.Program, rundown, r.Spec.Corners, chars)
	if err != nil {
		return fmt.Errorf("script: %w", err)
	}
	if err := fileio.WriteJSON(layout.Script(), scr); err != nil {
		return err
	}
	if err := fileio.WriteJSON(layout.Lines(), scriptLines); err != nil {
		return err
	}
	if pr != nil {
		if err := fileio.WriteJSON(layout.Proofread(), pr); err != nil {
			return err
		}
	}

	clips, err := r.Synther.Run(ctx, scr, layout.ClipsDir())
	if err != nil {
		return fmt.Errorf("synth: %w", err)
	}
	if clips == nil {
		clips = &model.ClipsMeta{Clips: make([]model.ClipMeta, 0)}
	}

	// Summary runs before mix: EpisodeTitle (from programSummary) is embedded in ID3 tags.
	generatedAt := opts.GeneratedAt
	if generatedAt.IsZero() {
		generatedAt = time.Now().UTC()
	}

	var programSummary model.ProgramSummary
	var cornerSummaries map[string]model.CornerSummary
	if r.ProgramSummarizer != nil || r.CornerSummarizer != nil {
		if r.ProgramSummarizer != nil {
			programSummary, err = r.ProgramSummarizer.Summarize(ctx, scriptLines)
			if err != nil {
				return fmt.Errorf("summarize program: %w", err)
			}
		}

		if r.CornerSummarizer != nil {
			cornerSummaries = make(map[string]model.CornerSummary, len(scriptLines.Corners))
			for _, cl := range scriptLines.Corners {
				cs, err := r.CornerSummarizer.SummarizeCorner(ctx, cl, r.Spec.CornerSummaryLength(cl.Title))
				if err != nil {
					return fmt.Errorf("summarize corner %s: %w", cl.Title, err)
				}
				cornerSummaries[cl.Title] = cs
			}
		}
	}

	episodeMeta := model.EpisodeMeta{
		Number:      opts.EpisodeNumber,
		Title:       programSummary.EpisodeTitle,
		GeneratedAt: generatedAt,
	}
	cornerDurations, err := r.Mixer.Run(ctx, scr, *clips, layout.ClipsDir(), layout.Episode(), episodeMeta)
	if err != nil {
		return fmt.Errorf("mix: %w", err)
	}

	if err := writeTimeline(layout, r.Spec.Corners, cornerDurations); err != nil {
		return fmt.Errorf("write timeline: %w", err)
	}

	m := manifest.Build(manifest.BuildParams{
		Program:           r.Spec.Program,
		Corners:           r.Spec.Corners,
		Rundown:           rundown,
		AudioFile:         fileio.EpisodeFileName(r.Spec.Program.ID, opts.EpisodeNumber),
		GeneratedAt:       generatedAt,
		Summary:           programSummary.Summary,
		CornerSummaries:   cornerSummaries,
		ConversationNotes: programSummary.ConversationNotes,
		EpisodeNumber:     opts.EpisodeNumber,
		EpisodeTitle:      programSummary.EpisodeTitle,
		Assets:            r.Spec.Assets,
		Characters:        chars,
		Lines:             &scriptLines,
		Script:            &scr,
		Clips:             clips,
		CornerDurations:   cornerDurations,
	})
	if err := fileio.WriteJSON(layout.Manifest(), m); err != nil {
		return fmt.Errorf("write manifest: %w", err)
	}

	return nil
}
