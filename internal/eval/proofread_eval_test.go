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

// proofreadLine mirrors the input format for proofread.md.
type proofreadLine struct {
	CornerIndex   int    `json:"corner_index"`
	LineIndex     int    `json:"line_index"`
	OriginalText  string `json:"original_text"`
	ConvertedText string `json:"converted_text"`
}

// proofreadCase is one entry in the testdata files.
type proofreadCase struct {
	Name        string          `json:"name"`
	Category    string          `json:"category"`
	Lines       []proofreadLine `json:"lines"`
	Expectation string          `json:"expectation,omitempty"`
}

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

var proofreadJudgeSchema = json.RawMessage(`{
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
            "enum": ["detection_recall", "false_positive_suppression", "correction_accuracy", "reason_validity"]
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

func runProofread(ctx context.Context, t *testing.T, client llm.Client, proofreadPrompt, linesJSON string) (json.RawMessage, error) {
	t.Helper()
	prompt := strings.NewReplacer("{{lines}}", linesJSON).Replace(proofreadPrompt)
	return client.Complete(ctx, llm.CompletionRequest{
		Messages:   []llm.Message{{Role: "user", Content: prompt}},
		JSONSchema: correctionsSchema,
	})
}

func TestProofreadEval(t *testing.T) {
	requireGeminiKey(t)

	targetClient, judgeClient := buildEvalClients(t)

	threshold, err := getEnvFloat("VOX_EVAL_PROOFREAD_THRESHOLD", 4.0)
	if err != nil {
		t.Fatalf("parse VOX_EVAL_PROOFREAD_THRESHOLD: %v", err)
	}

	sampleSize, seed := loadSampleParams(t)

	proofreadPrompt, err := eval.LoadPrompt("proofread")
	if err != nil {
		t.Fatalf("load proofread prompt: %v", err)
	}

	judgePrompt := loadTestdataString(t, "proofread_judge.md")

	regressionCases := loadCasesJSON[proofreadCase](t, "proofread_regression_cases.json")
	poolCases := loadCasesJSON[proofreadCase](t, "proofread_pool_cases.json")

	sampled := eval.Sample(poolCases, sampleSize, seed)
	sampledNames := make([]string, len(sampled))
	for i, c := range sampled {
		sampledNames[i] = c.Name
	}
	t.Logf("seed=%d, sampled generalization cases: %v", seed, sampledNames)

	caseByName := make(map[string]proofreadCase, len(regressionCases)+len(sampled))
	var allCases []harnessCase
	for _, c := range regressionCases {
		caseByName[c.Name] = c
		allCases = append(allCases, harnessCase{c.Name, "regression"})
	}
	for _, c := range sampled {
		caseByName[c.Name] = c
		allCases = append(allCases, harnessCase{c.Name, "generalization"})
	}

	ctx := context.Background()

	runEvalHarness(ctx, t, allCases, harnessConfig{
		Criteria:    eval.AllCriteria,
		JudgeClient: judgeClient,
		JudgePrompt: judgePrompt,
		JudgeSchema: proofreadJudgeSchema,
		Threshold:   threshold,
		RunCase: func(ctx context.Context, t *testing.T, c harnessCase) (map[string]string, error) {
			ec := caseByName[c.Name]
			linesJSONBytes, err := json.Marshal(ec.Lines)
			if err != nil {
				return nil, fmt.Errorf("marshal lines for case %s: %w", c.Name, err)
			}
			linesJSON := string(linesJSONBytes)

			raw, err := runProofread(ctx, t, targetClient, proofreadPrompt, linesJSON)
			if err != nil {
				return nil, err
			}

			return map[string]string{
				"lines":       linesJSON,
				"corrections": string(raw),
				"expectation": eval.ResolveExpectation(ec.Expectation),
			}, nil
		},
	})
}
