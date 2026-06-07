//go:build eval

package eval_test

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/canpok1/vox-radio/internal/eval"
	"github.com/canpok1/vox-radio/internal/script/llm"
)

// directLine mirrors model.Line for direct eval test input.
type directLine struct {
	SpeakerRole string `json:"speaker_role"`
	Text        string `json:"text"`
}

// directCorner mirrors cornerLLMPayload passed to direct.md.
type directCorner struct {
	Title     string       `json:"title"`
	Direction string       `json:"direction,omitempty"`
	Lines     []directLine `json:"lines"`
}

// directAssetEntry mirrors model.AssetCatalogEntry.
type directAssetEntry struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
}

// directAssetCatalog mirrors model.AssetCatalog.
type directAssetCatalog struct {
	SE []directAssetEntry `json:"se"`
}

// directCase is one entry in the direct testdata files.
type directCase struct {
	Name             string             `json:"name"`
	Category         string             `json:"category"`
	ProgramDirection string             `json:"program_direction"`
	Corners          []directCorner     `json:"corners"`
	AssetCatalog     directAssetCatalog `json:"asset_catalog"`
	Expectation      string             `json:"expectation,omitempty"`
}

// directInsertion mirrors the SE insertion in direct.md output.
type directInsertion struct {
	CornerIndex    int    `json:"corner_index"`
	AfterLineIndex int    `json:"after_line_index"`
	Type           string `json:"type"`
	AssetName      string `json:"asset_name"`
	Reason         string `json:"reason,omitempty"`
}

// directPauseInsertion mirrors the pause insertion in direct.md output.
type directPauseInsertion struct {
	CornerIndex    int     `json:"corner_index"`
	AfterLineIndex int     `json:"after_line_index"`
	DurationSec    float64 `json:"duration_sec"`
	Reason         string  `json:"reason,omitempty"`
}

// directLineConversion mirrors the line conversion in direct.md output.
type directLineConversion struct {
	CornerIndex int    `json:"corner_index"`
	LineIndex   int    `json:"line_index"`
	Text        string `json:"text"`
}

// directOutput mirrors the full output of direct.md.
type directOutput struct {
	Insertions      []directInsertion      `json:"insertions"`
	PauseInsertions []directPauseInsertion `json:"pause_insertions"`
	LineConversions []directLineConversion `json:"line_conversions"`
}

// directOutputSchema is the JSON schema for the direct.md output.
var directOutputSchema = json.RawMessage(`{
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

// directJudgeSchema is the JSON schema for the direct judge LLM output.
var directJudgeSchema = json.RawMessage(`{
  "type": "object",
  "required": ["scores"],
  "properties": {
    "scores": {
      "type": "array",
      "items": {
        "type": "object",
        "required": ["criterion", "score", "reason"],
        "properties": {
          "criterion": {
            "type": "string",
            "enum": ["se_pause_placement", "index_validity", "conversion_completeness", "reading_accuracy", "content_preservation"]
          },
          "score": {"type": "integer", "minimum": 1, "maximum": 5},
          "reason": {"type": "string"}
        },
        "additionalProperties": false
      }
    }
  },
  "additionalProperties": false
}`)

// runDirect calls the direct.md LLM and returns raw JSON output.
// programDirection must already be resolved (empty → "（なし）") by the caller.
func runDirect(ctx context.Context, t *testing.T, client llm.Client, promptTemplate, cornersJSON, assetCatalogJSON, programDirection string) (json.RawMessage, error) {
	t.Helper()
	prompt := strings.NewReplacer(
		"{{corners}}", cornersJSON,
		"{{asset_catalog}}", assetCatalogJSON,
		"{{program_direction}}", programDirection,
	).Replace(promptTemplate)

	return client.Complete(ctx, llm.CompletionRequest{
		Messages:   []llm.Message{{Role: "user", Content: prompt}},
		JSONSchema: directOutputSchema,
	})
}

func TestDirectEval(t *testing.T) {
	requireGeminiKey(t)

	targetClient, judgeClient := buildEvalClients(t)

	threshold, err := getEnvFloat("VOX_EVAL_DIRECT_THRESHOLD", 4.0)
	if err != nil {
		t.Fatalf("parse VOX_EVAL_DIRECT_THRESHOLD: %v", err)
	}

	sampleSize, seed := loadSampleParams(t)

	directPrompt, err := eval.LoadPrompt("direct")
	if err != nil {
		t.Fatalf("load direct prompt: %v", err)
	}

	judgePrompt := loadTestdataString(t, "direct_judge.md")

	regressionCases := loadCasesJSON[directCase](t, "direct_regression_cases.json")
	poolCases := loadCasesJSON[directCase](t, "direct_pool_cases.json")

	allCases, caseByName := buildHarnessCases(t, regressionCases, poolCases, sampleSize, seed, func(c directCase) string { return c.Name })

	ctx := context.Background()

	runEvalHarness(ctx, t, allCases, harnessConfig{
		Criteria:    eval.AllDirectCriteria,
		JudgeClient: judgeClient,
		JudgePrompt: judgePrompt,
		JudgeSchema: directJudgeSchema,
		Threshold:   threshold,
		RunCase: func(ctx context.Context, t *testing.T, c harnessCase) (map[string]string, error) {
			ec := caseByName[c.Name]

			resolvedDirection := ec.ProgramDirection
			if strings.TrimSpace(resolvedDirection) == "" {
				resolvedDirection = "（なし）"
			}

			cornersJSON, err := json.Marshal(ec.Corners)
			if err != nil {
				return nil, fmt.Errorf("marshal corners for case %s: %w", c.Name, err)
			}
			assetCatalogJSON, err := json.Marshal(ec.AssetCatalog)
			if err != nil {
				return nil, fmt.Errorf("marshal asset_catalog for case %s: %w", c.Name, err)
			}

			raw, err := runDirect(ctx, t, targetClient, directPrompt, string(cornersJSON), string(assetCatalogJSON), resolvedDirection)
			if err != nil {
				return nil, err
			}

			var output directOutput
			if err := json.Unmarshal(raw, &output); err != nil {
				return nil, fmt.Errorf("unmarshal direct output for case %s: %w", c.Name, err)
			}

			// Mechanical verification: index bounds and asset names.
			seNames := make(map[string]bool, len(ec.AssetCatalog.SE))
			for _, e := range ec.AssetCatalog.SE {
				seNames[e.Name] = true
			}

			checkAfterLine := func(label string, cornerIdx, afterLineIdx int) bool {
				if cornerIdx < 0 || cornerIdx >= len(ec.Corners) {
					t.Errorf("*** CONSTRAINT VIOLATION *** [%s] %s.corner_index=%d out of range [0,%d)",
						c.Name, label, cornerIdx, len(ec.Corners))
					return false
				}
				if afterLineIdx < 0 || afterLineIdx >= len(ec.Corners[cornerIdx].Lines) {
					t.Errorf("*** CONSTRAINT VIOLATION *** [%s] %s.after_line_index=%d out of range [0,%d) for corner_index=%d",
						c.Name, label, afterLineIdx, len(ec.Corners[cornerIdx].Lines), cornerIdx)
				}
				return true
			}

			for i, ins := range output.Insertions {
				if !checkAfterLine(fmt.Sprintf("insertions[%d]", i), ins.CornerIndex, ins.AfterLineIndex) {
					continue
				}
				if !seNames[ins.AssetName] {
					t.Errorf("*** CONSTRAINT VIOLATION *** [%s] insertions[%d].asset_name=%q not in SE catalog",
						c.Name, i, ins.AssetName)
				}
			}

			for i, p := range output.PauseInsertions {
				checkAfterLine(fmt.Sprintf("pause_insertions[%d]", i), p.CornerIndex, p.AfterLineIndex)
			}

			// Mechanical verification: line_conversion indices must be in bounds.
			for i, lc := range output.LineConversions {
				if lc.CornerIndex < 0 || lc.CornerIndex >= len(ec.Corners) {
					t.Errorf("*** CONSTRAINT VIOLATION *** [%s] line_conversions[%d].corner_index=%d out of range [0,%d)",
						c.Name, i, lc.CornerIndex, len(ec.Corners))
					continue
				}
				if lc.LineIndex < 0 || lc.LineIndex >= len(ec.Corners[lc.CornerIndex].Lines) {
					t.Errorf("*** CONSTRAINT VIOLATION *** [%s] line_conversions[%d].line_index=%d out of range [0,%d) for corner_index=%d",
						c.Name, i, lc.LineIndex, len(ec.Corners[lc.CornerIndex].Lines), lc.CornerIndex)
				}
			}

			// Mechanical verification: all lines must have a conversion.
			type lineKey struct{ cornerIdx, lineIdx int }
			covered := make(map[lineKey]bool, len(output.LineConversions))
			for _, lc := range output.LineConversions {
				covered[lineKey{lc.CornerIndex, lc.LineIndex}] = true
			}
			for ci, corner := range ec.Corners {
				for li := range corner.Lines {
					if !covered[lineKey{ci, li}] {
						t.Errorf("*** CONSTRAINT VIOLATION *** [%s] missing line_conversion for corner_index=%d line_index=%d",
							c.Name, ci, li)
					}
				}
			}

			return map[string]string{
				"program_direction": resolvedDirection,
				"corners":           string(cornersJSON),
				"asset_catalog":     string(assetCatalogJSON),
				"direct_output":     string(raw),
				"expectation":       eval.ResolveExpectation(ec.Expectation),
			}, nil
		},
	})
}
