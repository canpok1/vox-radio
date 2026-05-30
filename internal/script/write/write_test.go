package write_test

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/canpok1/vox-radio/internal/model"
	"github.com/canpok1/vox-radio/internal/script/llm"
	"github.com/canpok1/vox-radio/internal/script/write"
)

type mockClient struct {
	response  json.RawMessage
	err       error
	callCount int
	responses []json.RawMessage
	captured  []llm.CompletionRequest
}

func (m *mockClient) Complete(_ context.Context, req llm.CompletionRequest) (json.RawMessage, error) {
	m.captured = append(m.captured, req)
	if len(m.responses) > 0 {
		idx := m.callCount
		m.callCount++
		if idx < len(m.responses) {
			return m.responses[idx], m.err
		}
	}
	return m.response, m.err
}

var linesJSON = json.RawMessage(`{
  "lines": [
    {"speaker_role": "host", "text": "こんにちは"},
    {"speaker_role": "guest", "text": "よろしく"}
  ]
}`)

func TestLLMWriter_Write_Success(t *testing.T) {
	mc := &mockClient{response: linesJSON}
	w := write.NewLLMWriter(mc, "corner={{corner}} summaries={{summaries}} persona={{persona}}")

	corner := model.Corner{Title: "コーナー1", Topic: "AI", Points: []string{"p1"}, TargetChars: 100}
	summaries := []model.Summary{{URL: "https://example.com/1", Summary: "要約", Points: []string{"p1"}}}
	show := model.ShowConfig{Persona: "ホスト設定", TargetChars: 1000}

	got, err := w.Write(context.Background(), corner, summaries, show)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 2 {
		t.Errorf("Lines: got %d, want 2", len(got))
	}
	if got[0].SpeakerRole != "host" {
		t.Errorf("SpeakerRole: got %q, want host", got[0].SpeakerRole)
	}
}

func TestLLMWriter_Write_PromptContainsCornerAndSummaries(t *testing.T) {
	mc := &mockClient{response: linesJSON}
	w := write.NewLLMWriter(mc, "c={{corner}} s={{summaries}} p={{persona}}")

	corner := model.Corner{Title: "AIコーナー", Topic: "AI", Points: []string{"p1"}, TargetChars: 100}
	summaries := []model.Summary{{URL: "https://example.com/1", Summary: "AI要約", Points: []string{"p1"}}}
	show := model.ShowConfig{Persona: "キャラ設定テキスト"}

	_, _ = w.Write(context.Background(), corner, summaries, show)

	if len(mc.captured) == 0 {
		t.Fatal("LLM was not called")
	}
	prompt := mc.captured[0].Messages[0].Content
	if !strings.Contains(prompt, "AIコーナー") {
		t.Errorf("prompt should contain corner title, got: %s", prompt)
	}
	if !strings.Contains(prompt, "AI要約") {
		t.Errorf("prompt should contain summary, got: %s", prompt)
	}
	if !strings.Contains(prompt, "キャラ設定テキスト") {
		t.Errorf("prompt should contain persona, got: %s", prompt)
	}
}

func TestLLMWriter_Write_LLMError(t *testing.T) {
	mc := &mockClient{err: context.Canceled}
	w := write.NewLLMWriter(mc, "{{corner}}")

	_, err := w.Write(context.Background(), model.Corner{}, nil, model.ShowConfig{})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}
