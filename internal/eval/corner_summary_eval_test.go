//go:build eval

package eval_test

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"testing"

	"github.com/canpok1/vox-radio/internal/eval"
	"github.com/canpok1/vox-radio/internal/script/llm"
)

// cornerSummaryCase is one entry in the corner_summary testdata files.
type cornerSummaryCase struct {
	Name          string   `json:"name"`
	Category      string   `json:"category"`
	CornerTitle   string   `json:"corner_title"`
	ScriptLines   []string `json:"script_lines"`
	SummaryLength int      `json:"summary_length"`
	Expectation   string   `json:"expectation,omitempty"`
}

// cornerSummaryJudgeSchema is the JSON schema for the corner_summary judge LLM output.
var cornerSummaryJudgeSchema = json.RawMessage(`{
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
            "enum": ["faithfulness", "coverage", "specificity", "format_compliance"]
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

func runCornerSummary(ctx context.Context, t *testing.T, client llm.Client, promptTemplate string, ec cornerSummaryCase, linesJSON string) (json.RawMessage, error) {
	t.Helper()
	prompt := strings.NewReplacer(
		"{{corner_title}}", ec.CornerTitle,
		"{{script_lines}}", linesJSON,
		"{{summary_length}}", strconv.Itoa(ec.SummaryLength),
	).Replace(promptTemplate)
	return client.Complete(ctx, llm.CompletionRequest{
		Messages:   []llm.Message{{Role: "user", Content: prompt}},
		JSONSchema: summarizeSchema,
	})
}

func TestCornerSummaryEval(t *testing.T) {
	requireGeminiKey(t)

	targetClient, judgeClient := buildEvalClients(t)

	threshold, err := getEnvFloat("VOX_EVAL_CORNER_SUMMARY_THRESHOLD", 4.0)
	if err != nil {
		t.Fatalf("parse VOX_EVAL_CORNER_SUMMARY_THRESHOLD: %v", err)
	}

	sampleSize, seed := loadSampleParams(t)

	cornerSummaryPrompt, err := eval.LoadPrompt("corner_summary")
	if err != nil {
		t.Fatalf("load corner_summary prompt: %v", err)
	}

	judgePrompt := loadTestdataString(t, "corner_summary_judge.md")

	regressionCases := loadCasesJSON[cornerSummaryCase](t, "corner_summary_regression_cases.json")
	poolCases := loadCasesJSON[cornerSummaryCase](t, "corner_summary_pool_cases.json")

	allCases, caseByName := buildHarnessCases(t, regressionCases, poolCases, sampleSize, seed, func(c cornerSummaryCase) string { return c.Name })

	ctx := context.Background()

	runEvalHarness(ctx, t, allCases, harnessConfig{
		Criteria:    eval.AllCornerSummaryCriteria,
		JudgeClient: judgeClient,
		JudgePrompt: judgePrompt,
		JudgeSchema: cornerSummaryJudgeSchema,
		Threshold:   threshold,
		RunCase: func(ctx context.Context, t *testing.T, c harnessCase) (map[string]string, error) {
			ec := caseByName[c.Name]
			linesJSONBytes, err := json.Marshal(ec.ScriptLines)
			if err != nil {
				return nil, fmt.Errorf("marshal script_lines for case %s: %w", c.Name, err)
			}
			linesJSON := string(linesJSONBytes)

			raw, err := runCornerSummary(ctx, t, targetClient, cornerSummaryPrompt, ec, linesJSON)
			if err != nil {
				return nil, err
			}

			return map[string]string{
				"corner_title":          ec.CornerTitle,
				"script_lines":          linesJSON,
				"corner_summary_output": string(raw),
				"expectation":           eval.ResolveExpectation(ec.Expectation),
			}, nil
		},
	})
}
