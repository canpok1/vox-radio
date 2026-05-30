package plan

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/canpok1/vox-radio/internal/model"
	"github.com/canpok1/vox-radio/internal/script/llm"
)

var rundownSchema = json.RawMessage(`{
  "type": "object",
  "required": ["corners"],
  "properties": {
    "corners": {
      "type": "array",
      "minItems": 1,
      "items": {
        "type": "object",
        "required": ["title", "topic", "points", "target_chars", "summary_urls"],
        "properties": {
          "title":       {"type": "string"},
          "topic":       {"type": "string"},
          "points":      {"type": "array", "items": {"type": "string"}},
          "target_chars": {"type": "integer"},
          "summary_urls": {"type": "array", "items": {"type": "string"}}
        },
        "additionalProperties": false
      }
    }
  },
  "additionalProperties": false
}`)

type Planner interface {
	Plan(ctx context.Context, summaries []model.Summary, show model.ShowConfig) (model.Rundown, error)
}

type LLMPlanner struct {
	client         llm.Client
	promptTemplate string
}

func NewLLMPlanner(client llm.Client, promptTemplate string) *LLMPlanner {
	return &LLMPlanner{client: client, promptTemplate: promptTemplate}
}

func (p *LLMPlanner) Plan(ctx context.Context, summaries []model.Summary, show model.ShowConfig) (model.Rundown, error) {
	summariesJSON, err := json.Marshal(model.Summaries{Summaries: summaries})
	if err != nil {
		return model.Rundown{}, fmt.Errorf("marshal summaries: %w", err)
	}

	showJSON, err := json.Marshal(show)
	if err != nil {
		return model.Rundown{}, fmt.Errorf("marshal show config: %w", err)
	}

	prompt := strings.ReplaceAll(p.promptTemplate, "{{summaries}}", string(summariesJSON))
	prompt = strings.ReplaceAll(prompt, "{{show_config}}", string(showJSON))

	raw, err := p.client.Complete(ctx, llm.CompletionRequest{
		Messages:   []llm.Message{{Role: "user", Content: prompt}},
		JSONSchema: rundownSchema,
	})
	if err != nil {
		return model.Rundown{}, fmt.Errorf("llm complete: %w", err)
	}

	var rundown model.Rundown
	if err := json.Unmarshal(raw, &rundown); err != nil {
		return model.Rundown{}, fmt.Errorf("unmarshal rundown: %w", err)
	}

	return rundown, nil
}
