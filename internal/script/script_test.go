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
	lines            []model.Line
	err              error
	callCount        int
	responses        [][]model.Line
	receivedPrevious [][]model.CornerLines
}

func (m *mockWriter) Write(_ context.Context, _ config.ProgramConfig, _ config.CornerConfig, _ []config.CornerConfig, previousCorners []model.CornerLines, _ []model.RundownArticle, _ string, _ map[string]config.CharacterConfig) ([]model.Line, error) {
	m.receivedPrevious = append(m.receivedPrevious, previousCorners)
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
	{Title: "AIコーナー", Content: "AI紹介", Cast: map[string]string{"zundamon": "司会"}, LengthSec: 15},
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
		model.AssetCatalog{SE: []model.AssetCatalogEntry{{Name: "chime"}}},
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
		model.AssetCatalog{SE: make([]model.AssetCatalogEntry, 0)},
		"",
	)

	_, err := gen.Generate(context.Background(), config.ProgramConfig{}, corneredRundown("AIコーナー", model.RundownArticle{URL: "u"}), testCorners, testChars)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestLLMScriptGenerator_Generate_CharCountRegen(t *testing.T) {
	// LengthSec=15 → 105 chars, but writer first returns 1 char total (huge deficit)
	// should trigger regen of the worst corner
	corners := []config.CornerConfig{
		{Title: "C", Content: "内容", Cast: map[string]string{"zundamon": "司会"}, LengthSec: 15},
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
		model.AssetCatalog{SE: make([]model.AssetCatalogEntry, 0)},
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
		{Title: "C", Content: "内容", Cast: map[string]string{"zundamon": "司会"}, LengthSec: 15},
	}
	rundown := corneredRundown("C", model.RundownArticle{URL: "https://example.com/1"})
	// 95 chars → ~9.5% deviation (target=105 = 15sec*7), within 20% threshold
	lines := []model.Line{{SpeakerRole: "zundamon", Text: "あいうえおかきくけこさしすせそたちつてとなにぬねのはひふへほまみむめもやゆよらりるれろわをんあいうえおかきくけこさしすせそたちつてとなにぬねのはひふへほまみむめもやゆよらりるれろわをんあいう"}}

	mw := &mockWriter{lines: lines}
	gen := script.NewLLMScriptGenerator(
		mw,
		&mockDirector{},
		model.AssetCatalog{SE: make([]model.AssetCatalogEntry, 0)},
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
		model.AssetCatalog{SE: make([]model.AssetCatalogEntry, 0)},
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
		model.AssetCatalog{SE: make([]model.AssetCatalogEntry, 0)},
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
		{Title: "C", Content: "内容", Cast: map[string]string{"zundamon": "司会"}, LengthSec: 0},
	}
	rundown := corneredRundown("C", model.RundownArticle{URL: "https://example.com/1"})
	lines := []model.Line{{SpeakerRole: "zundamon", Text: "A"}}

	mw := &mockWriter{lines: lines}
	gen := script.NewLLMScriptGenerator(
		mw,
		&mockDirector{},
		model.AssetCatalog{SE: make([]model.AssetCatalogEntry, 0)},
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

	gen := script.NewLLMScriptGenerator(mw, md, model.AssetCatalog{SE: make([]model.AssetCatalogEntry, 0)}, "", script.WithLogger(logger))

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
		model.AssetCatalog{SE: make([]model.AssetCatalogEntry, 0)},
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

func TestBuildScriptLines(t *testing.T) {
	corners := []config.CornerConfig{
		{Title: "C1", Direction: "演出1", Content: "内容1"},
		{Title: "C2", Direction: "", Content: "内容2"},
	}
	lines1 := []model.Line{{SpeakerRole: "zundamon", Text: "line1"}}
	lines2 := []model.Line{{SpeakerRole: "metan", Text: "line2"}, {SpeakerRole: "metan", Text: "line3"}}
	cornerLines := [][]model.Line{lines1, lines2}

	got := script.BuildScriptLines(corners, cornerLines)

	if len(got) != 2 {
		t.Fatalf("len: got %d, want 2", len(got))
	}
	if got[0].Title != "C1" {
		t.Errorf("got[0].Title: got %q, want C1", got[0].Title)
	}
	if got[0].Direction != "演出1" {
		t.Errorf("got[0].Direction: got %q, want 演出1", got[0].Direction)
	}
	if len(got[0].Lines) != 1 || got[0].Lines[0].Text != "line1" {
		t.Errorf("got[0].Lines: unexpected %+v", got[0].Lines)
	}
	if got[1].Title != "C2" {
		t.Errorf("got[1].Title: got %q, want C2", got[1].Title)
	}
	if got[1].Direction != "" {
		t.Errorf("got[1].Direction: got %q, want empty", got[1].Direction)
	}
	if len(got[1].Lines) != 2 {
		t.Errorf("got[1].Lines: got %d lines, want 2", len(got[1].Lines))
	}
}

func TestLLMScriptGenerator_Generate_LinesFileUsesCornerStructure(t *testing.T) {
	workDir := t.TempDir()
	rundown := corneredRundown("AIコーナー",
		model.RundownArticle{URL: "https://example.com/1", Title: "AI", Summary: "要約", Points: []string{"p1"}},
	)
	lines := []model.Line{{SpeakerRole: "zundamon", Text: "テスト"}}

	corners := []config.CornerConfig{
		{Title: "AIコーナー", Content: "AI紹介", Direction: "冒頭でSEを流す。", Cast: map[string]string{"zundamon": "司会"}, LengthSec: 15},
	}
	gen := script.NewLLMScriptGenerator(
		&mockWriter{lines: lines},
		&mockDirector{},
		model.AssetCatalog{SE: make([]model.AssetCatalogEntry, 0)},
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

func TestBuildScriptLines_TransfersCornerAssets(t *testing.T) {
	corners := []config.CornerConfig{
		{Title: "OP", Direction: "dir", StartJingle: "opening", BGM: "bgm1"},
		{Title: "ED", EndJingle: "ending"},
	}
	lines := [][]model.Line{
		{{SpeakerRole: "host", Text: "A"}},
		{{SpeakerRole: "host", Text: "B"}},
	}
	got := script.BuildScriptLines(corners, lines)
	if got[0].StartJingle != "opening" {
		t.Errorf("Corners[0].StartJingle: got %q, want opening", got[0].StartJingle)
	}
	if got[0].BGM != "bgm1" {
		t.Errorf("Corners[0].BGM: got %q, want bgm1", got[0].BGM)
	}
	if got[0].EndJingle != "" {
		t.Errorf("Corners[0].EndJingle: got %q, want empty", got[0].EndJingle)
	}
	if got[1].EndJingle != "ending" {
		t.Errorf("Corners[1].EndJingle: got %q, want ending", got[1].EndJingle)
	}
	if got[1].StartJingle != "" {
		t.Errorf("Corners[1].StartJingle: got %q, want empty", got[1].StartJingle)
	}
}

func TestLLMScriptGenerator_Generate_PassesPreviousCornersAccumulated(t *testing.T) {
	corners := []config.CornerConfig{
		{Title: "C1", Content: "内容1", Cast: map[string]string{"zundamon": "司会"}, LengthSec: 15},
		{Title: "C2", Content: "内容2", Cast: map[string]string{"zundamon": "司会"}, LengthSec: 15},
		{Title: "C3", Content: "内容3", Cast: map[string]string{"zundamon": "司会"}, LengthSec: 15},
	}
	rundown := model.Rundown{
		Corners: []model.RundownCorner{
			{Title: "C1", Flow: "フロー1"},
			{Title: "C2", Flow: "フロー2"},
			{Title: "C3", Flow: "フロー3"},
		},
	}
	c1Lines := []model.Line{{SpeakerRole: "zundamon", Text: "C1のセリフ"}}
	c2Lines := []model.Line{{SpeakerRole: "metan", Text: "C2のセリフ"}}
	c3Lines := []model.Line{{SpeakerRole: "zundamon", Text: "C3のセリフ"}}

	mw := &mockWriter{responses: [][]model.Line{c1Lines, c2Lines, c3Lines}}
	gen := script.NewLLMScriptGenerator(
		mw,
		&mockDirector{},
		model.AssetCatalog{SE: make([]model.AssetCatalogEntry, 0)},
		"",
	)

	_, err := gen.Generate(context.Background(), config.ProgramConfig{}, rundown, corners, testChars)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(mw.receivedPrevious) < 3 {
		t.Fatalf("expected 3 Write calls, got %d", len(mw.receivedPrevious))
	}
	// C1: no previous corners
	if len(mw.receivedPrevious[0]) != 0 {
		t.Errorf("C1 should receive empty previousCorners, got %d", len(mw.receivedPrevious[0]))
	}
	// C2: receives C1's lines
	if len(mw.receivedPrevious[1]) != 1 {
		t.Fatalf("C2 should receive 1 previousCorner, got %d", len(mw.receivedPrevious[1]))
	}
	if mw.receivedPrevious[1][0].Title != "C1" {
		t.Errorf("C2's previousCorners[0].Title: got %q, want C1", mw.receivedPrevious[1][0].Title)
	}
	if len(mw.receivedPrevious[1][0].Lines) != 1 || mw.receivedPrevious[1][0].Lines[0].Text != "C1のセリフ" {
		t.Errorf("C2's previousCorners[0].Lines unexpected: %+v", mw.receivedPrevious[1][0].Lines)
	}
	// C3: receives C1's and C2's lines
	if len(mw.receivedPrevious[2]) != 2 {
		t.Fatalf("C3 should receive 2 previousCorners, got %d", len(mw.receivedPrevious[2]))
	}
	if mw.receivedPrevious[2][0].Title != "C1" {
		t.Errorf("C3's previousCorners[0].Title: got %q, want C1", mw.receivedPrevious[2][0].Title)
	}
	if mw.receivedPrevious[2][1].Title != "C2" {
		t.Errorf("C3's previousCorners[1].Title: got %q, want C2", mw.receivedPrevious[2][1].Title)
	}
}
