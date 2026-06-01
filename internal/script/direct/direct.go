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
        "required": ["corner_index", "after_line_index", "type", "asset_name"],
        "properties": {
          "corner_index":     {"type": "integer", "minimum": 0},
          "after_line_index": {"type": "integer", "minimum": 0},
          "type":             {"type": "string", "enum": ["se"]},
          "asset_name":       {"type": "string"},
          "reason":           {"type": "string"}
        },
        "additionalProperties": false
      }
    },
    "pause_insertions": {
      "type": "array",
      "items": {
        "type": "object",
        "required": ["corner_index", "after_line_index", "duration_sec"],
        "properties": {
          "corner_index":     {"type": "integer", "minimum": 0},
          "after_line_index": {"type": "integer", "minimum": 0},
          "duration_sec":     {"type": "number", "exclusiveMinimum": 0, "maximum": 5.0},
          "reason":           {"type": "string"}
        },
        "additionalProperties": false
      }
    }
  },
  "additionalProperties": false
}`)

// cornerLLMPayload is the subset of CornerLines sent to the LLM.
// Asset fields (StartJingle, EndJingle, BGM) are excluded because
// they are injected deterministically and must not influence SE/pause placement.
type cornerLLMPayload struct {
	Title     string       `json:"title"`
	Direction string       `json:"direction,omitempty"`
	Lines     []model.Line `json:"lines"`
}

type Director interface {
	Direct(ctx context.Context, corners []model.CornerLines, catalog model.AssetCatalog) (model.Script, error)
}

type insertion struct {
	CornerIndex    int               `json:"corner_index"`
	AfterLineIndex int               `json:"after_line_index"`
	Type           model.SegmentType `json:"type"`
	AssetName      string            `json:"asset_name"`
	Reason         string            `json:"reason,omitempty"`
}

type pauseInsertion struct {
	CornerIndex    int     `json:"corner_index"`
	AfterLineIndex int     `json:"after_line_index"`
	DurationSec    float64 `json:"duration_sec"`
	Reason         string  `json:"reason,omitempty"`
}

type insertionsResponse struct {
	Insertions      []insertion      `json:"insertions"`
	PauseInsertions []pauseInsertion `json:"pause_insertions"`
}

type LLMDirector struct {
	client         llm.Client
	promptTemplate string
	temperature    float64
}

func NewLLMDirector(client llm.Client, promptTemplate string, temperature float64) *LLMDirector {
	return &LLMDirector{client: client, promptTemplate: promptTemplate, temperature: temperature}
}

func (d *LLMDirector) Direct(ctx context.Context, corners []model.CornerLines, catalog model.AssetCatalog) (model.Script, error) {
	payload := make([]cornerLLMPayload, len(corners))
	for i, c := range corners {
		payload[i] = cornerLLMPayload{
			Title:     c.Title,
			Direction: c.Direction,
			Lines:     c.Lines,
		}
	}

	cornersJSON, err := json.Marshal(payload)
	if err != nil {
		return model.Script{}, fmt.Errorf("marshal corners: %w", err)
	}

	catalogJSON, err := json.Marshal(catalog)
	if err != nil {
		return model.Script{}, fmt.Errorf("marshal asset catalog: %w", err)
	}

	prompt := strings.NewReplacer(
		"{{corners}}", string(cornersJSON),
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

	return buildScript(corners, resp.Insertions, resp.PauseInsertions), nil
}

type insertKey struct{ cornerIdx, lineIdx int }

func buildScript(corners []model.CornerLines, insertions []insertion, pauseInsertions []pauseInsertion) model.Script {
	insertionMap := make(map[insertKey][]insertion, len(insertions))
	for _, ins := range insertions {
		key := insertKey{ins.CornerIndex, ins.AfterLineIndex}
		insertionMap[key] = append(insertionMap[key], ins)
	}

	pauseMap := make(map[insertKey][]pauseInsertion, len(pauseInsertions))
	for _, p := range pauseInsertions {
		if p.DurationSec > 0 {
			key := insertKey{p.CornerIndex, p.AfterLineIndex}
			pauseMap[key] = append(pauseMap[key], p)
		}
	}

	segments := make([]model.ScriptSegment, 0, len(insertions)+len(pauseInsertions)+len(corners)*4)

	activeBGM := ""

	for ci, corner := range corners {
		if corner.StartPauseSec > 0 {
			segments = append(segments, model.ScriptSegment{
				Type:        model.SegmentTypePause,
				DurationSec: corner.StartPauseSec,
			})
		}

		if corner.StartJingle != "" {
			segments = append(segments, model.ScriptSegment{
				Type:      model.SegmentTypeJingle,
				AssetName: corner.StartJingle,
			})
			activeBGM = ""
		}

		if corner.BGM != activeBGM {
			segments = append(segments, model.ScriptSegment{
				Type:      model.SegmentTypeBGM,
				AssetName: corner.BGM,
			})
			activeBGM = corner.BGM
		}

		for li, line := range corner.Lines {
			segments = append(segments, model.ScriptSegment{
				Type:        model.SegmentTypeSpeech,
				SpeakerRole: line.SpeakerRole,
				Style:       line.Style,
				Intonation:  line.Intonation,
				Pitch:       line.Pitch,
				Speed:       line.Speed,
				Text:        line.Text,
			})
			key := insertKey{ci, li}
			for _, ins := range insertionMap[key] {
				segments = append(segments, model.ScriptSegment{
					Type:      ins.Type,
					AssetName: ins.AssetName,
				})
			}
			for _, p := range pauseMap[key] {
				segments = append(segments, model.ScriptSegment{
					Type:        model.SegmentTypePause,
					DurationSec: p.DurationSec,
				})
			}
		}

		if corner.EndJingle != "" {
			segments = append(segments, model.ScriptSegment{
				Type:      model.SegmentTypeJingle,
				AssetName: corner.EndJingle,
			})
			activeBGM = ""
		}

		if corner.EndPauseSec > 0 {
			segments = append(segments, model.ScriptSegment{
				Type:        model.SegmentTypePause,
				DurationSec: corner.EndPauseSec,
			})
		}
	}

	return model.Script{Segments: segments}
}
