package direct

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/canpok1/vox-radio/internal/model"
	"github.com/canpok1/vox-radio/internal/script/llm"
)

var insertionsSchema = json.RawMessage(`{
  "type": "object",
  "required": ["insertions"],
  "properties": {
    "insertions": {
      "type": "array",
      "items": {
        "type": "object",
        "required": ["after_line_index", "type", "asset_name"],
        "properties": {
          "after_line_index": {"type": "integer", "minimum": 0},
          "type":             {"type": "string", "enum": ["se", "bgm", "jingle"]},
          "asset_name":       {"type": "string"},
          "reason":           {"type": "string"}
        },
        "additionalProperties": false
      }
    }
  },
  "additionalProperties": false
}`)

type Director interface {
	Direct(ctx context.Context, lines []model.Line, catalog model.AssetCatalog) (model.Script, error)
}

type insertion struct {
	AfterLineIndex int               `json:"after_line_index"`
	Type           model.SegmentType `json:"type"`
	AssetName      string            `json:"asset_name"`
	Reason         string            `json:"reason,omitempty"`
}

type insertionsResponse struct {
	Insertions []insertion `json:"insertions"`
}

type LLMDirector struct {
	client         llm.Client
	promptTemplate string
	temperature    float64
}

func NewLLMDirector(client llm.Client, promptTemplate string, temperature float64) *LLMDirector {
	return &LLMDirector{client: client, promptTemplate: promptTemplate, temperature: temperature}
}

func (d *LLMDirector) Direct(ctx context.Context, lines []model.Line, catalog model.AssetCatalog) (model.Script, error) {
	linesJSON, err := json.Marshal(model.Lines{Lines: lines})
	if err != nil {
		return model.Script{}, fmt.Errorf("marshal lines: %w", err)
	}

	catalogJSON, err := json.Marshal(catalog)
	if err != nil {
		return model.Script{}, fmt.Errorf("marshal asset catalog: %w", err)
	}

	prompt := strings.NewReplacer(
		"{{lines}}", string(linesJSON),
		"{{asset_catalog}}", string(catalogJSON),
	).Replace(d.promptTemplate)

	raw, err := d.client.Complete(ctx, llm.CompletionRequest{
		Messages:    []llm.Message{{Role: "user", Content: prompt}},
		JSONSchema:  insertionsSchema,
		Temperature: d.temperature,
	})
	if err != nil {
		return model.Script{}, fmt.Errorf("llm complete: %w", err)
	}

	var resp insertionsResponse
	if err := json.Unmarshal(raw, &resp); err != nil {
		return model.Script{}, fmt.Errorf("unmarshal insertions: %w", err)
	}

	return buildScript(lines, resp.Insertions), nil
}

func buildScript(lines []model.Line, insertions []insertion) model.Script {
	insertionMap := make(map[int][]insertion, len(insertions))
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
				Type:      ins.Type,
				AssetName: ins.AssetName,
			})
		}
	}

	return model.Script{Segments: segments}
}
