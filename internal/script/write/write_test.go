package write_test

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/canpok1/vox-radio/internal/config"
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
    {"speaker_role": "zundamon", "text": "こんにちは"},
    {"speaker_role": "metan", "text": "よろしく"}
  ]
}`)

func TestLLMWriter_Write_Success(t *testing.T) {
	mc := &mockClient{response: linesJSON}
	w := write.NewLLMWriter(mc, "corner={{corner}} summaries={{summary}} cast={{cast_info}}", 0)

	corner := config.CornerConfig{Title: "コーナー1", Content: "内容", Cast: map[string]string{"zundamon": "司会"}, TargetDurationSec: 14}
	summaries := []model.Summary{{URL: "https://example.com/1", Summary: "要約", Points: []string{"p1"}}}
	chars := map[string]config.CharacterConfig{
		"zundamon": {Name: "ずんだもん", Pronoun: "ボク", SpeechSuffix: []string{"〜のだ"}, Personality: []string{"元気"}},
	}

	got, err := w.Write(context.Background(), corner, summaries, chars)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 2 {
		t.Errorf("Lines: got %d, want 2", len(got))
	}
	if got[0].SpeakerRole != "zundamon" {
		t.Errorf("SpeakerRole: got %q, want zundamon", got[0].SpeakerRole)
	}
}

func TestLLMWriter_Write_PromptContainsCornerAndCastInfo(t *testing.T) {
	mc := &mockClient{response: linesJSON}
	w := write.NewLLMWriter(mc, "c={{corner}} s={{summary}} cast={{cast_info}}", 0)

	corner := config.CornerConfig{Title: "AIコーナー", Content: "AI紹介", Cast: map[string]string{"zundamon": "司会"}, TargetDurationSec: 14}
	summaries := []model.Summary{{URL: "https://example.com/1", Summary: "AI要約", Points: []string{"p1"}}}
	chars := map[string]config.CharacterConfig{
		"zundamon": {Name: "ずんだもん", Pronoun: "ボク", SpeechSuffix: []string{"〜のだ"}, Personality: []string{"元気"}},
	}

	_, _ = w.Write(context.Background(), corner, summaries, chars)

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
	if !strings.Contains(prompt, "ずんだもん") {
		t.Errorf("prompt should contain character name, got: %s", prompt)
	}
	if !strings.Contains(prompt, "司会") {
		t.Errorf("prompt should contain role, got: %s", prompt)
	}
}

func TestLLMWriter_Write_LLMError(t *testing.T) {
	mc := &mockClient{err: context.Canceled}
	w := write.NewLLMWriter(mc, "{{corner}}", 0)

	_, err := w.Write(context.Background(), config.CornerConfig{}, nil, nil)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestLLMWriter_Write_PromptContainsConvertedTargetChars(t *testing.T) {
	mc := &mockClient{response: linesJSON}
	w := write.NewLLMWriter(mc, "c={{corner}}", 0)

	// 14sec * 7chars/sec = 98 chars
	corner := config.CornerConfig{Title: "Test", Content: "内容", Cast: map[string]string{"zundamon": "司会"}, TargetDurationSec: 14}
	_, _ = w.Write(context.Background(), corner, nil, nil)

	if len(mc.captured) == 0 {
		t.Fatal("LLM was not called")
	}
	prompt := mc.captured[0].Messages[0].Content
	if !strings.Contains(prompt, `"target_chars":98`) {
		t.Errorf("prompt should contain target_chars:98 (14sec*7), got: %s", prompt)
	}
	if strings.Contains(prompt, "target_duration_sec") {
		t.Errorf("prompt should not expose target_duration_sec to LLM, got: %s", prompt)
	}
}
