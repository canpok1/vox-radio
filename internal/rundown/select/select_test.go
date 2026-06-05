package sel_test

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/canpok1/vox-radio/internal/config"
	"github.com/canpok1/vox-radio/internal/model"
	sel "github.com/canpok1/vox-radio/internal/rundown/select"
	"github.com/canpok1/vox-radio/internal/script/llm"
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

func TestLLMSelector_Select_Success(t *testing.T) {
	mc := &mockClient{
		response: json.RawMessage(`{"selected_urls":["https://example.com/1"],"selection_reason":"AIチップ記事が最も関連性が高く、コーナーの趣旨に合致する"}`),
	}
	s := sel.NewLLMSelector(mc, "コーナー: {{corner}} 記事: {{articles}}", 0)

	corner := config.CornerConfig{Title: "テックニュース", Content: "最新技術を紹介", LengthSec: 60}
	articles := []model.Article{
		{URL: "https://example.com/1", Title: "記事1", Body: "本文1"},
		{URL: "https://example.com/2", Title: "記事2", Body: "本文2"},
	}

	got, err := s.Select(context.Background(), corner, articles)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got.SelectedURLs) != 1 {
		t.Errorf("SelectedURLs: got %d, want 1", len(got.SelectedURLs))
	}
	if got.SelectedURLs[0] != "https://example.com/1" {
		t.Errorf("SelectedURLs[0]: got %q, want %q", got.SelectedURLs[0], "https://example.com/1")
	}
	if got.SelectionReason != "AIチップ記事が最も関連性が高く、コーナーの趣旨に合致する" {
		t.Errorf("SelectionReason: got %q, want %q", got.SelectionReason, "AIチップ記事が最も関連性が高く、コーナーの趣旨に合致する")
	}
}

func TestLLMSelector_Select_PromptContainsCornerAndArticles(t *testing.T) {
	mc := &mockClient{
		response: json.RawMessage(`{"selected_urls":["https://example.com/1"],"selection_reason":"理由"}`),
	}
	s := sel.NewLLMSelector(mc, "コーナー: {{corner}} 記事: {{articles}}", 0)

	corner := config.CornerConfig{Title: "テック", Content: "内容", LengthSec: 60}
	articles := []model.Article{
		{URL: "https://example.com/1", Title: "記事1", Body: "本文1"},
	}
	_, _ = s.Select(context.Background(), corner, articles)

	if len(mc.captured) == 0 {
		t.Fatal("LLM was not called")
	}
	prompt := mc.captured[0].Messages[0].Content
	if !strings.Contains(prompt, "テック") {
		t.Errorf("prompt should contain corner title, got: %s", prompt)
	}
	if !strings.Contains(prompt, "https://example.com/1") {
		t.Errorf("prompt should contain article URL, got: %s", prompt)
	}
}

func TestLLMSelector_Select_LLMError(t *testing.T) {
	mc := &mockClient{err: context.Canceled}
	s := sel.NewLLMSelector(mc, "{{corner}} {{articles}}", 0)

	_, err := s.Select(context.Background(), config.CornerConfig{Title: "t"}, []model.Article{{URL: "u"}})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestLLMSelector_SetCasts_PromptContainsCastInfo(t *testing.T) {
	mc := &mockClient{
		response: json.RawMessage(`{"selected_urls":["https://example.com/1"],"selection_reason":"理由"}`),
	}
	s := sel.NewLLMSelector(mc, "キャスト: {{casts}} コーナー: {{corner}} 記事: {{articles}}", 0)
	s.SetCasts([]model.RundownCast{
		{CharacterID: "zundamon", Role: "MC", Type: "regular", AppearanceCount: 5},
		{CharacterID: "guest1", Role: "ゲスト", Type: "guest", AppearanceCount: 0},
	})

	corner := config.CornerConfig{Title: "テック", Content: "内容", LengthSec: 60}
	articles := []model.Article{{URL: "https://example.com/1", Title: "記事1"}}
	_, _ = s.Select(context.Background(), corner, articles)

	if len(mc.captured) == 0 {
		t.Fatal("LLM was not called")
	}
	prompt := mc.captured[0].Messages[0].Content
	if !strings.Contains(prompt, "zundamon") {
		t.Errorf("prompt should contain cast character ID, got: %s", prompt)
	}
	if !strings.Contains(prompt, "guest1") {
		t.Errorf("prompt should contain guest character ID, got: %s", prompt)
	}
	if !strings.Contains(prompt, "appearance_count") {
		t.Errorf("prompt should contain appearance_count field, got: %s", prompt)
	}
}

func TestLLMSelector_NoCasts_PromptHasEmptyArray(t *testing.T) {
	mc := &mockClient{
		response: json.RawMessage(`{"selected_urls":["https://example.com/1"],"selection_reason":"理由"}`),
	}
	s := sel.NewLLMSelector(mc, "キャスト: {{casts}} コーナー: {{corner}} 記事: {{articles}}", 0)
	// SetCasts not called

	corner := config.CornerConfig{Title: "テック", Content: "内容", LengthSec: 60}
	articles := []model.Article{{URL: "https://example.com/1", Title: "記事1"}}
	_, _ = s.Select(context.Background(), corner, articles)

	if len(mc.captured) == 0 {
		t.Fatal("LLM was not called")
	}
	prompt := mc.captured[0].Messages[0].Content
	if !strings.Contains(prompt, "[]") {
		t.Errorf("prompt should contain empty array [] for casts, got: %s", prompt)
	}
}

func TestLLMSelector_Select_PromptUsesSemanticFieldName(t *testing.T) {
	mc := &mockClient{
		response: json.RawMessage(`{"selected_urls":["https://example.com/1"],"selection_reason":"理由"}`),
	}
	s := sel.NewLLMSelector(mc, "コーナー: {{corner}} 記事: {{articles}}", 0)

	corner := config.CornerConfig{Title: "テック", Content: "内容", LengthSec: 120}
	articles := []model.Article{{URL: "https://example.com/1", Title: "記事1"}}
	_, _ = s.Select(context.Background(), corner, articles)

	if len(mc.captured) == 0 {
		t.Fatal("LLM was not called")
	}
	prompt := mc.captured[0].Messages[0].Content
	if strings.Contains(prompt, "length_sec") {
		t.Errorf("prompt must not contain internal field name 'length_sec', got: %s", prompt)
	}
	if !strings.Contains(prompt, "target_duration_seconds") {
		t.Errorf("prompt should contain semantic field name 'target_duration_seconds', got: %s", prompt)
	}
}
