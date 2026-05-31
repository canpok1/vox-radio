package summary

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
  "required": ["summary"],
  "properties": {
    "summary": {"type": "string"}
  },
  "additionalProperties": false
}`)

// ProgramSummarizer generates a summary of the episode from the script.
type ProgramSummarizer interface {
	Summarize(ctx context.Context, scr model.Script) (string, error)
}

// LLMProgramSummarizer generates a program summary using an LLM.
type LLMProgramSummarizer struct {
	client         llm.Client
	promptTemplate string
	temperature    float64
}

// NewLLMProgramSummarizer creates a new LLMProgramSummarizer.
func NewLLMProgramSummarizer(client llm.Client, promptTemplate string, temperature float64) *LLMProgramSummarizer {
	return &LLMProgramSummarizer{client: client, promptTemplate: promptTemplate, temperature: temperature}
}

type summaryResponse struct {
	Summary string `json:"summary"`
}

// Summarize generates a program summary from the speech segments of the script.
func (s *LLMProgramSummarizer) Summarize(ctx context.Context, scr model.Script) (string, error) {
	lines := make([]string, 0)
	for _, seg := range scr.Segments {
		if seg.Type == model.SegmentTypeSpeech && seg.Text != "" {
			lines = append(lines, seg.Text)
		}
	}

	linesJSON, err := json.Marshal(lines)
	if err != nil {
		return "", fmt.Errorf("marshal script lines: %w", err)
	}

	prompt := strings.ReplaceAll(s.promptTemplate, "{{script_lines}}", string(linesJSON))

	raw, err := s.client.Complete(ctx, llm.CompletionRequest{
		Messages:    []llm.Message{{Role: "user", Content: prompt}},
		JSONSchema:  summarySchema,
		Temperature: s.temperature,
	})
	if err != nil {
		return "", fmt.Errorf("llm complete: %w", err)
	}

	var resp summaryResponse
	if err := json.Unmarshal(raw, &resp); err != nil {
		return "", fmt.Errorf("unmarshal response: %w", err)
	}

	return resp.Summary, nil
}
