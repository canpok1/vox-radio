package script_test

import (
	"context"
	"testing"

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

type mockPlanner struct {
	rundown model.Rundown
	err     error
}

func (m *mockPlanner) Plan(_ context.Context, _ []model.Summary, _ model.ShowConfig) (model.Rundown, error) {
	return m.rundown, m.err
}

type mockWriter struct {
	lines     []model.Line
	err       error
	callCount int
	responses [][]model.Line
}

func (m *mockWriter) Write(_ context.Context, _ model.Corner, _ []model.Summary, _ model.ShowConfig) ([]model.Line, error) {
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

func TestLLMScriptGenerator_Generate_HappyPath(t *testing.T) {
	articles := []model.Article{
		{URL: "https://example.com/1", Title: "AI", Body: "本文"},
	}
	rundown := model.Rundown{
		Corners: []model.Corner{
			{Title: "AIコーナー", Topic: "AI", Points: []string{"p1"}, TargetChars: 100, SummaryURLs: []string{"https://example.com/1"}},
		},
	}
	lines := []model.Line{
		{SpeakerRole: "host", Text: "テスト"},
	}

	gen := script.NewLLMScriptGenerator(
		&mockSummarizer{},
		&mockPlanner{rundown: rundown},
		&mockWriter{lines: lines},
		&mockDirector{},
		model.SECatalog{Names: []string{"chime"}},
		"",
	)

	got, err := gen.Generate(context.Background(), articles, model.ShowConfig{TargetChars: 500, Corners: 1})
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
		&mockPlanner{},
		&mockWriter{},
		&mockDirector{},
		model.SECatalog{},
		"",
	)

	_, err := gen.Generate(context.Background(), []model.Article{{URL: "u"}}, model.ShowConfig{})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestLLMScriptGenerator_Generate_PlanError(t *testing.T) {
	gen := script.NewLLMScriptGenerator(
		&mockSummarizer{},
		&mockPlanner{err: context.Canceled},
		&mockWriter{},
		&mockDirector{},
		model.SECatalog{},
		"",
	)

	_, err := gen.Generate(context.Background(), []model.Article{{URL: "u"}}, model.ShowConfig{})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestLLMScriptGenerator_Generate_WriteError(t *testing.T) {
	rundown := model.Rundown{
		Corners: []model.Corner{{Title: "C", Topic: "T", TargetChars: 100}},
	}
	gen := script.NewLLMScriptGenerator(
		&mockSummarizer{},
		&mockPlanner{rundown: rundown},
		&mockWriter{err: context.Canceled},
		&mockDirector{},
		model.SECatalog{},
		"",
	)

	_, err := gen.Generate(context.Background(), []model.Article{{URL: "u"}}, model.ShowConfig{})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestLLMScriptGenerator_Generate_CharCountRegen(t *testing.T) {
	// TargetChars=100, but writer first returns 1 char total (huge deficit)
	// should trigger regen of the worst corner
	articles := []model.Article{{URL: "https://example.com/1"}}
	rundown := model.Rundown{
		Corners: []model.Corner{
			{Title: "C", Topic: "T", TargetChars: 100, SummaryURLs: []string{"https://example.com/1"}},
		},
	}
	shortLines := []model.Line{{SpeakerRole: "host", Text: "A"}}                                                  // 1 char
	longLines := []model.Line{{SpeakerRole: "host", Text: "あいうえおかきくけこさしすせそたちつてとなにぬねのはひふへほまみむめもやゆよらりるれろわをんあいうえお"}} // ~45 chars, closer to 100

	mw := &mockWriter{
		responses: [][]model.Line{shortLines, longLines},
	}

	gen := script.NewLLMScriptGenerator(
		&mockSummarizer{},
		&mockPlanner{rundown: rundown},
		mw,
		&mockDirector{},
		model.SECatalog{},
		"",
	)

	_, err := gen.Generate(context.Background(), articles, model.ShowConfig{TargetChars: 100, Corners: 1})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Writer should have been called twice (once initial, once regen)
	if mw.callCount < 2 {
		t.Errorf("expected writer called at least 2 times, got %d", mw.callCount)
	}
}

func TestLLMScriptGenerator_Generate_NoRegenWhenWithinThreshold(t *testing.T) {
	articles := []model.Article{{URL: "https://example.com/1"}}
	rundown := model.Rundown{
		Corners: []model.Corner{
			{Title: "C", Topic: "T", TargetChars: 100, SummaryURLs: []string{"https://example.com/1"}},
		},
	}
	// 95 chars → 5% deviation (target=100), within 20% threshold
	lines := []model.Line{{SpeakerRole: "host", Text: "あいうえおかきくけこさしすせそたちつてとなにぬねのはひふへほまみむめもやゆよらりるれろわをんあいうえおかきくけこさしすせそたちつてとなにぬねのはひふへほまみむめもやゆよらりるれろわをんあいう"}}

	mw := &mockWriter{lines: lines}
	gen := script.NewLLMScriptGenerator(
		&mockSummarizer{},
		&mockPlanner{rundown: rundown},
		mw,
		&mockDirector{},
		model.SECatalog{},
		"",
	)

	_, err := gen.Generate(context.Background(), articles, model.ShowConfig{TargetChars: 100, Corners: 1})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if mw.callCount != 1 {
		t.Errorf("expected writer called 1 time, got %d", mw.callCount)
	}
}

func TestLLMScriptGenerator_Generate_EmptyArticles(t *testing.T) {
	rundown := model.Rundown{
		Corners: []model.Corner{
			{Title: "C", Topic: "T", TargetChars: 100},
		},
	}
	gen := script.NewLLMScriptGenerator(
		&mockSummarizer{},
		&mockPlanner{rundown: rundown},
		&mockWriter{lines: []model.Line{{SpeakerRole: "host", Text: "テスト"}}},
		&mockDirector{},
		model.SECatalog{},
		"",
	)

	got, err := gen.Generate(context.Background(), []model.Article{}, model.ShowConfig{TargetChars: 100})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// no articles → no summaries; planner still called with empty summaries
	if len(got.Segments) == 0 {
		t.Error("expected non-empty segments from planner+writer+director")
	}
}

func TestLLMScriptGenerator_Generate_EmptyCorners(t *testing.T) {
	gen := script.NewLLMScriptGenerator(
		&mockSummarizer{},
		&mockPlanner{rundown: model.Rundown{Corners: []model.Corner{}}},
		&mockWriter{},
		&mockDirector{},
		model.SECatalog{},
		"",
	)

	got, err := gen.Generate(context.Background(), []model.Article{{URL: "u"}}, model.ShowConfig{TargetChars: 100})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// no corners → no lines → director gets empty input → empty script
	if got.Segments == nil {
		t.Error("segments should be non-nil (empty slice expected)")
	}
}

func TestLLMScriptGenerator_Generate_NoRegenWhenAllCornersHaveZeroTarget(t *testing.T) {
	articles := []model.Article{{URL: "https://example.com/1"}}
	// corners with TargetChars=0 should skip regen even with bad total char count
	rundown := model.Rundown{
		Corners: []model.Corner{
			{Title: "C", Topic: "T", TargetChars: 0, SummaryURLs: []string{"https://example.com/1"}},
		},
	}
	lines := []model.Line{{SpeakerRole: "host", Text: "A"}} // 1 char, show target=100

	mw := &mockWriter{lines: lines}
	gen := script.NewLLMScriptGenerator(
		&mockSummarizer{},
		&mockPlanner{rundown: rundown},
		mw,
		&mockDirector{},
		model.SECatalog{},
		"",
	)

	_, err := gen.Generate(context.Background(), articles, model.ShowConfig{TargetChars: 100})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// corners with TargetChars=0 → no regen triggered
	if mw.callCount != 1 {
		t.Errorf("expected writer called 1 time (no regen), got %d", mw.callCount)
	}
}
