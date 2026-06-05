package sel

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/canpok1/vox-radio/internal/config"
	"github.com/canpok1/vox-radio/internal/model"
	"github.com/canpok1/vox-radio/internal/script/llm"
)

var selectSchema = json.RawMessage(`{
  "type": "object",
  "required": ["selected_urls", "selection_reason"],
  "properties": {
    "selected_urls": {"type": "array", "items": {"type": "string"}, "minItems": 1},
    "selection_reason": {"type": "string"}
  },
  "additionalProperties": false
}`)

// SelectResult holds the output of a selection operation.
type SelectResult struct {
	SelectedURLs    []string
	SelectionReason string
}

// Selector selects articles from candidates and designs the talk flow for a corner.
type Selector interface {
	Select(ctx context.Context, corner config.CornerConfig, articles []model.Article) (SelectResult, error)
}

// articleForPrompt is the subset of article data passed to the LLM (body excluded to save tokens).
type articleForPrompt struct {
	URL   string `json:"url"`
	Title string `json:"title"`
}

// cornerForPrompt is the subset of corner data passed to the LLM.
type cornerForPrompt struct {
	Title                 string `json:"title"`
	Content               string `json:"content"`
	TargetDurationSeconds int    `json:"target_duration_seconds"`
}

type selectResponse struct {
	SelectedURLs    []string `json:"selected_urls"`
	SelectionReason string   `json:"selection_reason"`
}

// LLMSelector uses an LLM to select articles and design a talk flow.
type LLMSelector struct {
	client         llm.Client
	promptTemplate string
	temperature    float64
}

func NewLLMSelector(client llm.Client, promptTemplate string, temperature float64) *LLMSelector {
	return &LLMSelector{client: client, promptTemplate: promptTemplate, temperature: temperature}
}

func (s *LLMSelector) Select(ctx context.Context, corner config.CornerConfig, articles []model.Article) (SelectResult, error) {
	cp := cornerForPrompt{
		Title:                 corner.Title,
		Content:               corner.Content,
		TargetDurationSeconds: corner.LengthSec,
	}
	cornerJSON, err := json.Marshal(cp)
	if err != nil {
		return SelectResult{}, fmt.Errorf("marshal corner: %w", err)
	}

	aps := make([]articleForPrompt, len(articles))
	for i, a := range articles {
		aps[i] = articleForPrompt{URL: a.URL, Title: a.Title}
	}
	articlesJSON, err := json.Marshal(aps)
	if err != nil {
		return SelectResult{}, fmt.Errorf("marshal articles: %w", err)
	}

	prompt := strings.NewReplacer(
		"{{corner}}", string(cornerJSON),
		"{{articles}}", string(articlesJSON),
	).Replace(s.promptTemplate)

	raw, err := s.client.Complete(ctx, llm.CompletionRequest{
		Messages:    []llm.Message{{Role: "user", Content: prompt}},
		JSONSchema:  selectSchema,
		Temperature: s.temperature,
	})
	if err != nil {
		return SelectResult{}, fmt.Errorf("llm complete: %w", err)
	}

	var resp selectResponse
	if err := json.Unmarshal(raw, &resp); err != nil {
		return SelectResult{}, fmt.Errorf("unmarshal response: %w", err)
	}

	urls := resp.SelectedURLs
	if urls == nil {
		urls = make([]string, 0)
	}
	return SelectResult{SelectedURLs: urls, SelectionReason: resp.SelectionReason}, nil
}
