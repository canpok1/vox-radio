package write

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/canpok1/vox-radio/internal/model"
	"github.com/canpok1/vox-radio/internal/script/llm"
)

var linesSchema = json.RawMessage(`{
  "type": "object",
  "required": ["lines"],
  "properties": {
    "lines": {
      "type": "array",
      "minItems": 1,
      "items": {
        "type": "object",
        "required": ["speaker_role", "text"],
        "properties": {
          "speaker_role": {"type": "string"},
          "text":         {"type": "string"}
        },
        "additionalProperties": false
      }
    }
  },
  "additionalProperties": false
}`)

type Writer interface {
	Write(ctx context.Context, corner model.Corner, summaries []model.Summary, show model.ShowConfig) ([]model.Line, error)
}

type LLMWriter struct {
	client         llm.Client
	promptTemplate string
	temperature    float64
}

func NewLLMWriter(client llm.Client, promptTemplate string, temperature float64) *LLMWriter {
	return &LLMWriter{client: client, promptTemplate: promptTemplate, temperature: temperature}
}

func (w *LLMWriter) Write(ctx context.Context, corner model.Corner, summaries []model.Summary, show model.ShowConfig) ([]model.Line, error) {
	cornerJSON, err := json.Marshal(corner)
	if err != nil {
		return nil, fmt.Errorf("marshal corner: %w", err)
	}

	summariesJSON, err := json.Marshal(model.Summaries{Summaries: summaries})
	if err != nil {
		return nil, fmt.Errorf("marshal summaries: %w", err)
	}

	prompt := strings.ReplaceAll(w.promptTemplate, "{{corner}}", string(cornerJSON))
	prompt = strings.ReplaceAll(prompt, "{{summary}}", string(summariesJSON))
	prompt = strings.ReplaceAll(prompt, "{{persona}}", show.Persona)

	raw, err := w.client.Complete(ctx, llm.CompletionRequest{
		Messages:    []llm.Message{{Role: "user", Content: prompt}},
		JSONSchema:  linesSchema,
		Temperature: w.temperature,
	})
	if err != nil {
		return nil, fmt.Errorf("llm complete: %w", err)
	}

	var resp model.Lines
	if err := json.Unmarshal(raw, &resp); err != nil {
		return nil, fmt.Errorf("unmarshal lines: %w", err)
	}

	return resp.Lines, nil
}
