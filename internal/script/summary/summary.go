package summary

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"time"

	"github.com/canpok1/vox-radio/internal/model"
	"github.com/canpok1/vox-radio/internal/script/llm"
)

// summarizerCore holds common fields shared by all LLM summarizers.
type summarizerCore struct {
	logger *slog.Logger
}

// Option configures a summarizerCore (shared by LLMProgramSummarizer and LLMCornerSummarizer).
type Option func(*summarizerCore)

// WithLogger returns an option that sets the logger.
func WithLogger(l *slog.Logger) Option {
	return func(c *summarizerCore) { c.logger = l }
}

// initLogger finalises the logger by falling back to slog.Default() when no
// logger was set and then attaching the step name.
func (c *summarizerCore) initLogger(step string) {
	if c.logger == nil {
		c.logger = slog.Default()
	}
	c.logger = c.logger.With("step", step)
}

var summarySchema = json.RawMessage(`{
  "type": "object",
  "required": ["summary", "episode_title", "conversation_notes"],
  "properties": {
    "summary": {"type": "string"},
    "episode_title": {"type": "string"},
    "conversation_notes": {
      "type": "array",
      "items": {
        "type": "object",
        "required": ["category", "character_ids", "note"],
        "properties": {
          "category":      {"type": "string"},
          "character_ids": {"type": "array", "items": {"type": "string"}},
          "note":          {"type": "string"}
        },
        "additionalProperties": false
      }
    }
  },
  "additionalProperties": false
}`)

// ProgramSummarizer generates a summary of the episode from the write-step output.
type ProgramSummarizer interface {
	Summarize(ctx context.Context, lines model.ScriptLines) (model.ProgramSummary, error)
}

// LLMProgramSummarizer generates a program summary using an LLM.
type LLMProgramSummarizer struct {
	summarizerCore
	client         llm.Client
	promptTemplate string
	temperature    float64
	summaryLength  int
}

// NewLLMProgramSummarizer creates a new LLMProgramSummarizer.
func NewLLMProgramSummarizer(client llm.Client, promptTemplate string, temperature float64, summaryLength int, opts ...Option) *LLMProgramSummarizer {
	s := &LLMProgramSummarizer{
		client:         client,
		promptTemplate: promptTemplate,
		temperature:    temperature,
		summaryLength:  summaryLength,
	}
	for _, opt := range opts {
		opt(&s.summarizerCore)
	}
	s.initLogger("summary/program")
	return s
}

type speechEntry struct {
	Speaker string `json:"speaker"`
	Text    string `json:"text"`
}

// Summarize generates a program summary and conversation notes from the write-step output lines.
func (s *LLMProgramSummarizer) Summarize(ctx context.Context, lines model.ScriptLines) (model.ProgramSummary, error) {
	start := time.Now()
	s.logger.Info("開始")
	defer func() { s.logger.Info("完了", "elapsed_s", time.Since(start).Seconds()) }()

	totalLines := lines.TotalLines()
	entries := make([]speechEntry, 0, totalLines)
	for _, corner := range lines.Corners {
		for _, line := range corner.Lines {
			if line.Text != "" {
				entries = append(entries, speechEntry{Speaker: line.SpeakerRole, Text: line.Text})
			}
		}
	}

	linesJSON, err := json.Marshal(entries)
	if err != nil {
		return model.ProgramSummary{}, fmt.Errorf("marshal script lines: %w", err)
	}

	prompt := strings.NewReplacer(
		"{{script_lines}}", string(linesJSON),
		"{{summary_length}}", strconv.Itoa(s.summaryLength),
	).Replace(s.promptTemplate)

	raw, err := s.client.Complete(ctx, llm.CompletionRequest{
		Messages:    []llm.Message{{Role: "user", Content: prompt}},
		JSONSchema:  summarySchema,
		Temperature: s.temperature,
	})
	if err != nil {
		return model.ProgramSummary{}, fmt.Errorf("llm complete: %w", err)
	}

	var result model.ProgramSummary
	if err := json.Unmarshal(raw, &result); err != nil {
		return model.ProgramSummary{}, fmt.Errorf("unmarshal response: %w", err)
	}

	return result, nil
}
