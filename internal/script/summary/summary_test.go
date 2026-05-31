package summary_test

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/canpok1/vox-radio/internal/model"
	"github.com/canpok1/vox-radio/internal/script/llm"
	"github.com/canpok1/vox-radio/internal/script/summary"
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

func TestLLMProgramSummarizer_Summarize_ReturnsSummaryText(t *testing.T) {
	mc := &mockClient{
		response: json.RawMessage(`{"summary":"技術ニュースとAIの最新動向を紹介しました。"}`),
	}
	s := summary.NewLLMProgramSummarizer(mc, "台本: {{script_lines}}", 0)

	scr := model.Script{
		Segments: []model.ScriptSegment{
			{Type: model.SegmentTypeSpeech, SpeakerRole: "zundamon", Text: "今日はAIについて話すのだ"},
			{Type: model.SegmentTypeSE, SEName: "chime"},
			{Type: model.SegmentTypeSpeech, SpeakerRole: "metan", Text: "そうですね"},
		},
	}

	got, err := s.Summarize(context.Background(), scr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "技術ニュースとAIの最新動向を紹介しました。" {
		t.Errorf("Summary = %q, want %q", got, "技術ニュースとAIの最新動向を紹介しました。")
	}
}

func TestLLMProgramSummarizer_Summarize_PromptContainsSpeechLines(t *testing.T) {
	mc := &mockClient{
		response: json.RawMessage(`{"summary":"要約"}`),
	}
	s := summary.NewLLMProgramSummarizer(mc, "台本: {{script_lines}}", 0)

	scr := model.Script{
		Segments: []model.ScriptSegment{
			{Type: model.SegmentTypeSpeech, SpeakerRole: "zundamon", Text: "AIチップの話"},
			{Type: model.SegmentTypeSE, SEName: "chime"},
			{Type: model.SegmentTypeSpeech, SpeakerRole: "metan", Text: "最新ニュースです"},
		},
	}

	_, _ = s.Summarize(context.Background(), scr)

	if len(mc.captured) == 0 {
		t.Fatal("LLM was not called")
	}
	prompt := mc.captured[0].Messages[0].Content
	if !strings.Contains(prompt, "AIチップの話") {
		t.Errorf("prompt should contain speech text, got: %s", prompt)
	}
	if !strings.Contains(prompt, "最新ニュースです") {
		t.Errorf("prompt should contain speech text, got: %s", prompt)
	}
}

func TestLLMProgramSummarizer_Summarize_ExcludesSESegments(t *testing.T) {
	mc := &mockClient{
		response: json.RawMessage(`{"summary":"要約"}`),
	}
	s := summary.NewLLMProgramSummarizer(mc, "{{script_lines}}", 0)

	scr := model.Script{
		Segments: []model.ScriptSegment{
			{Type: model.SegmentTypeSE, SEName: "opening_jingle"},
			{Type: model.SegmentTypeSpeech, SpeakerRole: "zundamon", Text: "こんにちは"},
		},
	}

	_, _ = s.Summarize(context.Background(), scr)

	if len(mc.captured) == 0 {
		t.Fatal("LLM was not called")
	}
	prompt := mc.captured[0].Messages[0].Content
	if strings.Contains(prompt, "opening_jingle") {
		t.Errorf("prompt should not contain SE names, got: %s", prompt)
	}
}

func TestLLMProgramSummarizer_Summarize_LLMError(t *testing.T) {
	mc := &mockClient{err: context.Canceled}
	s := summary.NewLLMProgramSummarizer(mc, "{{script_lines}}", 0)

	scr := model.Script{
		Segments: []model.ScriptSegment{
			{Type: model.SegmentTypeSpeech, SpeakerRole: "zundamon", Text: "テスト"},
		},
	}

	_, err := s.Summarize(context.Background(), scr)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}
