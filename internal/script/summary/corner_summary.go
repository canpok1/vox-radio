package summary

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/canpok1/vox-radio/internal/model"
	"github.com/canpok1/vox-radio/internal/script/llm"
)

var cornerSummarySchema = json.RawMessage(`{
  "type": "object",
  "required": ["summary", "points"],
  "properties": {
    "summary": {"type": "string"},
    "points": {"type": "array", "items": {"type": "string"}}
  },
  "additionalProperties": false
}`)

// LLMCornerSummarizer generates a corner summary using an LLM.
type LLMCornerSummarizer struct {
	summarizerCore
	client         llm.Client
	promptTemplate string
	temperature    float64
}

// NewLLMCornerSummarizer creates a new LLMCornerSummarizer.
func NewLLMCornerSummarizer(client llm.Client, promptTemplate string, temperature float64, opts ...Option) *LLMCornerSummarizer {
	s := &LLMCornerSummarizer{
		client:         client,
		promptTemplate: promptTemplate,
		temperature:    temperature,
	}
	for _, opt := range opts {
		opt(&s.summarizerCore)
	}
	s.initLogger("summary/corner")
	return s
}

type cornerSummaryResponse struct {
	Summary string   `json:"summary"`
	Points  []string `json:"points"`
}

// SummarizeCorner generates a summary and points for a single corner from its script lines.
// summaryLength specifies the target character count for the summary.
func (s *LLMCornerSummarizer) SummarizeCorner(ctx context.Context, corner model.CornerLines, summaryLength int) (model.CornerSummary, error) {
	title := corner.Title
	start := time.Now()
	s.logger.Info("開始", "corner", title)
	defer func() { s.logger.Info("完了", "corner", title, "elapsed_s", time.Since(start).Seconds()) }()

	lines := make([]string, 0, len(corner.Lines))
	for _, l := range corner.Lines {
		if l.Text != "" {
			lines = append(lines, l.Text)
		}
	}

	linesJSON, err := json.Marshal(lines)
	if err != nil {
		return model.CornerSummary{}, fmt.Errorf("marshal corner lines: %w", err)
	}

	prompt := strings.NewReplacer(
		"{{corner_title}}", corner.Title,
		"{{script_lines}}", string(linesJSON),
		"{{summary_length}}", strconv.Itoa(summaryLength),
	).Replace(s.promptTemplate)

	raw, err := s.client.Complete(ctx, llm.CompletionRequest{
		Messages:    []llm.Message{{Role: "user", Content: prompt}},
		JSONSchema:  cornerSummarySchema,
		Temperature: s.temperature,
	})
	if err != nil {
		return model.CornerSummary{}, fmt.Errorf("llm complete: %w", err)
	}

	var resp cornerSummaryResponse
	if err := json.Unmarshal(raw, &resp); err != nil {
		return model.CornerSummary{}, fmt.Errorf("unmarshal response: %w", err)
	}

	return model.NewCornerSummary(resp.Summary, resp.Points), nil
}
