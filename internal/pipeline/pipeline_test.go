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

type stubRundowner struct {
	rundown model.Rundown
	err     error
	called  bool
}

func (s *stubRundowner) Run(_ context.Context, _ []config.CornerConfig, _ model.Articles) (model.Rundown, error) {
	s.called = true
	return s.rundown, s.err
}

type stubScripter struct {
	script     model.Script
	err        error
	called     bool
	gotRundown model.Rundown
}

func (s *stubScripter) Generate(_ context.Context, _ config.ProgramConfig, rundown model.Rundown, _ []config.CornerConfig, _ map[string]config.CharacterConfig) (model.Script, error) {
	s.called = true
	s.gotRundown = rundown
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
	notes   []model.ConversationNote
	err     error
	called  bool
}

func (s *stubProgramSummarizer) Summarize(_ context.Context, _ model.ScriptLines) (model.ProgramSummary, error) {
	s.called = true
	notes := s.notes
	if notes == nil {
		notes = make([]model.ConversationNote, 0)
	}
	return model.ProgramSummary{Summary: s.summary, ConversationNotes: notes}, s.err
}

type stubCornerSummarizer struct {
	result model.CornerSummary
	err    error
	called bool
}

func (s *stubCornerSummarizer) SummarizeCorner(_ context.Context, _ model.CornerLines, _ int) (model.CornerSummary, error) {
	s.called = true
	return s.result, s.err
}

// --- helpers ---

type stubs struct {
	col  *stubCollector
	rnd  *stubRundowner
	scr  *stubScripter
	syn  *stubSynther
	asm  *stubAssembler
	sum  *stubProgramSummarizer
	csum *stubCornerSummarizer
}

func defaultStubs() stubs {
	return stubs{
		col: &stubCollector{articles: model.Articles{Corners: make([]model.CornerArticles, 0)}},
		rnd: &stubRundowner{rundown: model.Rundown{Corners: make([]model.RundownCorner, 0)}},
		scr: &stubScripter{script: model.Script{Segments: make([]model.ScriptSegment, 0)}},
		syn: &stubSynther{clips: &model.ClipsMeta{Clips: make([]model.ClipMeta, 0)}},
		asm: &stubAssembler{result: &assemble.Result{}},
	}
}

func newRunner(s stubs) *pipeline.Runner {
	r := &pipeline.Runner{
		Profile:   &config.Profile{},
		Collector: s.col,
		Rundowner: s.rnd,
		Scripter:  s.scr,
		Synther:   s.syn,
		Assembler: s.asm,
	}
	if s.sum != nil {
		r.ProgramSummarizer = s.sum
	}
	if s.csum != nil {
		r.CornerSummarizer = s.csum
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
	if !s.rnd.called {
		t.Error("Rundowner.Run not called")
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

	for _, path := range []string{fileio.ArticlesPath(outDir), fileio.RundownPath(outDir), fileio.ScriptPath(outDir), fileio.ManifestPath(outDir)} {
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

func TestRunner_Run_PassesRundownToScripter(t *testing.T) {
	outDir := t.TempDir()
	s := defaultStubs()
	wantRundown := model.Rundown{
		Corners: []model.RundownCorner{
			{
				Title: "テストコーナー",
				Flow:  "記事を順に紹介",
				Articles: []model.RundownArticle{
					{URL: "http://example.com", Title: "記事", Summary: "要約", Points: []string{"p1"}},
				},
			},
		},
	}
	s.rnd.rundown = wantRundown

	if err := newRunner(s).Run(context.Background(), pipeline.Options{OutDir: outDir}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(s.scr.gotRundown.Corners) != 1 {
		t.Fatalf("Scripter.Generate received %d corners, want 1", len(s.scr.gotRundown.Corners))
	}
	got := s.scr.gotRundown.Corners[0]
	if got.Title != "テストコーナー" {
		t.Errorf("Rundown corner title: got %q, want %q", got.Title, "テストコーナー")
	}
}

func TestRunner_Run_CollectError(t *testing.T) {
	outDir := t.TempDir()
	s := defaultStubs()
	s.col.err = errors.New("network error")

	if err := newRunner(s).Run(context.Background(), pipeline.Options{OutDir: outDir}); err == nil {
		t.Fatal("expected error from Collector, got nil")
	}
	if s.rnd.called {
		t.Error("Rundowner should not be called after Collector error")
	}
}

func TestRunner_Run_RundownError(t *testing.T) {
	outDir := t.TempDir()
	s := defaultStubs()
	s.rnd.err = errors.New("llm error")

	if err := newRunner(s).Run(context.Background(), pipeline.Options{OutDir: outDir}); err == nil {
		t.Fatal("expected error from Rundowner, got nil")
	}
	if s.scr.called {
		t.Error("Scripter should not be called after Rundowner error")
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

	writeScriptLines(t, outDir, model.ScriptLines{Corners: []model.CornerLines{}})

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

	writeScriptLines(t, outDir, model.ScriptLines{Corners: []model.CornerLines{}})

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

	writeScriptLines(t, outDir, model.ScriptLines{Corners: []model.CornerLines{}})

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

func writeScriptLines(t *testing.T, outDir string, sl model.ScriptLines) {
	t.Helper()
	if err := fileio.WriteJSON(fileio.LinesPath(outDir), sl); err != nil {
		t.Fatalf("write script lines: %v", err)
	}
}

func TestRunner_Run_CallsCornerSummarizer(t *testing.T) {
	outDir := t.TempDir()
	s := defaultStubs()
	s.csum = &stubCornerSummarizer{result: model.CornerSummary{Summary: "要約", Points: []string{"p1"}}}

	writeScriptLines(t, outDir, model.ScriptLines{
		Corners: []model.CornerLines{
			{Title: "テストコーナー", Lines: []model.Line{{Text: "テスト"}}},
		},
	})

	if err := newRunner(s).Run(context.Background(), pipeline.Options{OutDir: outDir}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !s.csum.called {
		t.Error("CornerSummarizer.SummarizeCorner not called")
	}
}

func TestRunner_Run_ManifestIncludesCornerSummary(t *testing.T) {
	outDir := t.TempDir()
	s := defaultStubs()
	s.csum = &stubCornerSummarizer{result: model.CornerSummary{Summary: "コーナー要約テスト", Points: []string{"要点A"}}}

	writeScriptLines(t, outDir, model.ScriptLines{
		Corners: []model.CornerLines{
			{Title: "テストコーナー", Lines: []model.Line{{Text: "テスト"}}},
		},
	})

	r := newRunner(s)
	r.Profile = &config.Profile{
		Corners: []config.CornerConfig{{Title: "テストコーナー"}},
	}

	if err := r.Run(context.Background(), pipeline.Options{OutDir: outDir}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	data, err := os.ReadFile(fileio.ManifestPath(outDir))
	if err != nil {
		t.Fatalf("read manifest: %v", err)
	}
	if !strings.Contains(string(data), "コーナー要約テスト") {
		t.Errorf("manifest should contain corner summary, got: %s", data)
	}
}

func TestRunner_Run_SkipsCornerSummaryWhenSummarizerIsNil(t *testing.T) {
	outDir := t.TempDir()
	s := defaultStubs()

	if err := newRunner(s).Run(context.Background(), pipeline.Options{OutDir: outDir}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if _, err := os.Stat(fileio.ManifestPath(outDir)); err != nil {
		t.Fatalf("manifest should exist: %v", err)
	}
}

func TestRunner_Run_CornerSummarizerError(t *testing.T) {
	outDir := t.TempDir()
	s := defaultStubs()
	s.csum = &stubCornerSummarizer{err: errors.New("llm error")}

	writeScriptLines(t, outDir, model.ScriptLines{
		Corners: []model.CornerLines{
			{Title: "コーナー", Lines: []model.Line{{Text: "テスト"}}},
		},
	})

	if err := newRunner(s).Run(context.Background(), pipeline.Options{OutDir: outDir}); err == nil {
		t.Fatal("expected error from CornerSummarizer, got nil")
	}
}

func TestRunner_Run_LogsSummaryStep(t *testing.T) {
	outDir := t.TempDir()
	s := defaultStubs()
	s.sum = &stubProgramSummarizer{summary: "要約テスト"}
	s.csum = &stubCornerSummarizer{result: model.CornerSummary{Summary: "コーナー要約", Points: []string{"p1"}}}

	writeScriptLines(t, outDir, model.ScriptLines{
		Corners: []model.CornerLines{
			{Title: "テストコーナー", Lines: []model.Line{{Text: "テスト"}}},
		},
	})

	var buf strings.Builder
	logger := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelInfo}))

	r := newRunner(s)
	r.Logger = logger

	if err := r.Run(context.Background(), pipeline.Options{OutDir: outDir}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	logs := buf.String()
	if !strings.Contains(logs, "summary") {
		t.Errorf("should log summary step: %q", logs)
	}
	if !strings.Contains(logs, "要約中") {
		t.Errorf("should log 要約中: %q", logs)
	}
}

func TestRunner_Run_NoSummaryLogWhenSummarizersNil(t *testing.T) {
	outDir := t.TempDir()
	s := defaultStubs() // both sum and csum are nil

	var buf strings.Builder
	logger := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelInfo}))

	r := newRunner(s)
	r.Logger = logger

	if err := r.Run(context.Background(), pipeline.Options{OutDir: outDir}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	logs := buf.String()
	if strings.Contains(logs, "summary") {
		t.Errorf("should not log summary step when summarizers are nil: %q", logs)
	}
}

func TestRunner_Run_ManifestIncludesConversationNotes(t *testing.T) {
	outDir := t.TempDir()
	s := defaultStubs()
	s.sum = &stubProgramSummarizer{
		summary: "要約",
		notes: []model.ConversationNote{
			{Category: "近況", CharacterIDs: []string{"zundamon"}, Note: "カフェにハマっている"},
		},
	}

	writeScriptLines(t, outDir, model.ScriptLines{Corners: []model.CornerLines{}})

	if err := newRunner(s).Run(context.Background(), pipeline.Options{OutDir: outDir}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	manifestPath := fileio.ManifestPath(outDir)
	data, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatalf("read manifest: %v", err)
	}
	if !strings.Contains(string(data), "カフェにハマっている") {
		t.Errorf("manifest should contain conversation note, got: %s", string(data))
	}
	if !strings.Contains(string(data), `"conversation_notes"`) {
		t.Errorf("manifest should contain conversation_notes field, got: %s", string(data))
	}
}
