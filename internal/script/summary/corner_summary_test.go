package summary_test

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"strings"
	"testing"

	"github.com/canpok1/vox-radio/internal/model"
	"github.com/canpok1/vox-radio/internal/script/summary"
)

func TestLLMCornerSummarizer_SummarizeCorner_ReturnsSummaryAndPoints(t *testing.T) {
	mc := &mockClient{
		response: json.RawMessage(`{"summary":"AIチップについて議論しました。","points":["要点1","要点2"]}`),
	}
	s := summary.NewLLMCornerSummarizer(mc, "コーナー: {{corner_title}}\nセリフ: {{script_lines}}", 0)

	corner := model.CornerLines{
		Title: "今日のテックニュース",
		Lines: []model.Line{
			{SpeakerRole: "zundamon", Text: "AIチップについて話すのだ"},
			{SpeakerRole: "metan", Text: "そうですね"},
		},
	}

	got, err := s.SummarizeCorner(context.Background(), corner, 100)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Summary != "AIチップについて議論しました。" {
		t.Errorf("Summary = %q, want %q", got.Summary, "AIチップについて議論しました。")
	}
	if len(got.Points) != 2 {
		t.Fatalf("Points len = %d, want 2", len(got.Points))
	}
	if got.Points[0] != "要点1" {
		t.Errorf("Points[0] = %q, want %q", got.Points[0], "要点1")
	}
}

func TestLLMCornerSummarizer_SummarizeCorner_PromptContainsCornerTitleAndLines(t *testing.T) {
	mc := &mockClient{
		response: json.RawMessage(`{"summary":"要約","points":[]}`),
	}
	s := summary.NewLLMCornerSummarizer(mc, "コーナー: {{corner_title}}\nセリフ: {{script_lines}}", 0)

	corner := model.CornerLines{
		Title: "今日のテックニュース",
		Lines: []model.Line{
			{SpeakerRole: "zundamon", Text: "最新のAIニュース"},
		},
	}

	_, _ = s.SummarizeCorner(context.Background(), corner, 100)

	if len(mc.captured) == 0 {
		t.Fatal("LLM was not called")
	}
	prompt := mc.captured[0].Messages[0].Content
	if !strings.Contains(prompt, "今日のテックニュース") {
		t.Errorf("prompt should contain corner title, got: %s", prompt)
	}
	if !strings.Contains(prompt, "最新のAIニュース") {
		t.Errorf("prompt should contain script line text, got: %s", prompt)
	}
}

func TestLLMCornerSummarizer_SummarizeCorner_EmptyLinesReturnsEmptyResult(t *testing.T) {
	mc := &mockClient{
		response: json.RawMessage(`{"summary":"","points":[]}`),
	}
	s := summary.NewLLMCornerSummarizer(mc, "{{corner_title}} {{script_lines}}", 0)

	corner := model.CornerLines{
		Title: "オープニング",
		Lines: []model.Line{},
	}

	got, err := s.SummarizeCorner(context.Background(), corner, 100)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Summary != "" {
		t.Errorf("Summary = %q, want empty string", got.Summary)
	}
	if got.Points == nil {
		t.Error("Points must be [] not nil")
	}
}

func TestLLMCornerSummarizer_SummarizeCorner_PointsNeverNil(t *testing.T) {
	mc := &mockClient{
		response: json.RawMessage(`{"summary":"要約","points":null}`),
	}
	s := summary.NewLLMCornerSummarizer(mc, "{{script_lines}}", 0)

	corner := model.CornerLines{
		Title: "コーナー",
		Lines: []model.Line{{Text: "テスト"}},
	}

	got, err := s.SummarizeCorner(context.Background(), corner, 100)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Points == nil {
		t.Error("Points must be [] not nil even when LLM returns null")
	}
}

func TestLLMCornerSummarizer_SummarizeCorner_LLMError(t *testing.T) {
	mc := &mockClient{err: errors.New("llm error")}
	s := summary.NewLLMCornerSummarizer(mc, "{{script_lines}}", 0)

	corner := model.CornerLines{
		Title: "コーナー",
		Lines: []model.Line{{Text: "テスト"}},
	}

	_, err := s.SummarizeCorner(context.Background(), corner, 100)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestLLMCornerSummarizer_SummarizeCorner_PromptContainsSummaryLength(t *testing.T) {
	mc := &mockClient{
		response: json.RawMessage(`{"summary":"要約","points":[]}`),
	}
	s := summary.NewLLMCornerSummarizer(mc, "{{corner_title}} {{summary_length}}文字程度", 0)

	corner := model.CornerLines{
		Title: "テックニュース",
		Lines: []model.Line{{Text: "テスト"}},
	}

	_, _ = s.SummarizeCorner(context.Background(), corner, 120)

	if len(mc.captured) == 0 {
		t.Fatal("LLM was not called")
	}
	prompt := mc.captured[0].Messages[0].Content
	if !strings.Contains(prompt, "120") {
		t.Errorf("prompt should contain summary_length=120, got: %s", prompt)
	}
	if strings.Contains(prompt, "{{summary_length}}") {
		t.Errorf("prompt should not contain unexpanded placeholder, got: %s", prompt)
	}
}

func TestLLMCornerSummarizer_SummarizeCorner_LogsProgressWithWithLogger(t *testing.T) {
	var buf strings.Builder
	logger := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelInfo}))

	mc := &mockClient{
		response: json.RawMessage(`{"summary":"要約","points":[]}`),
	}
	s := summary.NewLLMCornerSummarizer(mc, "{{corner_title}} {{script_lines}}", 0, summary.WithLogger(logger))

	corner := model.CornerLines{
		Title: "テストコーナー",
		Lines: []model.Line{{Text: "テスト"}},
	}

	_, err := s.SummarizeCorner(context.Background(), corner, 100)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	logs := buf.String()
	if !strings.Contains(logs, "summary/corner") {
		t.Errorf("should log step=summary/corner, got: %q", logs)
	}
}
