package write

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/canpok1/vox-radio/internal/config"
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
          "style":        {"type": "string"},
          "text":         {"type": "string"}
        },
        "additionalProperties": false
      }
    }
  },
  "additionalProperties": false
}`)

// cornerForPrompt is the subset of corner data passed to the LLM.
// TargetChars is computed from TargetDurationSec via config.DurationSecToTargetChars.
type cornerForPrompt struct {
	Title       string            `json:"title"`
	Content     string            `json:"content"`
	Cast        map[string]string `json:"cast"`
	TargetChars int               `json:"target_chars"`
}

type Writer interface {
	Write(ctx context.Context, corner config.CornerConfig, summaries []model.Summary, chars map[string]config.CharacterConfig) ([]model.Line, error)
}

type LLMWriter struct {
	client         llm.Client
	promptTemplate string
	temperature    float64
}

func NewLLMWriter(client llm.Client, promptTemplate string, temperature float64) *LLMWriter {
	return &LLMWriter{client: client, promptTemplate: promptTemplate, temperature: temperature}
}

func (w *LLMWriter) Write(ctx context.Context, corner config.CornerConfig, summaries []model.Summary, chars map[string]config.CharacterConfig) ([]model.Line, error) {
	promptCorner := cornerForPrompt{
		Title:       corner.Title,
		Content:     corner.Content,
		Cast:        corner.Cast,
		TargetChars: config.DurationSecToTargetChars(corner.TargetDurationSec),
	}
	cornerJSON, err := json.Marshal(promptCorner)
	if err != nil {
		return nil, fmt.Errorf("marshal corner: %w", err)
	}

	summariesJSON, err := json.Marshal(struct {
		Summaries []model.Summary `json:"summaries"`
	}{Summaries: summaries})
	if err != nil {
		return nil, fmt.Errorf("marshal summaries: %w", err)
	}

	castInfo := buildCastInfo(corner.Cast, chars)

	prompt := strings.NewReplacer(
		"{{corner}}", string(cornerJSON),
		"{{summary}}", string(summariesJSON),
		"{{cast_info}}", castInfo,
	).Replace(w.promptTemplate)

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

// buildCastInfo formats cast assignments with character catalog features for the prompt.
func buildCastInfo(cast map[string]string, chars map[string]config.CharacterConfig) string {
	var sb strings.Builder
	for charID, role := range cast {
		ch, ok := chars[charID]
		if !ok {
			fmt.Fprintf(&sb, "- %s（%s）\n", charID, role)
			continue
		}
		styleNames := make([]string, 0, len(ch.Styles))
		for s := range ch.Styles {
			styleNames = append(styleNames, s)
		}
		sort.Strings(styleNames)
		fmt.Fprintf(&sb, "- %s（%s）: 名前=%s、一人称=%s、語尾=[%s]、性格=[%s]、スタイル=[%s]（デフォルト: %s）\n",
			charID, role,
			ch.Name,
			ch.Pronoun,
			strings.Join(ch.SpeechSuffix, ", "),
			strings.Join(ch.Personality, ", "),
			strings.Join(styleNames, ", "),
			ch.DefaultStyle,
		)
	}
	return sb.String()
}
