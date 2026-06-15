package direct

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"

	"github.com/canpok1/vox-radio/internal/model"
	"github.com/canpok1/vox-radio/internal/script/llm"
)

var correctionsSchema = json.RawMessage(`{
  "type": "object",
  "required": ["corrections"],
  "properties": {
    "corrections": {
      "type": "array",
      "items": {
        "type": "object",
        "required": ["corner_index", "line_index", "text"],
        "properties": {
          "corner_index": {"type": "integer", "minimum": 0},
          "line_index":   {"type": "integer", "minimum": 0},
          "text":         {"type": "string"},
          "reason":       {"type": "string"}
        },
        "additionalProperties": false
      }
    }
  },
  "additionalProperties": false
}`)

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
    },
    "line_conversions": {
      "type": "array",
      "items": {
        "type": "object",
        "required": ["corner_index", "line_index", "text"],
        "properties": {
          "corner_index": {"type": "integer", "minimum": 0},
          "line_index":   {"type": "integer", "minimum": 0},
          "text":         {"type": "string"}
        },
        "additionalProperties": false
      }
    }
  },
  "additionalProperties": false
}`)

// cornerLLMPayload is the subset of CornerLines sent to the LLM.
// Asset fields (StartAudio, EndAudio, BGM) are excluded because
// they are injected deterministically and must not influence SE/pause placement.
type cornerLLMPayload struct {
	Title     string       `json:"title"`
	Direction string       `json:"direction,omitempty"`
	Lines     []model.Line `json:"lines"`
}

type Director interface {
	Direct(ctx context.Context, corners []model.CornerLines, catalog model.AssetCatalog, programDirection string) (model.Script, *model.ProofreadResult, error)
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

type lineConversion struct {
	CornerIndex int    `json:"corner_index"`
	LineIndex   int    `json:"line_index"`
	Text        string `json:"text"`
}

type insertionsResponse struct {
	Insertions      []insertion      `json:"insertions"`
	PauseInsertions []pauseInsertion `json:"pause_insertions"`
	LineConversions []lineConversion `json:"line_conversions"`
}

type correction struct {
	CornerIndex int    `json:"corner_index"`
	LineIndex   int    `json:"line_index"`
	Text        string `json:"text"`
	Reason      string `json:"reason,omitempty"`
}

type correctionsResponse struct {
	Corrections []correction `json:"corrections"`
}

// proofreadLine is the per-line input sent to the proofreading LLM.
type proofreadLine struct {
	CornerIndex   int    `json:"corner_index"`
	LineIndex     int    `json:"line_index"`
	OriginalText  string `json:"original_text"`
	ConvertedText string `json:"converted_text"`
}

type proofreadConfig struct {
	prompt      string
	temperature float64
}

type LLMDirector struct {
	client         llm.Client
	promptTemplate string
	temperature    float64
	proofread      *proofreadConfig
}

// WithProofread enables a proofreading LLM pass that detects and corrects misreadings
// in the kana conversion output. When not set, proofreading is skipped (backward-compatible).
func WithProofread(prompt string, temperature float64) func(*LLMDirector) {
	return func(d *LLMDirector) {
		d.proofread = &proofreadConfig{prompt: prompt, temperature: temperature}
	}
}

func NewLLMDirector(client llm.Client, promptTemplate string, temperature float64, opts ...func(*LLMDirector)) *LLMDirector {
	d := &LLMDirector{client: client, promptTemplate: promptTemplate, temperature: temperature}
	for _, opt := range opts {
		opt(d)
	}
	return d
}

func (d *LLMDirector) Direct(ctx context.Context, corners []model.CornerLines, catalog model.AssetCatalog, programDirection string) (model.Script, *model.ProofreadResult, error) {
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
		return model.Script{}, nil, fmt.Errorf("marshal corners: %w", err)
	}

	catalogJSON, err := json.Marshal(catalog)
	if err != nil {
		return model.Script{}, nil, fmt.Errorf("marshal asset catalog: %w", err)
	}

	prompt := strings.NewReplacer(
		"{{corners}}", string(cornersJSON),
		"{{asset_catalog}}", string(catalogJSON),
		"{{program_direction}}", stringOrNone(programDirection),
	).Replace(d.promptTemplate)

	raw, err := d.client.Complete(ctx, llm.CompletionRequest{
		Messages:    []llm.Message{{Role: "user", Content: prompt}},
		JSONSchema:  insertionsSchema,
		Temperature: d.temperature,
	})
	if err != nil {
		return model.Script{}, nil, fmt.Errorf("llm complete: %w", err)
	}

	var resp insertionsResponse
	if err := json.Unmarshal(raw, &resp); err != nil {
		return model.Script{}, nil, fmt.Errorf("unmarshal insertions: %w", err)
	}

	lineConversions := resp.LineConversions
	var pr *model.ProofreadResult

	if d.proofread != nil {
		corrections, beforeMap, err := d.runProofread(ctx, corners, lineConversions)
		if err != nil {
			slog.Default().Warn("proofread failed, using direct conversion", "err", err)
		} else {
			cs := make([]model.ProofreadCorrection, 0, len(corrections))
			for _, c := range corrections {
				lineConversions = append(lineConversions, lineConversion{
					CornerIndex: c.CornerIndex,
					LineIndex:   c.LineIndex,
					Text:        c.Text,
				})
				before := beforeMap[insertKey{c.CornerIndex, c.LineIndex}]
				cs = append(cs, model.ProofreadCorrection{
					CornerIndex: c.CornerIndex,
					LineIndex:   c.LineIndex,
					Before:      before,
					After:       c.Text,
					Reason:      c.Reason,
				})
			}
			pr = &model.ProofreadResult{Corrections: cs}
		}
	}

	return buildScript(corners, resp.Insertions, resp.PauseInsertions, lineConversions), pr, nil
}

func (d *LLMDirector) runProofread(ctx context.Context, corners []model.CornerLines, lineConversions []lineConversion) ([]correction, map[insertKey]string, error) {
	convMap := buildConversionMap(lineConversions)

	totalLines := 0
	for _, corner := range corners {
		totalLines += len(corner.Lines)
	}
	lines := make([]proofreadLine, 0, totalLines)
	for ci, corner := range corners {
		for li, line := range corner.Lines {
			converted := line.Text
			if v, ok := convMap[insertKey{ci, li}]; ok && v != "" {
				converted = v
			}
			lines = append(lines, proofreadLine{
				CornerIndex:   ci,
				LineIndex:     li,
				OriginalText:  line.Text,
				ConvertedText: converted,
			})
		}
	}

	beforeMap := make(map[insertKey]string, len(lines))
	for _, pl := range lines {
		beforeMap[insertKey{pl.CornerIndex, pl.LineIndex}] = pl.ConvertedText
	}

	linesJSON, err := json.Marshal(lines)
	if err != nil {
		return nil, nil, fmt.Errorf("marshal proofread lines: %w", err)
	}

	prompt := strings.NewReplacer("{{lines}}", string(linesJSON)).Replace(d.proofread.prompt)

	raw, err := d.client.Complete(ctx, llm.CompletionRequest{
		Messages:    []llm.Message{{Role: "user", Content: prompt}},
		JSONSchema:  correctionsSchema,
		Temperature: d.proofread.temperature,
	})
	if err != nil {
		return nil, nil, fmt.Errorf("proofread llm: %w", err)
	}

	var cr correctionsResponse
	if err := json.Unmarshal(raw, &cr); err != nil {
		return nil, nil, fmt.Errorf("unmarshal corrections: %w", err)
	}

	return cr.Corrections, beforeMap, nil
}

func stringOrNone(s string) string {
	if s == "" {
		return "（なし）"
	}
	return s
}

type insertKey struct{ cornerIdx, lineIdx int }

func buildConversionMap(lineConversions []lineConversion) map[insertKey]string {
	m := make(map[insertKey]string, len(lineConversions))
	for _, lc := range lineConversions {
		m[insertKey{lc.CornerIndex, lc.LineIndex}] = lc.Text
	}
	return m
}

func buildScript(corners []model.CornerLines, insertions []insertion, pauseInsertions []pauseInsertion, lineConversions []lineConversion) model.Script {
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

	conversionMap := buildConversionMap(lineConversions)

	segments := make([]model.ScriptSegment, 0, len(insertions)+len(pauseInsertions)+len(corners)*4)

	activeBGM := ""

	for ci, corner := range corners {
		cornerID := corner.ID
		if corner.StartPauseSec > 0 {
			segments = append(segments, model.ScriptSegment{
				Type:        model.SegmentTypePause,
				CornerID:    cornerID,
				DurationSec: corner.StartPauseSec,
			})
		}

		segments = appendBoundaryAudio(segments, corner.StartAudio, corner.BGM, &activeBGM, cornerID)

		if corner.BGM != activeBGM {
			segments = append(segments, model.ScriptSegment{
				Type:      model.SegmentTypeBGM,
				CornerID:  cornerID,
				AssetName: corner.BGM,
			})
			activeBGM = corner.BGM
		}

		for li, line := range corner.Lines {
			text := line.Text
			if converted, ok := conversionMap[insertKey{ci, li}]; ok && converted != "" {
				text = converted
			}
			segments = append(segments, model.ScriptSegment{
				Type:        model.SegmentTypeSpeech,
				CornerID:    cornerID,
				SpeakerRole: line.SpeakerRole,
				Style:       line.Style,
				Intonation:  line.Intonation,
				Pitch:       line.Pitch,
				Speed:       line.Speed,
				Text:        text,
			})
			key := insertKey{ci, li}
			for _, ins := range insertionMap[key] {
				segments = append(segments, model.ScriptSegment{
					Type:      ins.Type,
					CornerID:  cornerID,
					AssetName: ins.AssetName,
				})
			}
			for _, p := range pauseMap[key] {
				segments = append(segments, model.ScriptSegment{
					Type:        model.SegmentTypePause,
					CornerID:    cornerID,
					DurationSec: p.DurationSec,
				})
			}
		}

		segments = appendBoundaryAudio(segments, corner.EndAudio, "", &activeBGM, cornerID)

		if corner.EndPauseSec > 0 {
			segments = append(segments, model.ScriptSegment{
				Type:        model.SegmentTypePause,
				CornerID:    cornerID,
				DurationSec: corner.EndPauseSec,
			})
		}
	}

	return model.Script{Segments: segments}
}

// appendBoundaryAudio emits segments for a corner boundary audio (start or end).
// For type:jingle, emits a jingle segment and resets activeBGM.
// For type:se, pre-emits BGM (if cornerBGM is non-empty and differs from activeBGM) then emits SE without resetting activeBGM.
// Pass cornerBGM="" for end boundaries where BGM pre-emit is not desired.
func appendBoundaryAudio(segments []model.ScriptSegment, audio *model.CornerAudio, cornerBGM string, activeBGM *string, cornerID string) []model.ScriptSegment {
	if audio == nil {
		return segments
	}
	switch audio.Type {
	case model.SegmentTypeJingle:
		segments = append(segments, model.ScriptSegment{
			Type:      model.SegmentTypeJingle,
			CornerID:  cornerID,
			AssetName: audio.AssetName,
		})
		*activeBGM = ""
	case model.SegmentTypeSE:
		if cornerBGM != "" && cornerBGM != *activeBGM {
			segments = append(segments, model.ScriptSegment{
				Type:      model.SegmentTypeBGM,
				CornerID:  cornerID,
				AssetName: cornerBGM,
			})
			*activeBGM = cornerBGM
		}
		segments = append(segments, model.ScriptSegment{
			Type:      model.SegmentTypeSE,
			CornerID:  cornerID,
			AssetName: audio.AssetName,
		})
	}
	return segments
}
