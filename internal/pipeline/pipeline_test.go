package pipeline_test

import (
	"context"
	"errors"
	"os"
	"strings"
	"testing"

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

func (s *stubCollector) RunAll(_ context.Context, _ []config.CornerConfig, _ []string) (model.Articles, error) {
	s.called = true
	return s.articles, s.err
}

type stubRundowner struct {
	rundown model.Rundown
	err     error
	called  bool
}

func (s *stubRundowner) Run(_ context.Context, _ []config.CornerConfig, _ model.Articles, casts []model.RundownCast) (model.Rundown, error) {
	s.called = true
	rd := s.rundown
	if casts == nil {
		casts = make([]model.RundownCast, 0)
	}
	rd.Casts = casts
	return rd, s.err
}

type stubScripter struct {
	script     model.Script
	lines      model.ScriptLines
	proofread  *model.ProofreadResult
	err        error
	called     bool
	gotRundown model.Rundown
}

func (s *stubScripter) Generate(_ context.Context, _ config.ProgramConfig, rundown model.Rundown, _ []config.CornerConfig, _ map[string]config.CharacterConfig) (model.Script, model.ScriptLines, *model.ProofreadResult, error) {
	s.called = true
	s.gotRundown = rundown
	return s.script, s.lines, s.proofread, s.err
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
	err              error
	cornerDurations  map[string]float64
	called           bool
	capturedClipsDir string
	capturedOutPath  string
	capturedMeta     model.EpisodeMeta
}

func (s *stubAssembler) Run(_ context.Context, _ model.Script, _ model.ClipsMeta, clipsDir, outPath string, meta model.EpisodeMeta) (map[string]float64, error) {
	s.called = true
	s.capturedClipsDir = clipsDir
	s.capturedOutPath = outPath
	s.capturedMeta = meta
	return s.cornerDurations, s.err
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

// outLayout returns the EpisodeLayout the Runner builds for tests using the
// default stubs (empty program ID, episode number 0).
func outLayout(outDir string) fileio.EpisodeLayout {
	return fileio.EpisodeLayout{OutDir: outDir}
}

func mustReadManifest(t *testing.T, outDir string) string {
	t.Helper()
	data, err := os.ReadFile(outLayout(outDir).Manifest())
	if err != nil {
		t.Fatalf("read manifest: %v", err)
	}
	return string(data)
}

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
		asm: &stubAssembler{},
	}
}

func newRunner(s stubs) *pipeline.Runner {
	r := &pipeline.Runner{
		Spec:      &config.EpisodeSpec{},
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

	l := outLayout(outDir)
	for _, path := range []string{l.Articles(), l.Rundown(), l.Script(), l.Lines(), l.Manifest()} {
		if _, err := os.Stat(path); err != nil {
			t.Errorf("expected file %q to exist: %v", path, err)
		}
	}
}

func TestRunner_Run_UsesCorrectPaths(t *testing.T) {
	outDir := t.TempDir()
	s := defaultStubs()

	r := newRunner(s)
	r.Spec.Program.ID = "test-prog"

	if err := r.Run(context.Background(), pipeline.Options{OutDir: outDir, EpisodeNumber: 5}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	l := fileio.EpisodeLayout{OutDir: outDir, ProgramID: "test-prog", EpisodeNumber: 5}
	if s.syn.capturedDir != l.ClipsDir() {
		t.Errorf("Synther.Run outDir = %q, want %q", s.syn.capturedDir, l.ClipsDir())
	}
	if s.asm.capturedClipsDir != l.ClipsDir() {
		t.Errorf("Assembler.Run clipsDir = %q, want %q", s.asm.capturedClipsDir, l.ClipsDir())
	}
	if s.asm.capturedOutPath != l.Episode() {
		t.Errorf("Assembler.Run outPath = %q, want %q", s.asm.capturedOutPath, l.Episode())
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
					{URL: "http://example.com", Title: "記事", Body: "記事の本文", Points: []string{"p1"}},
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
	s.scr.lines = model.ScriptLines{Corners: make([]model.CornerLines, 0)}
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
	s.scr.lines = model.ScriptLines{Corners: make([]model.CornerLines, 0)}
	s.sum = &stubProgramSummarizer{summary: wantSummary}

	if err := newRunner(s).Run(context.Background(), pipeline.Options{OutDir: outDir}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got := mustReadManifest(t, outDir)
	if !strings.Contains(got, wantSummary) {
		t.Errorf("manifest should contain summary %q, got: %s", wantSummary, got)
	}
}

func TestRunner_Run_SkipsSummaryWhenSummarizerIsNil(t *testing.T) {
	outDir := t.TempDir()
	s := defaultStubs()

	if err := newRunner(s).Run(context.Background(), pipeline.Options{OutDir: outDir}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	manifestPath := outLayout(outDir).Manifest()
	if _, err := os.Stat(manifestPath); err != nil {
		t.Fatalf("manifest should exist: %v", err)
	}
}

func TestRunner_Run_ProgramSummarizerError(t *testing.T) {
	outDir := t.TempDir()
	s := defaultStubs()
	s.scr.lines = model.ScriptLines{Corners: make([]model.CornerLines, 0)}
	s.sum = &stubProgramSummarizer{err: errors.New("llm error")}

	if err := newRunner(s).Run(context.Background(), pipeline.Options{OutDir: outDir}); err == nil {
		t.Fatal("expected error from ProgramSummarizer, got nil")
	}
}

func TestRunner_Run_CallsCornerSummarizer(t *testing.T) {
	outDir := t.TempDir()
	s := defaultStubs()
	s.scr.lines = model.ScriptLines{
		Corners: []model.CornerLines{
			{Title: "テストコーナー", Lines: []model.Line{{Text: "テスト"}}},
		},
	}
	s.csum = &stubCornerSummarizer{result: model.CornerSummary{Summary: "要約", Points: []string{"p1"}}}

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
	s.scr.lines = model.ScriptLines{
		Corners: []model.CornerLines{
			{Title: "テストコーナー", Lines: []model.Line{{Text: "テスト"}}},
		},
	}
	s.csum = &stubCornerSummarizer{result: model.CornerSummary{Summary: "コーナー要約テスト", Points: []string{"要点A"}}}

	r := newRunner(s)
	r.Spec = &config.EpisodeSpec{
		Corners: []config.CornerConfig{{Title: "テストコーナー"}},
	}

	if err := r.Run(context.Background(), pipeline.Options{OutDir: outDir}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got := mustReadManifest(t, outDir)
	if !strings.Contains(got, "コーナー要約テスト") {
		t.Errorf("manifest should contain corner summary, got: %s", got)
	}
}

func TestRunner_Run_SkipsCornerSummaryWhenSummarizerIsNil(t *testing.T) {
	outDir := t.TempDir()
	s := defaultStubs()

	if err := newRunner(s).Run(context.Background(), pipeline.Options{OutDir: outDir}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if _, err := os.Stat(outLayout(outDir).Manifest()); err != nil {
		t.Fatalf("manifest should exist: %v", err)
	}
}

func TestRunner_Run_CornerSummarizerError(t *testing.T) {
	outDir := t.TempDir()
	s := defaultStubs()
	s.scr.lines = model.ScriptLines{
		Corners: []model.CornerLines{
			{Title: "コーナー", Lines: []model.Line{{Text: "テスト"}}},
		},
	}
	s.csum = &stubCornerSummarizer{err: errors.New("llm error")}

	if err := newRunner(s).Run(context.Background(), pipeline.Options{OutDir: outDir}); err == nil {
		t.Fatal("expected error from CornerSummarizer, got nil")
	}
}

func TestRunner_Run_CastsWrittenToRundown(t *testing.T) {
	outDir := t.TempDir()
	s := defaultStubs()
	casts := []model.RundownCast{
		{CharacterID: "metan", Role: "解説ゲスト", Type: "guest"},
	}

	if err := newRunner(s).Run(context.Background(), pipeline.Options{OutDir: outDir, Casts: casts}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// rundown.json に Casts が書き込まれていること
	var rd model.Rundown
	if err := fileio.ReadJSON(outLayout(outDir).Rundown(), &rd); err != nil {
		t.Fatalf("read rundown: %v", err)
	}
	if len(rd.Casts) != 1 || rd.Casts[0].CharacterID != "metan" {
		t.Errorf("unexpected casts in rundown: %+v", rd.Casts)
	}
}

func TestRunner_Run_NoCastsRundownHasEmptySlice(t *testing.T) {
	outDir := t.TempDir()
	s := defaultStubs()

	if err := newRunner(s).Run(context.Background(), pipeline.Options{OutDir: outDir}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var rd model.Rundown
	if err := fileio.ReadJSON(outLayout(outDir).Rundown(), &rd); err != nil {
		t.Fatalf("read rundown: %v", err)
	}
	// Casts が nil でなく空スライスであること（JSON で null になるのを防ぐ）
	if rd.Casts == nil {
		t.Error("rundown.Casts should be non-nil empty slice, not nil")
	}
	if len(rd.Casts) != 0 {
		t.Errorf("expected 0 casts, got %d", len(rd.Casts))
	}
}

func TestRunner_Run_ManifestIncludesConversationNotes(t *testing.T) {
	outDir := t.TempDir()
	s := defaultStubs()
	s.scr.lines = model.ScriptLines{Corners: make([]model.CornerLines, 0)}
	s.sum = &stubProgramSummarizer{
		summary: "要約",
		notes: []model.ConversationNote{
			{Category: "近況", CharacterIDs: []string{"zundamon"}, Note: "カフェにハマっている"},
		},
	}

	if err := newRunner(s).Run(context.Background(), pipeline.Options{OutDir: outDir}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got := mustReadManifest(t, outDir)
	if !strings.Contains(got, "カフェにハマっている") {
		t.Errorf("manifest should contain conversation note, got: %s", got)
	}
	if !strings.Contains(got, `"conversation_notes"`) {
		t.Errorf("manifest should contain conversation_notes field, got: %s", got)
	}
}

func TestRunner_Run_WritesProofreadFile_WhenProofreadResultIsNotNil(t *testing.T) {
	outDir := t.TempDir()
	s := defaultStubs()
	s.scr.proofread = &model.ProofreadResult{
		Corrections: []model.ProofreadCorrection{
			{CornerIndex: 0, LineIndex: 0, Before: "誤字", After: "正字"},
		},
	}

	if err := newRunner(s).Run(context.Background(), pipeline.Options{OutDir: outDir}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	proofreadPath := outLayout(outDir).Proofread()
	if _, err := os.Stat(proofreadPath); err != nil {
		t.Errorf("expected proofread file %q to exist: %v", proofreadPath, err)
	}
}

func TestRunner_Run_SkipsProofreadFile_WhenProofreadResultIsNil(t *testing.T) {
	outDir := t.TempDir()
	s := defaultStubs()
	// s.scr.proofread is nil by default

	if err := newRunner(s).Run(context.Background(), pipeline.Options{OutDir: outDir}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	proofreadPath := outLayout(outDir).Proofread()
	if _, err := os.Stat(proofreadPath); err == nil {
		t.Errorf("proofread file should not exist when ProofreadResult is nil, but found %q", proofreadPath)
	}
}

func TestRunner_Run_SummaryBeforeAssemble(t *testing.T) {
	outDir := t.TempDir()
	s := defaultStubs()
	s.scr.lines = model.ScriptLines{Corners: make([]model.CornerLines, 0)}
	s.sum = &stubProgramSummarizer{err: errors.New("summary failed")}

	// When ProgramSummarizer fails, Assembler must NOT be called (summary runs before assemble).
	if err := newRunner(s).Run(context.Background(), pipeline.Options{OutDir: outDir}); err == nil {
		t.Fatal("expected error when ProgramSummarizer fails")
	}
	if s.asm.called {
		t.Error("Assembler should NOT be called when ProgramSummarizer fails (summary runs before assemble)")
	}
}

func TestRunner_Run_ManifestIncludesAssetCredits(t *testing.T) {
	outDir := t.TempDir()
	s := defaultStubs()
	s.scr.lines = model.ScriptLines{
		Corners: []model.CornerLines{
			{Title: "コーナー1", BGM: "bgm1", Lines: []model.Line{{Text: "テスト"}}},
		},
	}

	r := newRunner(s)
	r.Spec.Assets = config.AssetsConfig{
		BGM: map[string]config.BGMEntry{"bgm1": {Credit: "OtoLogic / CC BY 4.0"}},
	}

	if err := r.Run(context.Background(), pipeline.Options{OutDir: outDir}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got := mustReadManifest(t, outDir)
	if !strings.Contains(got, "OtoLogic / CC BY 4.0") {
		t.Errorf("manifest should contain asset credit, got: %s", got)
	}
}

func TestRunner_Run_ManifestIncludesCharacterCredits(t *testing.T) {
	outDir := t.TempDir()
	s := defaultStubs()
	casts := []model.RundownCast{
		{CharacterID: "zundamon", Role: "MC", Type: "mc"},
	}

	r := newRunner(s)
	r.Config = &config.Config{
		Characters: map[string]config.CharacterConfig{
			"zundamon": {Credit: "VOICEVOX:ずんだもん"},
		},
	}

	if err := r.Run(context.Background(), pipeline.Options{OutDir: outDir, Casts: casts}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got := mustReadManifest(t, outDir)
	if !strings.Contains(got, "VOICEVOX:ずんだもん") {
		t.Errorf("manifest should contain character credit, got: %s", got)
	}
}

func TestRunner_Run_WritesTimeline(t *testing.T) {
	outDir := t.TempDir()
	s := defaultStubs()
	s.asm.cornerDurations = map[string]float64{"op": 12.5, "tech": 30.0}

	r := newRunner(s)
	r.Spec = &config.EpisodeSpec{
		Corners: []config.CornerConfig{
			{ID: "op", Title: "OP"},
			{ID: "tech", Title: "技術"},
		},
	}

	if err := r.Run(context.Background(), pipeline.Options{OutDir: outDir}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	timelinePath := outLayout(outDir).Timeline()
	if _, err := os.Stat(timelinePath); err != nil {
		t.Fatalf("06_timeline.json should exist: %v", err)
	}

	var tl model.Timeline
	if err := fileio.ReadJSON(timelinePath, &tl); err != nil {
		t.Fatalf("read timeline: %v", err)
	}
	if len(tl.Corners) != 2 {
		t.Fatalf("Corners: got %d, want 2", len(tl.Corners))
	}
	if tl.Corners[0].ID != "op" || tl.Corners[0].DurationSec != 12.5 {
		t.Errorf("Corners[0]: got {%s %.1f}, want {op 12.5}", tl.Corners[0].ID, tl.Corners[0].DurationSec)
	}
	if tl.Corners[1].ID != "tech" || tl.Corners[1].DurationSec != 30.0 {
		t.Errorf("Corners[1]: got {%s %.1f}, want {tech 30.0}", tl.Corners[1].ID, tl.Corners[1].DurationSec)
	}
}

func TestRunner_Run_EpisodeMetaPassedToAssembler(t *testing.T) {
	outDir := t.TempDir()
	s := defaultStubs()
	s.scr.lines = model.ScriptLines{Corners: make([]model.CornerLines, 0)}
	s.sum = &stubProgramSummarizer{
		summary: "要約",
		notes:   make([]model.ConversationNote, 0),
	}

	if err := newRunner(s).Run(context.Background(), pipeline.Options{
		OutDir:        outDir,
		EpisodeNumber: 3,
	}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if s.asm.capturedMeta.Number != 3 {
		t.Errorf("EpisodeMeta.Number = %d, want 3", s.asm.capturedMeta.Number)
	}
}
