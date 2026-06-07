//go:build eval

package eval_test

import (
	"context"
	"encoding/json"
	"log/slog"
	"os"
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
	if os.Getenv("GEMINI_API_KEY") == "" {
		t.Skip("GEMINI_API_KEY not set")
	}

	// Build LLM clients.
	targetCfg := eval.BuildLLMConfig("GEMINI_API_KEY", "VOX_EVAL_MODEL", "VOX_EVAL_MIN_INTERVAL_MS")
	targetCfg.MaxRetries = 2
	targetClient := llm.NewClient(targetCfg)

	// Judge model can be overridden via VOX_EVAL_JUDGE_MODEL.
	judgeCfg := eval.BuildLLMConfig("GEMINI_API_KEY", "VOX_EVAL_MODEL", "VOX_EVAL_MIN_INTERVAL_MS")
	if jm := os.Getenv("VOX_EVAL_JUDGE_MODEL"); jm != "" {
		judgeCfg.Model = jm
	}
	judgeCfg.MaxRetries = 2
	judgeClient := llm.NewClient(judgeCfg)

	threshold, err := getEnvFloat("VOX_EVAL_CORNER_SUMMARY_THRESHOLD", 4.0)
	if err != nil {
		t.Fatalf("parse VOX_EVAL_CORNER_SUMMARY_THRESHOLD: %v", err)
	}

	sampleSize, err := getEnvInt("VOX_EVAL_SAMPLE_SIZE", 8)
	if err != nil {
		t.Fatalf("parse VOX_EVAL_SAMPLE_SIZE: %v", err)
	}

	seed, err := getEnvInt64("VOX_EVAL_SAMPLE_SEED", eval.DefaultSeed())
	if err != nil {
		t.Fatalf("parse VOX_EVAL_SAMPLE_SEED: %v", err)
	}

	cornerSummaryPrompt, err := eval.LoadPrompt("corner_summary")
	if err != nil {
		t.Fatalf("load corner_summary prompt: %v", err)
	}

	judgePrompt := loadTestdataString(t, "corner_summary_judge.md")

	// Load test sets.
	regressionCases := loadCasesJSON[cornerSummaryCase](t, "corner_summary_regression_cases.json")
	poolCases := loadCasesJSON[cornerSummaryCase](t, "corner_summary_pool_cases.json")

	sampled := eval.Sample(poolCases, sampleSize, seed)
	sampledNames := make([]string, len(sampled))
	for i, c := range sampled {
		sampledNames[i] = c.Name
	}
	t.Logf("seed=%d, sampled generalization cases: %v", seed, sampledNames)

	// Bundle all cases: regression (all) + generalization (sampled).
	type evaluationCase struct {
		cornerSummaryCase
		setType string
	}
	var allCases []evaluationCase
	for _, c := range regressionCases {
		allCases = append(allCases, evaluationCase{c, "regression"})
	}
	for _, c := range sampled {
		allCases = append(allCases, evaluationCase{c, "generalization"})
	}

	ctx := context.Background()
	var results []eval.CaseResult

	for _, ec := range allCases {
		t.Logf("evaluating [%s] %s ...", ec.setType, ec.Name)

		linesJSONBytes, err := json.Marshal(ec.ScriptLines)
		if err != nil {
			t.Fatalf("marshal script_lines for case %s: %v", ec.Name, err)
		}
		linesJSON := string(linesJSONBytes)

		// Step 1: run corner_summary.
		raw, err := runCornerSummary(ctx, t, targetClient, cornerSummaryPrompt, ec.cornerSummaryCase, linesJSON)
		if err != nil {
			if eval.IsInconclusive(err) {
				t.Skipf("corner_summary API call failed (inconclusive) for case %s: %v", ec.Name, err)
			}
			t.Fatalf("corner_summary failed for case %s: %v", ec.Name, err)
		}

		// Step 2: judge.
		scores, err := eval.Judge(ctx, judgeClient, judgePrompt, cornerSummaryJudgeSchema, eval.JudgeInput{
			Placeholders: map[string]string{
				"corner_title":          ec.CornerTitle,
				"script_lines":          linesJSON,
				"corner_summary_output": string(raw),
				"expectation":           eval.ResolveExpectation(ec.Expectation),
			},
		})
		if err != nil {
			if eval.IsInconclusive(err) {
				t.Skipf("judge API call failed (inconclusive): %v", err)
			}
			t.Fatalf("judge failed for case %s: %v", ec.Name, err)
		}

		results = append(results, eval.CaseResult{
			CaseName: ec.Name,
			SetType:  ec.setType,
			Scores:   scores,
		})

		for _, s := range scores {
			t.Logf("  [%s] %s: %s=%d (%s)", ec.setType, ec.Name, s.Criterion, s.Score, s.Reason)
		}
	}

	if len(results) == 0 {
		t.Skip("no results collected (all cases inconclusive)")
	}

	// Aggregate.
	agg := eval.AggregateScores(results)

	t.Logf("=== Aggregated scores ===")
	t.Logf("overall average: %.2f (threshold: %.2f)", agg.Overall, threshold)
	for _, c := range eval.AllCornerSummaryCriteria {
		t.Logf("  %s: %.2f", c, agg.ByCriterion[c])
	}

	// Check for regression failures with emphasis.
	for _, r := range results {
		if r.SetType != "regression" {
			continue
		}
		caseAgg := eval.AggregateScores([]eval.CaseResult{r})
		if caseAgg.Overall < threshold {
			t.Errorf("*** REGRESSION CASE FAILED *** [%s] overall=%.2f < threshold=%.2f", r.CaseName, caseAgg.Overall, threshold)
			for _, s := range r.Scores {
				slog.Warn("regression failure detail", "case", r.CaseName, "criterion", s.Criterion, "score", s.Score, "reason", s.Reason)
			}
		}
	}

	// Overall quality check.
	if agg.Overall < threshold {
		t.Errorf("overall average %.2f is below threshold %.2f", agg.Overall, threshold)
	}
}
