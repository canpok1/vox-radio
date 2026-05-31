package pipeline

import (
	"context"
	"fmt"

	"github.com/canpok1/vox-radio/internal/assemble"
	"github.com/canpok1/vox-radio/internal/config"
	"github.com/canpok1/vox-radio/internal/fileio"
	"github.com/canpok1/vox-radio/internal/model"
	"github.com/canpok1/vox-radio/internal/publish"
)

// Collector gathers articles from configured feeds.
type Collector interface {
	Run(ctx context.Context, cfg config.FeedsConfig) (model.Articles, error)
}

// Scripter generates a radio script from articles.
type Scripter interface {
	Generate(ctx context.Context, articles []model.Article, corners []config.CornerConfig, chars map[string]config.CharacterConfig) (model.Script, error)
}

// Synther synthesizes voice clips from a script.
type Synther interface {
	Run(ctx context.Context, scr model.Script, outDir string) (*model.ClipsMeta, error)
}

// Assembler produces an MP3 episode from clips and a script.
type Assembler interface {
	Run(ctx context.Context, scr model.Script, clips model.ClipsMeta, clipsDir, outPath string) (*assemble.Result, error)
}

// Publisher publishes the episode to a hosting backend.
type Publisher interface {
	Run(ctx context.Context, mp3Path string, opts publish.Options) error
}

// Pruner removes old episodes beyond the configured keep limit.
type Pruner interface {
	Run(ctx context.Context) error
}

// Options configures a single pipeline run.
type Options struct {
	OutDir      string
	PublishOpts publish.Options
}

// Runner orchestrates the full collect→script→synth→assemble→publish→prune pipeline.
type Runner struct {
	Profile   *config.Profile
	Config    *config.Config
	Collector Collector
	Scripter  Scripter
	Synther   Synther
	Assembler Assembler
	Publisher Publisher
	Pruner    Pruner
}

// Run executes the full pipeline, writing intermediate files to <outDir>/intermediate/.
func (r *Runner) Run(ctx context.Context, opts Options) error {
	outDir := opts.OutDir

	if err := fileio.EnsureOutputDirs(outDir); err != nil {
		return fmt.Errorf("create output dirs: %w", err)
	}

	articles, err := r.Collector.Run(ctx, config.FeedsConfig{
		Feeds:    r.Profile.Feeds,
		Articles: r.Profile.Articles,
	})
	if err != nil {
		return fmt.Errorf("collect: %w", err)
	}
	if err := fileio.WriteJSON(fileio.ArticlesPath(outDir), articles); err != nil {
		return err
	}

	var chars map[string]config.CharacterConfig
	if r.Config != nil {
		chars = r.Config.Characters
	}

	scr, err := r.Scripter.Generate(ctx, articles.Articles, r.Profile.Corners, chars)
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

	if err := r.Publisher.Run(ctx, fileio.EpisodePath(outDir), opts.PublishOpts); err != nil {
		return fmt.Errorf("publish: %w", err)
	}

	if err := r.Pruner.Run(ctx); err != nil {
		return fmt.Errorf("prune: %w", err)
	}

	return nil
}
