package write

import (
	"context"
	"encoding/json"
	"fmt"
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
          "text":         {"type": "string"}
        },
        "additionalProperties": false
      }
    }
  },
  "additionalProperties": false
}`)

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
	cornerJSON, err := json.Marshal(corner)
	if err != nil {
		return nil, fmt.Errorf("marshal corner: %w", err)
	}

	summariesJSON, err := json.Marshal(model.Summaries{Summaries: summaries})
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
		fmt.Fprintf(&sb, "- %s（%s）: 名前=%s、一人称=%s、語尾=[%s]、性格=[%s]\n",
			charID, role,
			ch.Name,
			ch.Pronoun,
			strings.Join(ch.SpeechSuffix, ", "),
			strings.Join(ch.Personality, ", "),
		)
	}
	return sb.String()
}
