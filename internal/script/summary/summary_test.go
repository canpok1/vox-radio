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
		response: json.RawMessage(`{"summary":"技術ニュースとAIの最新動向を紹介しました。","conversation_notes":[]}`),
	}
	s := summary.NewLLMProgramSummarizer(mc, "台本: {{script_lines}}", 0)

	scr := model.Script{
		Segments: []model.ScriptSegment{
			{Type: model.SegmentTypeSpeech, SpeakerRole: "zundamon", Text: "今日はAIについて話すのだ"},
			{Type: model.SegmentTypeSE, AssetName: "chime"},
			{Type: model.SegmentTypeSpeech, SpeakerRole: "metan", Text: "そうですね"},
		},
	}

	got, err := s.Summarize(context.Background(), scr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Summary != "技術ニュースとAIの最新動向を紹介しました。" {
		t.Errorf("Summary = %q, want %q", got.Summary, "技術ニュースとAIの最新動向を紹介しました。")
	}
}

func TestLLMProgramSummarizer_Summarize_ReturnsConversationNotes(t *testing.T) {
	mc := &mockClient{
		response: json.RawMessage(`{
			"summary": "要約",
			"conversation_notes": [
				{"category": "近況", "character_ids": ["zundamon"], "note": "カフェにハマっている"}
			]
		}`),
	}
	s := summary.NewLLMProgramSummarizer(mc, "{{script_lines}}", 0)

	scr := model.Script{
		Segments: []model.ScriptSegment{
			{Type: model.SegmentTypeSpeech, SpeakerRole: "zundamon", Text: "カフェの話"},
		},
	}

	got, err := s.Summarize(context.Background(), scr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got.ConversationNotes) != 1 {
		t.Fatalf("ConversationNotes: got %d, want 1", len(got.ConversationNotes))
	}
	n := got.ConversationNotes[0]
	if n.Category != "近況" {
		t.Errorf("Category: got %q, want %q", n.Category, "近況")
	}
	if len(n.CharacterIDs) != 1 || n.CharacterIDs[0] != "zundamon" {
		t.Errorf("CharacterIDs: got %v, want [zundamon]", n.CharacterIDs)
	}
	if n.Note != "カフェにハマっている" {
		t.Errorf("Note: got %q, want %q", n.Note, "カフェにハマっている")
	}
}

func TestLLMProgramSummarizer_Summarize_NilConversationNotesNormalized(t *testing.T) {
	mc := &mockClient{
		response: json.RawMessage(`{"summary":"要約","conversation_notes":null}`),
	}
	s := summary.NewLLMProgramSummarizer(mc, "{{script_lines}}", 0)

	scr := model.Script{
		Segments: []model.ScriptSegment{
			{Type: model.SegmentTypeSpeech, SpeakerRole: "zundamon", Text: "テスト"},
		},
	}

	got, err := s.Summarize(context.Background(), scr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.ConversationNotes == nil {
		t.Error("ConversationNotes must not be nil (should be normalized to empty slice)")
	}
	if len(got.ConversationNotes) != 0 {
		t.Errorf("ConversationNotes: got %d items, want 0", len(got.ConversationNotes))
	}
}

func TestLLMProgramSummarizer_Summarize_NilCharacterIDsNormalized(t *testing.T) {
	mc := &mockClient{
		response: json.RawMessage(`{
			"summary": "要約",
			"conversation_notes": [
				{"category": "ハプニング", "character_ids": null, "note": "なにかが起きた"}
			]
		}`),
	}
	s := summary.NewLLMProgramSummarizer(mc, "{{script_lines}}", 0)

	scr := model.Script{
		Segments: []model.ScriptSegment{
			{Type: model.SegmentTypeSpeech, SpeakerRole: "zundamon", Text: "テスト"},
		},
	}

	got, err := s.Summarize(context.Background(), scr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got.ConversationNotes) != 1 {
		t.Fatalf("ConversationNotes: got %d, want 1", len(got.ConversationNotes))
	}
	if got.ConversationNotes[0].CharacterIDs == nil {
		t.Error("CharacterIDs must not be nil (should be normalized to empty slice)")
	}
}

func TestLLMProgramSummarizer_Summarize_PromptContainsSpeakerAndText(t *testing.T) {
	mc := &mockClient{
		response: json.RawMessage(`{"summary":"要約","conversation_notes":[]}`),
	}
	s := summary.NewLLMProgramSummarizer(mc, "台本: {{script_lines}}", 0)

	scr := model.Script{
		Segments: []model.ScriptSegment{
			{Type: model.SegmentTypeSpeech, SpeakerRole: "zundamon", Text: "AIチップの話"},
			{Type: model.SegmentTypeSE, AssetName: "chime"},
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
	if !strings.Contains(prompt, "zundamon") {
		t.Errorf("prompt should contain speaker role, got: %s", prompt)
	}
	if !strings.Contains(prompt, "metan") {
		t.Errorf("prompt should contain speaker role, got: %s", prompt)
	}
}

func TestLLMProgramSummarizer_Summarize_ExcludesSESegments(t *testing.T) {
	mc := &mockClient{
		response: json.RawMessage(`{"summary":"要約","conversation_notes":[]}`),
	}
	s := summary.NewLLMProgramSummarizer(mc, "{{script_lines}}", 0)

	scr := model.Script{
		Segments: []model.ScriptSegment{
			{Type: model.SegmentTypeSE, AssetName: "start_jingle"},
			{Type: model.SegmentTypeSpeech, SpeakerRole: "zundamon", Text: "こんにちは"},
		},
	}

	_, _ = s.Summarize(context.Background(), scr)

	if len(mc.captured) == 0 {
		t.Fatal("LLM was not called")
	}
	prompt := mc.captured[0].Messages[0].Content
	if strings.Contains(prompt, "start_jingle") {
		t.Errorf("prompt should not contain SE names, got: %s", prompt)
	}
}

func TestLLMProgramSummarizer_Summarize_ReturnsEpisodeTitle(t *testing.T) {
	mc := &mockClient{
		response: json.RawMessage(`{"summary":"要約","episode_title":"今週の面白技術","conversation_notes":[]}`),
	}
	s := summary.NewLLMProgramSummarizer(mc, "{{script_lines}}", 0)

	scr := model.Script{
		Segments: []model.ScriptSegment{
			{Type: model.SegmentTypeSpeech, SpeakerRole: "zundamon", Text: "テスト"},
		},
	}

	got, err := s.Summarize(context.Background(), scr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.EpisodeTitle != "今週の面白技術" {
		t.Errorf("EpisodeTitle = %q, want %q", got.EpisodeTitle, "今週の面白技術")
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
