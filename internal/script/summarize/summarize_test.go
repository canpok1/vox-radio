package summarize_test

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/canpok1/vox-radio/internal/model"
	"github.com/canpok1/vox-radio/internal/script/llm"
	"github.com/canpok1/vox-radio/internal/script/summarize"
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

func TestLLMSummarizer_Summarize_Success(t *testing.T) {
	mc := &mockClient{
		response: json.RawMessage(`{"summary":"AIチップ要約","points":["性能2倍","省電力"]}`),
	}
	s := summarize.NewLLMSummarizer(mc, "記事: {{article}}")

	article := model.Article{
		URL:   "https://example.com/1",
		Title: "AIチップ",
		Body:  "新型AIチップが登場",
	}

	got, err := s.Summarize(context.Background(), article)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.URL != article.URL {
		t.Errorf("URL: got %q, want %q", got.URL, article.URL)
	}
	if got.Summary != "AIチップ要約" {
		t.Errorf("Summary: got %q, want %q", got.Summary, "AIチップ要約")
	}
	if len(got.Points) != 2 {
		t.Errorf("Points: got %d, want 2", len(got.Points))
	}
}

func TestLLMSummarizer_Summarize_PromptContainsArticleJSON(t *testing.T) {
	mc := &mockClient{
		response: json.RawMessage(`{"summary":"要約","points":["p1"]}`),
	}
	s := summarize.NewLLMSummarizer(mc, "記事: {{article}}")

	article := model.Article{URL: "https://example.com/1", Title: "タイトル", Body: "本文"}
	_, _ = s.Summarize(context.Background(), article)

	if len(mc.captured) == 0 {
		t.Fatal("LLM was not called")
	}
	prompt := mc.captured[0].Messages[0].Content
	if len(prompt) == 0 {
		t.Fatal("prompt is empty")
	}
	// プロンプトにarticle JSONが含まれることを確認
	articleJSON, _ := json.Marshal(article)
	if !containsString(prompt, string(articleJSON)) {
		t.Errorf("prompt should contain article JSON, got: %s", prompt)
	}
}

func TestLLMSummarizer_Summarize_LLMError(t *testing.T) {
	mc := &mockClient{err: context.Canceled}
	s := summarize.NewLLMSummarizer(mc, "{{article}}")

	_, err := s.Summarize(context.Background(), model.Article{URL: "u"})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func containsString(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(s) > 0 && containsSubstring(s, sub))
}

func containsSubstring(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
