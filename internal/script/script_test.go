package script_test

import (
	"context"
	"testing"

	"github.com/canpok1/vox-radio/internal/config"
	"github.com/canpok1/vox-radio/internal/model"
	"github.com/canpok1/vox-radio/internal/script"
)

// mock implementations

type mockSummarizer struct {
	byURL map[string]model.Summary
	err   error
}

func (m *mockSummarizer) Summarize(_ context.Context, a model.Article) (model.Summary, error) {
	if m.err != nil {
		return model.Summary{}, m.err
	}
	if s, ok := m.byURL[a.URL]; ok {
		return s, nil
	}
	return model.Summary{URL: a.URL, Summary: "default", Points: []string{"p1"}}, nil
}

type mockWriter struct {
	lines     []model.Line
	err       error
	callCount int
	responses [][]model.Line
}

func (m *mockWriter) Write(_ context.Context, _ config.CornerConfig, _ []model.Summary, _ map[string]config.CharacterConfig) ([]model.Line, error) {
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

func (m *mockDirector) Direct(_ context.Context, lines []model.Line, _ model.SECatalog) (model.Script, error) {
	if m.err != nil {
		return model.Script{}, m.err
	}
	if m.script.Segments != nil {
		return m.script, nil
	}
	segs := make([]model.ScriptSegment, len(lines))
	for i, l := range lines {
		segs[i] = model.ScriptSegment{Type: model.SegmentTypeSpeech, SpeakerRole: l.SpeakerRole, Text: l.Text}
	}
	return model.Script{Segments: segs}, nil
}

var testChars = map[string]config.CharacterConfig{
	"zundamon": {Name: "ずんだもん", DefaultStyle: "ノーマル", Styles: map[string]int{"ノーマル": 3}},
}

var testCorners = []config.CornerConfig{
	{Title: "AIコーナー", Content: "AI紹介", Cast: map[string]string{"zundamon": "司会"}, TargetChars: 100},
}

// corneredArticles wraps articles into a model.Articles attributed to the given corner title.
func corneredArticles(cornerTitle string, arts ...model.Article) model.Articles {
	return model.Articles{
		Corners: []model.CornerArticles{
			{CornerTitle: cornerTitle, Articles: arts},
		},
	}
}

func TestLLMScriptGenerator_Generate_HappyPath(t *testing.T) {
	articles := corneredArticles("AIコーナー",
		model.Article{URL: "https://example.com/1", Title: "AI", Body: "本文"},
	)
	lines := []model.Line{
		{SpeakerRole: "zundamon", Text: "テスト"},
	}

	gen := script.NewLLMScriptGenerator(
		&mockSummarizer{},
		&mockWriter{lines: lines},
		&mockDirector{},
		model.SECatalog{Names: []string{"chime"}},
		"",
	)

	got, err := gen.Generate(context.Background(), articles, testCorners, testChars)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got.Segments) == 0 {
		t.Error("expected non-empty segments")
	}
}

func TestLLMScriptGenerator_Generate_SummarizeError(t *testing.T) {
	gen := script.NewLLMScriptGenerator(
		&mockSummarizer{err: context.Canceled},
		&mockWriter{},
		&mockDirector{},
		model.SECatalog{},
		"",
	)

	_, err := gen.Generate(context.Background(), corneredArticles("AIコーナー", model.Article{URL: "u"}), testCorners, testChars)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestLLMScriptGenerator_Generate_WriteError(t *testing.T) {
	gen := script.NewLLMScriptGenerator(
		&mockSummarizer{},
		&mockWriter{err: context.Canceled},
		&mockDirector{},
		model.SECatalog{},
		"",
	)

	_, err := gen.Generate(context.Background(), corneredArticles("AIコーナー", model.Article{URL: "u"}), testCorners, testChars)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestLLMScriptGenerator_Generate_CharCountRegen(t *testing.T) {
	// TargetChars=100, but writer first returns 1 char total (huge deficit)
	// should trigger regen of the worst corner
	corners := []config.CornerConfig{
		{Title: "C", Content: "内容", Cast: map[string]string{"zundamon": "司会"}, TargetChars: 100},
	}
	articles := corneredArticles("C", model.Article{URL: "https://example.com/1"})
	shortLines := []model.Line{{SpeakerRole: "zundamon", Text: "A"}}                                                  // 1 char
	longLines := []model.Line{{SpeakerRole: "zundamon", Text: "あいうえおかきくけこさしすせそたちつてとなにぬねのはひふへほまみむめもやゆよらりるれろわをんあいうえお"}} // ~45 chars

	mw := &mockWriter{
		responses: [][]model.Line{shortLines, longLines},
	}

	gen := script.NewLLMScriptGenerator(
		&mockSummarizer{},
		mw,
		&mockDirector{},
		model.SECatalog{},
		"",
	)

	_, err := gen.Generate(context.Background(), articles, corners, testChars)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if mw.callCount < 2 {
		t.Errorf("expected writer called at least 2 times, got %d", mw.callCount)
	}
}

func TestLLMScriptGenerator_Generate_NoRegenWhenWithinThreshold(t *testing.T) {
	corners := []config.CornerConfig{
		{Title: "C", Content: "内容", Cast: map[string]string{"zundamon": "司会"}, TargetChars: 100},
	}
	articles := corneredArticles("C", model.Article{URL: "https://example.com/1"})
	// 95 chars → 5% deviation (target=100), within 20% threshold
	lines := []model.Line{{SpeakerRole: "zundamon", Text: "あいうえおかきくけこさしすせそたちつてとなにぬねのはひふへほまみむめもやゆよらりるれろわをんあいうえおかきくけこさしすせそたちつてとなにぬねのはひふへほまみむめもやゆよらりるれろわをんあいう"}}

	mw := &mockWriter{lines: lines}
	gen := script.NewLLMScriptGenerator(
		&mockSummarizer{},
		mw,
		&mockDirector{},
		model.SECatalog{},
		"",
	)

	_, err := gen.Generate(context.Background(), articles, corners, testChars)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if mw.callCount != 1 {
		t.Errorf("expected writer called 1 time, got %d", mw.callCount)
	}
}

func TestLLMScriptGenerator_Generate_EmptyArticles(t *testing.T) {
	gen := script.NewLLMScriptGenerator(
		&mockSummarizer{},
		&mockWriter{lines: []model.Line{{SpeakerRole: "zundamon", Text: "テスト"}}},
		&mockDirector{},
		model.SECatalog{},
		"",
	)

	got, err := gen.Generate(context.Background(), model.Articles{}, testCorners, testChars)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got.Segments) == 0 {
		t.Error("expected non-empty segments from writer+director")
	}
}

func TestLLMScriptGenerator_Generate_EmptyCorners(t *testing.T) {
	gen := script.NewLLMScriptGenerator(
		&mockSummarizer{},
		&mockWriter{},
		&mockDirector{},
		model.SECatalog{},
		"",
	)

	got, err := gen.Generate(context.Background(), corneredArticles("AIコーナー", model.Article{URL: "u"}), []config.CornerConfig{}, testChars)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Segments == nil {
		t.Error("segments should be non-nil (empty slice expected)")
	}
}

func TestLLMScriptGenerator_Generate_NoRegenWhenAllCornersHaveZeroTarget(t *testing.T) {
	corners := []config.CornerConfig{
		{Title: "C", Content: "内容", Cast: map[string]string{"zundamon": "司会"}, TargetChars: 0},
	}
	articles := corneredArticles("C", model.Article{URL: "https://example.com/1"})
	lines := []model.Line{{SpeakerRole: "zundamon", Text: "A"}}

	mw := &mockWriter{lines: lines}
	gen := script.NewLLMScriptGenerator(
		&mockSummarizer{},
		mw,
		&mockDirector{},
		model.SECatalog{},
		"",
	)

	_, err := gen.Generate(context.Background(), articles, corners, testChars)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if mw.callCount != 1 {
		t.Errorf("expected writer called 1 time (no regen), got %d", mw.callCount)
	}
}
