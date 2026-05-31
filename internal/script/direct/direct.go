package direct

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/canpok1/vox-radio/internal/model"
	"github.com/canpok1/vox-radio/internal/script/llm"
)

var seInsertionsSchema = json.RawMessage(`{
  "type": "object",
  "required": ["se_insertions"],
  "properties": {
    "se_insertions": {
      "type": "array",
      "items": {
        "type": "object",
        "required": ["after_line_index", "se_name"],
        "properties": {
          "after_line_index": {"type": "integer", "minimum": 0},
          "se_name":          {"type": "string"},
          "reason":           {"type": "string"}
        },
        "additionalProperties": false
      }
    }
  },
  "additionalProperties": false
}`)

type Director interface {
	Direct(ctx context.Context, lines []model.Line, se model.SECatalog) (model.Script, error)
}

type seInsertion struct {
	AfterLineIndex int    `json:"after_line_index"`
	SEName         string `json:"se_name"`
	Reason         string `json:"reason,omitempty"`
}

type seInsertionsResponse struct {
	SEInsertions []seInsertion `json:"se_insertions"`
}

type LLMDirector struct {
	client         llm.Client
	promptTemplate string
	temperature    float64
}

func NewLLMDirector(client llm.Client, promptTemplate string, temperature float64) *LLMDirector {
	return &LLMDirector{client: client, promptTemplate: promptTemplate, temperature: temperature}
}

func (d *LLMDirector) Direct(ctx context.Context, lines []model.Line, se model.SECatalog) (model.Script, error) {
	linesJSON, err := json.Marshal(model.Lines{Lines: lines})
	if err != nil {
		return model.Script{}, fmt.Errorf("marshal lines: %w", err)
	}

	seJSON, err := json.Marshal(se)
	if err != nil {
		return model.Script{}, fmt.Errorf("marshal se catalog: %w", err)
	}

	prompt := strings.ReplaceAll(d.promptTemplate, "{{lines}}", string(linesJSON))
	prompt = strings.ReplaceAll(prompt, "{{se_catalog}}", string(seJSON))

	raw, err := d.client.Complete(ctx, llm.CompletionRequest{
		Messages:    []llm.Message{{Role: "user", Content: prompt}},
		JSONSchema:  seInsertionsSchema,
		Temperature: d.temperature,
	})
	if err != nil {
		return model.Script{}, fmt.Errorf("llm complete: %w", err)
	}

	var resp seInsertionsResponse
	if err := json.Unmarshal(raw, &resp); err != nil {
		return model.Script{}, fmt.Errorf("unmarshal se insertions: %w", err)
	}

	return buildScript(lines, resp.SEInsertions), nil
}

func buildScript(lines []model.Line, insertions []seInsertion) model.Script {
	insertionMap := make(map[int][]seInsertion, len(insertions))
	for _, ins := range insertions {
		insertionMap[ins.AfterLineIndex] = append(insertionMap[ins.AfterLineIndex], ins)
	}

	segments := make([]model.ScriptSegment, 0, len(lines)+len(insertions))
	for i, line := range lines {
		segments = append(segments, model.ScriptSegment{
			Type:        model.SegmentTypeSpeech,
			SpeakerRole: line.SpeakerRole,
			Style:       line.Style,
			Text:        line.Text,
		})
		for _, ins := range insertionMap[i] {
			segments = append(segments, model.ScriptSegment{
				Type:   model.SegmentTypeSE,
				SEName: ins.SEName,
			})
		}
	}

	return model.Script{Segments: segments}
}
