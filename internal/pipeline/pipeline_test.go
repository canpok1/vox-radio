package pipeline_test

import (
	"context"
	"errors"
	"os"
	"testing"

	"github.com/canpok1/vox-radio/internal/assemble"
	"github.com/canpok1/vox-radio/internal/config"
	"github.com/canpok1/vox-radio/internal/fileio"
	"github.com/canpok1/vox-radio/internal/model"
	"github.com/canpok1/vox-radio/internal/pipeline"
	"github.com/canpok1/vox-radio/internal/publish"
)

// --- stubs ---

type stubCollector struct {
	articles model.Articles
	err      error
	called   bool
}

func (s *stubCollector) Run(_ context.Context, _ config.FeedsConfig) (model.Articles, error) {
	s.called = true
	return s.articles, s.err
}

type stubScripter struct {
	script   model.Script
	err      error
	called   bool
	gotSlice []model.Article
}

func (s *stubScripter) Generate(_ context.Context, articles []model.Article, _ model.ShowConfig) (model.Script, error) {
	s.called = true
	s.gotSlice = articles
	return s.script, s.err
}

type stubSynther struct {
	clips       *model.ClipsMeta
	err         error
	called      bool
	capturedDir string
}

func (s *stubSynther) Run(_ context.Context, _ model.Script, outDir string) (*model.ClipsMeta, error) {
	s.called = true
	s.capturedDir = outDir
	return s.clips, s.err
}

type stubAssembler struct {
	result           *assemble.Result
	err              error
	called           bool
	capturedClipsDir string
	capturedOutPath  string
}

func (s *stubAssembler) Run(_ context.Context, _ model.Script, _ model.ClipsMeta, clipsDir, outPath string) (*assemble.Result, error) {
	s.called = true
	s.capturedClipsDir = clipsDir
	s.capturedOutPath = outPath
	return s.result, s.err
}

type stubPublisher struct {
	err             error
	called          bool
	capturedMp3Path string
}

func (s *stubPublisher) Run(_ context.Context, mp3Path string, _ publish.Options) error {
	s.called = true
	s.capturedMp3Path = mp3Path
	return s.err
}

type stubPruner struct {
	err    error
	called bool
}

func (s *stubPruner) Run(_ context.Context) error {
	s.called = true
	return s.err
}

// --- helpers ---

type stubs struct {
	col *stubCollector
	scr *stubScripter
	syn *stubSynther
	asm *stubAssembler
	pub *stubPublisher
	pru *stubPruner
}

func defaultStubs() stubs {
	return stubs{
		col: &stubCollector{articles: model.Articles{Articles: make([]model.Article, 0)}},
		scr: &stubScripter{script: model.Script{Segments: make([]model.ScriptSegment, 0)}},
		syn: &stubSynther{clips: &model.ClipsMeta{Clips: make([]model.ClipMeta, 0)}},
		asm: &stubAssembler{result: &assemble.Result{}},
		pub: &stubPublisher{},
		pru: &stubPruner{},
	}
}

func newRunner(s stubs) *pipeline.Runner {
	return &pipeline.Runner{
		Profile:   &config.Profile{},
		Collector: s.col,
		Scripter:  s.scr,
		Synther:   s.syn,
		Assembler: s.asm,
		Publisher: s.pub,
		Pruner:    s.pru,
	}
}

// --- tests ---

func TestRunner_Run_CallsAllSteps(t *testing.T) {
	outDir := t.TempDir()
	s := defaultStubs()

	if err := newRunner(s).Run(context.Background(), pipeline.Options{OutDir: outDir}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !s.col.called {
		t.Error("Collector.Run not called")
	}
	if !s.scr.called {
		t.Error("Scripter.Generate not called")
	}
	if !s.syn.called {
		t.Error("Synther.Run not called")
	}
	if !s.asm.called {
		t.Error("Assembler.Run not called")
	}
	if !s.pub.called {
		t.Error("Publisher.Run not called")
	}
	if !s.pru.called {
		t.Error("Pruner.Run not called")
	}
}

func TestRunner_Run_SavesIntermediateFiles(t *testing.T) {
	outDir := t.TempDir()

	if err := newRunner(defaultStubs()).Run(context.Background(), pipeline.Options{OutDir: outDir}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	for _, path := range []string{fileio.ArticlesPath(outDir), fileio.ScriptPath(outDir)} {
		if _, err := os.Stat(path); err != nil {
			t.Errorf("expected file %q to exist: %v", path, err)
		}
	}
}

func TestRunner_Run_UsesCorrectPaths(t *testing.T) {
	outDir := t.TempDir()
	s := defaultStubs()

	if err := newRunner(s).Run(context.Background(), pipeline.Options{OutDir: outDir}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if s.syn.capturedDir != fileio.ClipsDir(outDir) {
		t.Errorf("Synther.Run outDir = %q, want %q", s.syn.capturedDir, fileio.ClipsDir(outDir))
	}
	if s.asm.capturedClipsDir != fileio.ClipsDir(outDir) {
		t.Errorf("Assembler.Run clipsDir = %q, want %q", s.asm.capturedClipsDir, fileio.ClipsDir(outDir))
	}
	if s.asm.capturedOutPath != fileio.EpisodePath(outDir) {
		t.Errorf("Assembler.Run outPath = %q, want %q", s.asm.capturedOutPath, fileio.EpisodePath(outDir))
	}
	if s.pub.capturedMp3Path != fileio.EpisodePath(outDir) {
		t.Errorf("Publisher.Run mp3Path = %q, want %q", s.pub.capturedMp3Path, fileio.EpisodePath(outDir))
	}
}

func TestRunner_Run_PassesArticlesToScripter(t *testing.T) {
	outDir := t.TempDir()
	s := defaultStubs()
	article := model.Article{URL: "http://example.com", Title: "Test", Body: "body"}
	s.col.articles = model.Articles{Articles: []model.Article{article}}

	if err := newRunner(s).Run(context.Background(), pipeline.Options{OutDir: outDir}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(s.scr.gotSlice) != 1 || s.scr.gotSlice[0].URL != article.URL {
		t.Errorf("Scripter.Generate received %v, want [%v]", s.scr.gotSlice, article)
	}
}

func TestRunner_Run_CollectError(t *testing.T) {
	outDir := t.TempDir()
	s := defaultStubs()
	s.col.err = errors.New("network error")

	if err := newRunner(s).Run(context.Background(), pipeline.Options{OutDir: outDir}); err == nil {
		t.Fatal("expected error from Collector, got nil")
	}
	if s.scr.called {
		t.Error("Scripter should not be called after Collector error")
	}
}

func TestRunner_Run_ScriptError(t *testing.T) {
	outDir := t.TempDir()
	s := defaultStubs()
	s.scr.err = errors.New("llm error")

	if err := newRunner(s).Run(context.Background(), pipeline.Options{OutDir: outDir}); err == nil {
		t.Fatal("expected error from Scripter, got nil")
	}
	if s.syn.called {
		t.Error("Synther should not be called after Scripter error")
	}
}

func TestRunner_Run_SynthError(t *testing.T) {
	outDir := t.TempDir()
	s := defaultStubs()
	s.syn.err = errors.New("voicevox error")
	s.syn.clips = nil

	if err := newRunner(s).Run(context.Background(), pipeline.Options{OutDir: outDir}); err == nil {
		t.Fatal("expected error from Synther, got nil")
	}
	if s.asm.called {
		t.Error("Assembler should not be called after Synther error")
	}
}

func TestRunner_Run_AssembleError(t *testing.T) {
	outDir := t.TempDir()
	s := defaultStubs()
	s.asm.err = errors.New("ffmpeg error")

	if err := newRunner(s).Run(context.Background(), pipeline.Options{OutDir: outDir}); err == nil {
		t.Fatal("expected error from Assembler, got nil")
	}
	if s.pub.called {
		t.Error("Publisher should not be called after Assembler error")
	}
}

func TestRunner_Run_PublishError(t *testing.T) {
	outDir := t.TempDir()
	s := defaultStubs()
	s.pub.err = errors.New("publish error")

	if err := newRunner(s).Run(context.Background(), pipeline.Options{OutDir: outDir}); err == nil {
		t.Fatal("expected error from Publisher, got nil")
	}
	if s.pru.called {
		t.Error("Pruner should not be called after Publisher error")
	}
}

func TestRunner_Run_PruneError(t *testing.T) {
	outDir := t.TempDir()
	s := defaultStubs()
	s.pru.err = errors.New("prune error")

	if err := newRunner(s).Run(context.Background(), pipeline.Options{OutDir: outDir}); err == nil {
		t.Fatal("expected error from Pruner, got nil")
	}
}
