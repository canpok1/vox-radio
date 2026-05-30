package summarize

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/canpok1/vox-radio/internal/model"
	"github.com/canpok1/vox-radio/internal/script/llm"
)

var summarySchema = json.RawMessage(`{
  "type": "object",
  "required": ["summary", "points"],
  "properties": {
    "summary": {"type": "string"},
    "points": {"type": "array", "items": {"type": "string"}}
  },
  "additionalProperties": false
}`)

type Summarizer interface {
	Summarize(ctx context.Context, a model.Article) (model.Summary, error)
}

type LLMSummarizer struct {
	client         llm.Client
	promptTemplate string
	temperature    float64
}

func NewLLMSummarizer(client llm.Client, promptTemplate string, temperature float64) *LLMSummarizer {
	return &LLMSummarizer{client: client, promptTemplate: promptTemplate, temperature: temperature}
}

type summaryResponse struct {
	Summary string   `json:"summary"`
	Points  []string `json:"points"`
}

func (s *LLMSummarizer) Summarize(ctx context.Context, a model.Article) (model.Summary, error) {
	articleJSON, err := json.Marshal(a)
	if err != nil {
		return model.Summary{}, fmt.Errorf("marshal article: %w", err)
	}

	prompt := strings.ReplaceAll(s.promptTemplate, "{{article}}", string(articleJSON))

	raw, err := s.client.Complete(ctx, llm.CompletionRequest{
		Messages:    []llm.Message{{Role: "user", Content: prompt}},
		JSONSchema:  summarySchema,
		Temperature: s.temperature,
	})
	if err != nil {
		return model.Summary{}, fmt.Errorf("llm complete: %w", err)
	}

	var resp summaryResponse
	if err := json.Unmarshal(raw, &resp); err != nil {
		return model.Summary{}, fmt.Errorf("unmarshal response: %w", err)
	}

	return model.Summary{
		URL:     a.URL,
		Summary: resp.Summary,
		Points:  resp.Points,
	}, nil
}
