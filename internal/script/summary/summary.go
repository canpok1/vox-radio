package summary

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/canpok1/vox-radio/internal/model"
	"github.com/canpok1/vox-radio/internal/script/llm"
)

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
	client         llm.Client
	promptTemplate string
	temperature    float64
	summaryLength  int
}

// NewLLMProgramSummarizer creates a new LLMProgramSummarizer.
func NewLLMProgramSummarizer(client llm.Client, promptTemplate string, temperature float64, summaryLength int) *LLMProgramSummarizer {
	return &LLMProgramSummarizer{client: client, promptTemplate: promptTemplate, temperature: temperature, summaryLength: summaryLength}
}

type speechEntry struct {
	Speaker string `json:"speaker"`
	Text    string `json:"text"`
}

// Summarize generates a program summary and conversation notes from the write-step output lines.
func (s *LLMProgramSummarizer) Summarize(ctx context.Context, lines model.ScriptLines) (model.ProgramSummary, error) {
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

	// Normalize nil slices to empty slices so JSON marshals as [] not null.
	if result.ConversationNotes == nil {
		result.ConversationNotes = make([]model.ConversationNote, 0)
	}
	for i := range result.ConversationNotes {
		if result.ConversationNotes[i].CharacterIDs == nil {
			result.ConversationNotes[i].CharacterIDs = make([]string, 0)
		}
	}

	return result, nil
}
