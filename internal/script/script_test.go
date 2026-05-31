package script_test

import (
	"context"
	"encoding/json"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/canpok1/vox-radio/internal/config"
	"github.com/canpok1/vox-radio/internal/fileio"
	"github.com/canpok1/vox-radio/internal/model"
	"github.com/canpok1/vox-radio/internal/script"
)

// mock implementations

type mockWriter struct {
	lines     []model.Line
	err       error
	callCount int
	responses [][]model.Line
}

func (m *mockWriter) Write(_ context.Context, _ config.ProgramConfig, _ config.CornerConfig, _ []config.CornerConfig, _ []model.RundownArticle, _ string, _ map[string]config.CharacterConfig) ([]model.Line, error) {
	if m.err != nil {
		return nil, m.err
	}
	if len(m.responses) > 0 && m.callCount < len(m.responses) {
		resp := m.responses[m.callCount]
		m.callCount++
		return resp, nil
	}
	m.callCount++
	return m.lines, nil
}

type mockDirector struct {
	script model.Script
	err    error
}

func (m *mockDirector) Direct(_ context.Context, corners []model.CornerLines, _ model.AssetCatalog) (model.Script, error) {
	if m.err != nil {
		return model.Script{}, m.err
	}
	if m.script.Segments != nil {
		return m.script, nil
	}
	segs := make([]model.ScriptSegment, 0)
	for _, corner := range corners {
		for _, l := range corner.Lines {
			segs = append(segs, model.ScriptSegment{Type: model.SegmentTypeSpeech, SpeakerRole: l.SpeakerRole, Text: l.Text})
		}
	}
	return model.Script{Segments: segs}, nil
}

var testChars = map[string]config.CharacterConfig{
	"zundamon": {Name: "ずんだもん", DefaultStyle: "ノーマル", Styles: map[string]int{"ノーマル": 3}},
}

var testCorners = []config.CornerConfig{
	{Title: "AIコーナー", Content: "AI紹介", Cast: map[string]string{"zundamon": "司会"}, TargetDurationSec: 15},
}

// corneredRundown wraps rundown articles into a model.Rundown attributed to the given corner title.
func corneredRundown(cornerTitle string, arts ...model.RundownArticle) model.Rundown {
	return model.Rundown{
		Corners: []model.RundownCorner{
			{Title: cornerTitle, Flow: "テスト用フロー", Articles: arts},
		},
	}
}

func TestLLMScriptGenerator_Generate_HappyPath(t *testing.T) {
	rundown := corneredRundown("AIコーナー",
		model.RundownArticle{URL: "https://example.com/1", Title: "AI", Summary: "要約", Points: []string{"p1"}},
	)
	lines := []model.Line{
		{SpeakerRole: "zundamon", Text: "テスト"},
	}

	gen := script.NewLLMScriptGenerator(
		&mockWriter{lines: lines},
		&mockDirector{},
		model.AssetCatalog{SE: []model.AssetCatalogEntry{{Name: "chime"}}, BGM: make([]model.AssetCatalogEntry, 0), Jingle: make([]model.AssetCatalogEntry, 0)},
		"",
	)

	got, err := gen.Generate(context.Background(), config.ProgramConfig{}, rundown, testCorners, testChars)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got.Segments) == 0 {
		t.Error("expected non-empty segments")
	}
}

func TestLLMScriptGenerator_Generate_WriteError(t *testing.T) {
	gen := script.NewLLMScriptGenerator(
		&mockWriter{err: context.Canceled},
		&mockDirector{},
		model.AssetCatalog{SE: make([]model.AssetCatalogEntry, 0), BGM: make([]model.AssetCatalogEntry, 0), Jingle: make([]model.AssetCatalogEntry, 0)},
		"",
	)

	_, err := gen.Generate(context.Background(), config.ProgramConfig{}, corneredRundown("AIコーナー", model.RundownArticle{URL: "u"}), testCorners, testChars)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestLLMScriptGenerator_Generate_CharCountRegen(t *testing.T) {
	// TargetDurationSec=15 → 105 chars, but writer first returns 1 char total (huge deficit)
	// should trigger regen of the worst corner
	corners := []config.CornerConfig{
		{Title: "C", Content: "内容", Cast: map[string]string{"zundamon": "司会"}, TargetDurationSec: 15},
	}
	rundown := corneredRundown("C", model.RundownArticle{URL: "https://example.com/1"})
	shortLines := []model.Line{{SpeakerRole: "zundamon", Text: "A"}}                                                  // 1 char
	longLines := []model.Line{{SpeakerRole: "zundamon", Text: "あいうえおかきくけこさしすせそたちつてとなにぬねのはひふへほまみむめもやゆよらりるれろわをんあいうえお"}} // ~45 chars

	mw := &mockWriter{
		responses: [][]model.Line{shortLines, longLines},
	}

	gen := script.NewLLMScriptGenerator(
		mw,
		&mockDirector{},
		model.AssetCatalog{SE: make([]model.AssetCatalogEntry, 0), BGM: make([]model.AssetCatalogEntry, 0), Jingle: make([]model.AssetCatalogEntry, 0)},
		"",
	)

	_, err := gen.Generate(context.Background(), config.ProgramConfig{}, rundown, corners, testChars)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if mw.callCount < 2 {
		t.Errorf("expected writer called at least 2 times, got %d", mw.callCount)
	}
}

func TestLLMScriptGenerator_Generate_NoRegenWhenWithinThreshold(t *testing.T) {
	corners := []config.CornerConfig{
		{Title: "C", Content: "内容", Cast: map[string]string{"zundamon": "司会"}, TargetDurationSec: 15},
	}
	rundown := corneredRundown("C", model.RundownArticle{URL: "https://example.com/1"})
	// 95 chars → ~9.5% deviation (target=105 = 15sec*7), within 20% threshold
	lines := []model.Line{{SpeakerRole: "zundamon", Text: "あいうえおかきくけこさしすせそたちつてとなにぬねのはひふへほまみむめもやゆよらりるれろわをんあいうえおかきくけこさしすせそたちつてとなにぬねのはひふへほまみむめもやゆよらりるれろわをんあいう"}}

	mw := &mockWriter{lines: lines}
	gen := script.NewLLMScriptGenerator(
		mw,
		&mockDirector{},
		model.AssetCatalog{SE: make([]model.AssetCatalogEntry, 0), BGM: make([]model.AssetCatalogEntry, 0), Jingle: make([]model.AssetCatalogEntry, 0)},
		"",
	)

	_, err := gen.Generate(context.Background(), config.ProgramConfig{}, rundown, corners, testChars)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if mw.callCount != 1 {
		t.Errorf("expected writer called 1 time, got %d", mw.callCount)
	}
}

func TestLLMScriptGenerator_Generate_EmptyRundown(t *testing.T) {
	gen := script.NewLLMScriptGenerator(
		&mockWriter{lines: []model.Line{{SpeakerRole: "zundamon", Text: "テスト"}}},
		&mockDirector{},
		model.AssetCatalog{SE: make([]model.AssetCatalogEntry, 0), BGM: make([]model.AssetCatalogEntry, 0), Jingle: make([]model.AssetCatalogEntry, 0)},
		"",
	)

	got, err := gen.Generate(context.Background(), config.ProgramConfig{}, model.Rundown{}, testCorners, testChars)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got.Segments) == 0 {
		t.Error("expected non-empty segments from writer+director")
	}
}

func TestLLMScriptGenerator_Generate_EmptyCorners(t *testing.T) {
	gen := script.NewLLMScriptGenerator(
		&mockWriter{},
		&mockDirector{},
		model.AssetCatalog{SE: make([]model.AssetCatalogEntry, 0), BGM: make([]model.AssetCatalogEntry, 0), Jingle: make([]model.AssetCatalogEntry, 0)},
		"",
	)

	got, err := gen.Generate(context.Background(), config.ProgramConfig{}, corneredRundown("AIコーナー", model.RundownArticle{URL: "u"}), []config.CornerConfig{}, testChars)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Segments == nil {
		t.Error("segments should be non-nil (empty slice expected)")
	}
}

func TestLLMScriptGenerator_Generate_NoRegenWhenAllCornersHaveZeroTarget(t *testing.T) {
	corners := []config.CornerConfig{
		{Title: "C", Content: "内容", Cast: map[string]string{"zundamon": "司会"}, TargetDurationSec: 0},
	}
	rundown := corneredRundown("C", model.RundownArticle{URL: "https://example.com/1"})
	lines := []model.Line{{SpeakerRole: "zundamon", Text: "A"}}

	mw := &mockWriter{lines: lines}
	gen := script.NewLLMScriptGenerator(
		mw,
		&mockDirector{},
		model.AssetCatalog{SE: make([]model.AssetCatalogEntry, 0), BGM: make([]model.AssetCatalogEntry, 0), Jingle: make([]model.AssetCatalogEntry, 0)},
		"",
	)

	_, err := gen.Generate(context.Background(), config.ProgramConfig{}, rundown, corners, testChars)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if mw.callCount != 1 {
		t.Errorf("expected writer called 1 time (no regen), got %d", mw.callCount)
	}
}

func TestLLMScriptGenerator_Generate_LogsProgress(t *testing.T) {
	mw := &mockWriter{lines: []model.Line{{SpeakerRole: "zundamon", Text: "テスト"}}}
	md := &mockDirector{script: model.Script{Segments: []model.ScriptSegment{{Type: model.SegmentTypeSpeech, Text: "テスト"}}}}

	rundown := model.Rundown{
		Corners: []model.RundownCorner{
			{
				Title: "ニュース",
				Flow:  "記事を紹介",
				Articles: []model.RundownArticle{
					{URL: "https://example.com/1", Title: "記事1", Summary: "要約1", Points: []string{"p1"}},
				},
			},
		},
	}
	corners := []config.CornerConfig{{Title: "ニュース"}}

	var buf strings.Builder
	logger := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelInfo}))

	gen := script.NewLLMScriptGenerator(mw, md, model.AssetCatalog{SE: make([]model.AssetCatalogEntry, 0), BGM: make([]model.AssetCatalogEntry, 0), Jingle: make([]model.AssetCatalogEntry, 0)}, "", script.WithLogger(logger))

	_, err := gen.Generate(context.Background(), config.ProgramConfig{}, rundown, corners, testChars)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	logs := buf.String()
	if !strings.Contains(logs, "完了") {
		t.Errorf("should log complete: %q", logs)
	}
}

func TestLLMScriptGenerator_Generate_SavesLinesIntermediateFile(t *testing.T) {
	workDir := t.TempDir()
	rundown := corneredRundown("AIコーナー",
		model.RundownArticle{URL: "https://example.com/1", Title: "AI", Summary: "要約", Points: []string{"p1"}},
	)
	lines := []model.Line{{SpeakerRole: "zundamon", Text: "テスト"}}

	gen := script.NewLLMScriptGenerator(
		&mockWriter{lines: lines},
		&mockDirector{},
		model.AssetCatalog{SE: make([]model.AssetCatalogEntry, 0), BGM: make([]model.AssetCatalogEntry, 0), Jingle: make([]model.AssetCatalogEntry, 0)},
		workDir,
	)

	if _, err := gen.Generate(context.Background(), config.ProgramConfig{}, rundown, testCorners, testChars); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	path := filepath.Join(workDir, fileio.FileLines)
	if _, err := os.Stat(path); err != nil {
		t.Errorf("expected intermediate file %q to exist: %v", fileio.FileLines, err)
	}
}

func TestLLMScriptGenerator_Generate_LinesFileUsesCornerStructure(t *testing.T) {
	workDir := t.TempDir()
	rundown := corneredRundown("AIコーナー",
		model.RundownArticle{URL: "https://example.com/1", Title: "AI", Summary: "要約", Points: []string{"p1"}},
	)
	lines := []model.Line{{SpeakerRole: "zundamon", Text: "テスト"}}

	corners := []config.CornerConfig{
		{Title: "AIコーナー", Content: "AI紹介", Direction: "冒頭でSEを流す。", Cast: map[string]string{"zundamon": "司会"}, TargetDurationSec: 15},
	}
	gen := script.NewLLMScriptGenerator(
		&mockWriter{lines: lines},
		&mockDirector{},
		model.AssetCatalog{SE: make([]model.AssetCatalogEntry, 0), BGM: make([]model.AssetCatalogEntry, 0), Jingle: make([]model.AssetCatalogEntry, 0)},
		workDir,
	)

	if _, err := gen.Generate(context.Background(), config.ProgramConfig{}, rundown, corners, testChars); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	path := filepath.Join(workDir, fileio.FileLines)
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("expected intermediate file to exist: %v", err)
	}

	var sl model.ScriptLines
	if err := json.Unmarshal(data, &sl); err != nil {
		t.Fatalf("03_lines.json must be ScriptLines structure: %v\nContent: %s", err, data)
	}
	if len(sl.Corners) != 1 {
		t.Fatalf("ScriptLines.Corners: got %d, want 1", len(sl.Corners))
	}
	if sl.Corners[0].Title != "AIコーナー" {
		t.Errorf("Corners[0].Title: got %q, want AIコーナー", sl.Corners[0].Title)
	}
	if sl.Corners[0].Direction != "冒頭でSEを流す。" {
		t.Errorf("Corners[0].Direction: got %q, want 冒頭でSEを流す。", sl.Corners[0].Direction)
	}
}

// newJingleTestGen creates a generator with a fixed-script director for jingle injection tests.
// The mockDirector returns its fixed script, so mockWriter content is irrelevant.
func newJingleTestGen() *script.LLMScriptGenerator {
	speechSeg := model.ScriptSegment{Type: model.SegmentTypeSpeech, SpeakerRole: "zundamon", Text: "テスト"}
	md := &mockDirector{script: model.Script{Segments: []model.ScriptSegment{speechSeg}}}
	return script.NewLLMScriptGenerator(
		&mockWriter{},
		md,
		model.AssetCatalog{SE: make([]model.AssetCatalogEntry, 0), BGM: make([]model.AssetCatalogEntry, 0), Jingle: make([]model.AssetCatalogEntry, 0)},
		"",
	)
}

func TestLLMScriptGenerator_Generate_InjectsOpeningAndEndingJingles(t *testing.T) {
	gen := newJingleTestGen()
	program := config.ProgramConfig{OpeningJingle: "opening", EndingJingle: "ending"}
	got, err := gen.Generate(context.Background(), program, corneredRundown("AIコーナー", model.RundownArticle{URL: "u"}), testCorners, testChars)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(got.Segments) != 3 {
		t.Fatalf("expected 3 segments (opening jingle + speech + ending jingle), got %d", len(got.Segments))
	}
	if got.Segments[0].Type != model.SegmentTypeJingle || got.Segments[0].AssetName != "opening" {
		t.Errorf("first segment should be opening jingle, got %+v", got.Segments[0])
	}
	if got.Segments[1].Type != model.SegmentTypeSpeech {
		t.Errorf("second segment should be speech, got %+v", got.Segments[1])
	}
	if got.Segments[2].Type != model.SegmentTypeJingle || got.Segments[2].AssetName != "ending" {
		t.Errorf("third segment should be ending jingle, got %+v", got.Segments[2])
	}
}

func TestLLMScriptGenerator_Generate_NoJinglesWhenConfigEmpty(t *testing.T) {
	gen := newJingleTestGen()
	got, err := gen.Generate(context.Background(), config.ProgramConfig{}, corneredRundown("AIコーナー", model.RundownArticle{URL: "u"}), testCorners, testChars)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(got.Segments) != 1 {
		t.Fatalf("expected 1 segment (no jingles), got %d", len(got.Segments))
	}
	if got.Segments[0].Type != model.SegmentTypeSpeech {
		t.Errorf("expected speech segment, got %+v", got.Segments[0])
	}
}

func TestLLMScriptGenerator_Generate_InjectsOnlyOpeningJingle(t *testing.T) {
	gen := newJingleTestGen()
	program := config.ProgramConfig{OpeningJingle: "opening"}
	got, err := gen.Generate(context.Background(), program, corneredRundown("AIコーナー", model.RundownArticle{URL: "u"}), testCorners, testChars)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(got.Segments) != 2 {
		t.Fatalf("expected 2 segments (opening jingle + speech), got %d", len(got.Segments))
	}
	if got.Segments[0].Type != model.SegmentTypeJingle || got.Segments[0].AssetName != "opening" {
		t.Errorf("first segment should be opening jingle, got %+v", got.Segments[0])
	}
}

func TestLLMScriptGenerator_Generate_InjectsOnlyEndingJingle(t *testing.T) {
	gen := newJingleTestGen()
	program := config.ProgramConfig{EndingJingle: "ending"}
	got, err := gen.Generate(context.Background(), program, corneredRundown("AIコーナー", model.RundownArticle{URL: "u"}), testCorners, testChars)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(got.Segments) != 2 {
		t.Fatalf("expected 2 segments (speech + ending jingle), got %d", len(got.Segments))
	}
	if got.Segments[1].Type != model.SegmentTypeJingle || got.Segments[1].AssetName != "ending" {
		t.Errorf("last segment should be ending jingle, got %+v", got.Segments[1])
	}
}

func TestInjectProgramJingles_BothEmpty_ReturnsOriginal(t *testing.T) {
	seg := model.ScriptSegment{Type: model.SegmentTypeSpeech, Text: "x"}
	scr := model.Script{Segments: []model.ScriptSegment{seg}}
	got := script.InjectProgramJingles(scr, config.ProgramConfig{})
	if len(got.Segments) != 1 || got.Segments[0].Text != "x" {
		t.Errorf("expected unchanged script, got %+v", got.Segments)
	}
}

func TestInjectProgramJingles_BothSet_PrependAndAppend(t *testing.T) {
	seg := model.ScriptSegment{Type: model.SegmentTypeSpeech, Text: "mid"}
	scr := model.Script{Segments: []model.ScriptSegment{seg}}
	got := script.InjectProgramJingles(scr, config.ProgramConfig{OpeningJingle: "op", EndingJingle: "ed"})
	if len(got.Segments) != 3 {
		t.Fatalf("expected 3 segments, got %d", len(got.Segments))
	}
	if got.Segments[0].AssetName != "op" || got.Segments[2].AssetName != "ed" {
		t.Errorf("unexpected jingle placement: %+v", got.Segments)
	}
}
