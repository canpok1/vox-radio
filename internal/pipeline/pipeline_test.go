package pipeline_test

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"strings"
	"testing"

	"github.com/canpok1/vox-radio/internal/assemble"
	"github.com/canpok1/vox-radio/internal/config"
	"github.com/canpok1/vox-radio/internal/fileio"
	"github.com/canpok1/vox-radio/internal/model"
	"github.com/canpok1/vox-radio/internal/pipeline"
)

// --- stubs ---

type stubCollector struct {
	articles model.Articles
	err      error
	called   bool
}

func (s *stubCollector) RunAll(_ context.Context, _ []config.CornerConfig) (model.Articles, error) {
	s.called = true
	return s.articles, s.err
}

type stubScripter struct {
	script      model.Script
	err         error
	called      bool
	gotArticles model.Articles
}

func (s *stubScripter) Generate(_ context.Context, articles model.Articles, _ []config.CornerConfig, _ map[string]config.CharacterConfig) (model.Script, error) {
	s.called = true
	s.gotArticles = articles
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

type stubProgramSummarizer struct {
	summary string
	err     error
	called  bool
}

func (s *stubProgramSummarizer) Summarize(_ context.Context, _ model.Script) (string, error) {
	s.called = true
	return s.summary, s.err
}

// --- helpers ---

type stubs struct {
	col *stubCollector
	scr *stubScripter
	syn *stubSynther
	asm *stubAssembler
	sum *stubProgramSummarizer
}

func defaultStubs() stubs {
	return stubs{
		col: &stubCollector{articles: model.Articles{Corners: make([]model.CornerArticles, 0)}},
		scr: &stubScripter{script: model.Script{Segments: make([]model.ScriptSegment, 0)}},
		syn: &stubSynther{clips: &model.ClipsMeta{Clips: make([]model.ClipMeta, 0)}},
		asm: &stubAssembler{result: &assemble.Result{}},
	}
}

func newRunner(s stubs) *pipeline.Runner {
	r := &pipeline.Runner{
		Profile:   &config.Profile{},
		Collector: s.col,
		Scripter:  s.scr,
		Synther:   s.syn,
		Assembler: s.asm,
	}
	if s.sum != nil {
		r.ProgramSummarizer = s.sum
	}
	return r
}

// --- tests ---

func TestRunner_Run_CallsAllSteps(t *testing.T) {
	outDir := t.TempDir()
	s := defaultStubs()

	if err := newRunner(s).Run(context.Background(), pipeline.Options{OutDir: outDir}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !s.col.called {
		t.Error("Collector.RunAll not called")
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
}

func TestRunner_Run_SavesIntermediateFiles(t *testing.T) {
	outDir := t.TempDir()

	if err := newRunner(defaultStubs()).Run(context.Background(), pipeline.Options{OutDir: outDir}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	for _, path := range []string{fileio.ArticlesPath(outDir), fileio.ScriptPath(outDir), fileio.ManifestPath(outDir)} {
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
}

func TestRunner_Run_PassesArticlesToScripter(t *testing.T) {
	outDir := t.TempDir()
	s := defaultStubs()
	article := model.Article{URL: "http://example.com", Title: "Test", Body: "body"}
	s.col.articles = model.Articles{
		Corners: []model.CornerArticles{
			{CornerTitle: "テストコーナー", Articles: []model.Article{article}},
		},
	}

	if err := newRunner(s).Run(context.Background(), pipeline.Options{OutDir: outDir}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(s.scr.gotArticles.Corners) != 1 {
		t.Fatalf("Scripter.Generate received %d corners, want 1", len(s.scr.gotArticles.Corners))
	}
	got := s.scr.gotArticles.Corners[0]
	if len(got.Articles) != 1 || got.Articles[0].URL != article.URL {
		t.Errorf("Scripter.Generate corner articles = %v, want [%v]", got.Articles, article)
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
}

func TestRunner_Run_CallsProgramSummarizer(t *testing.T) {
	outDir := t.TempDir()
	s := defaultStubs()
	s.sum = &stubProgramSummarizer{summary: "今回は技術ニュースを紹介しました。"}

	if err := newRunner(s).Run(context.Background(), pipeline.Options{OutDir: outDir}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !s.sum.called {
		t.Error("ProgramSummarizer.Summarize not called")
	}
}

func TestRunner_Run_ManifestIncludesSummary(t *testing.T) {
	outDir := t.TempDir()
	s := defaultStubs()
	wantSummary := "今回は技術ニュースを紹介しました。"
	s.sum = &stubProgramSummarizer{summary: wantSummary}

	if err := newRunner(s).Run(context.Background(), pipeline.Options{OutDir: outDir}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	manifestPath := fileio.ManifestPath(outDir)
	data, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatalf("read manifest: %v", err)
	}
	if !strings.Contains(string(data), wantSummary) {
		t.Errorf("manifest should contain summary %q, got: %s", wantSummary, data)
	}
}

func TestRunner_Run_SkipsSummaryWhenSummarizerIsNil(t *testing.T) {
	outDir := t.TempDir()
	s := defaultStubs()
	// s.sum is nil — no summarizer set

	if err := newRunner(s).Run(context.Background(), pipeline.Options{OutDir: outDir}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	manifestPath := fileio.ManifestPath(outDir)
	if _, err := os.Stat(manifestPath); err != nil {
		t.Fatalf("manifest should exist: %v", err)
	}
}

func TestRunner_Run_ProgramSummarizerError(t *testing.T) {
	outDir := t.TempDir()
	s := defaultStubs()
	s.sum = &stubProgramSummarizer{err: errors.New("llm error")}

	if err := newRunner(s).Run(context.Background(), pipeline.Options{OutDir: outDir}); err == nil {
		t.Fatal("expected error from ProgramSummarizer, got nil")
	}
}

func TestRunner_Run_LogsManifestStep(t *testing.T) {
	outDir := t.TempDir()
	s := defaultStubs()

	var buf strings.Builder
	logger := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelInfo}))

	r := newRunner(s)
	r.Logger = logger

	if err := r.Run(context.Background(), pipeline.Options{OutDir: outDir}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	logs := buf.String()
	if !strings.Contains(logs, "manifest") {
		t.Errorf("should log manifest step: %q", logs)
	}
}
