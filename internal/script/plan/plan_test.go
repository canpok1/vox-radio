package plan_test

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/canpok1/vox-radio/internal/model"
	"github.com/canpok1/vox-radio/internal/script/llm"
	"github.com/canpok1/vox-radio/internal/script/plan"
)

type mockClient struct {
	response json.RawMessage
	err      error
	captured []llm.CompletionRequest
}

func (m *mockClient) Complete(_ context.Context, req llm.CompletionRequest) (json.RawMessage, error) {
	m.captured = append(m.captured, req)
	return m.response, m.err
}

var rundownJSON = json.RawMessage(`{
  "corners": [
    {
      "title": "AIコーナー",
      "topic": "AI動向",
      "points": ["ポイント1"],
      "target_chars": 500,
      "summary_urls": ["https://example.com/1"]
    }
  ]
}`)

func TestLLMPlanner_Plan_Success(t *testing.T) {
	mc := &mockClient{response: rundownJSON}
	p := plan.NewLLMPlanner(mc, "summaries={{summaries}} config={{show_config}}")

	summaries := []model.Summary{
		{URL: "https://example.com/1", Summary: "要約", Points: []string{"p1"}},
	}
	show := model.ShowConfig{TargetChars: 1000, Corners: 1}

	got, err := p.Plan(context.Background(), summaries, show)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got.Corners) != 1 {
		t.Errorf("Corners: got %d, want 1", len(got.Corners))
	}
	if got.Corners[0].Title != "AIコーナー" {
		t.Errorf("Title: got %q, want AIコーナー", got.Corners[0].Title)
	}
}

func TestLLMPlanner_Plan_PromptContainsSummariesAndConfig(t *testing.T) {
	mc := &mockClient{response: rundownJSON}
	p := plan.NewLLMPlanner(mc, "s={{summaries}} c={{show_config}}")

	summaries := []model.Summary{{URL: "https://example.com/1", Summary: "要約", Points: []string{"p1"}}}
	show := model.ShowConfig{TargetChars: 1000, Corners: 1, Persona: "ホスト"}

	_, _ = p.Plan(context.Background(), summaries, show)

	if len(mc.captured) == 0 {
		t.Fatal("LLM was not called")
	}
	prompt := mc.captured[0].Messages[0].Content
	if !strings.Contains(prompt, "要約") {
		t.Errorf("prompt should contain summaries, got: %s", prompt)
	}
	if !strings.Contains(prompt, "ホスト") {
		t.Errorf("prompt should contain show config persona, got: %s", prompt)
	}
}

func TestLLMPlanner_Plan_LLMError(t *testing.T) {
	mc := &mockClient{err: context.Canceled}
	p := plan.NewLLMPlanner(mc, "{{summaries}}")

	_, err := p.Plan(context.Background(), nil, model.ShowConfig{})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}
