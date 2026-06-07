//go:build eval

package eval_test

import (
	"context"
	"encoding/json"
	"log/slog"
	"os"
	"strings"
	"testing"

	"github.com/canpok1/vox-radio/internal/eval"
	"github.com/canpok1/vox-radio/internal/script/llm"
)

// summarizeArticle mirrors the input format for summarize.md.
type summarizeArticle struct {
	Title string `json:"title"`
	Body  string `json:"body"`
}

// summarizeCase is one entry in the summarize testdata files.
type summarizeCase struct {
	Name        string           `json:"name"`
	Category    string           `json:"category"`
	Article     summarizeArticle `json:"article"`
	Expectation string           `json:"expectation,omitempty"`
}

// summarizeSchema is the JSON schema for the summarize.md output.
var summarizeSchema = json.RawMessage(`{
  "type": "object",
  "required": ["summary", "points"],
  "properties": {
    "summary": {"type": "string"},
    "points": {
      "type": "array",
      "items": {"type": "string"}
    }
  },
  "additionalProperties": false
}`)

// summarizeJudgeSchema is the JSON schema for the summarize judge LLM output.
var summarizeJudgeSchema = json.RawMessage(`{
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
            "enum": ["faithfulness", "coverage", "conciseness", "format_compliance"]
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

func runSummarize(ctx context.Context, t *testing.T, client llm.Client, summarizePrompt string, articleJSON string) (json.RawMessage, error) {
	t.Helper()
	prompt := strings.NewReplacer("{{article}}", articleJSON).Replace(summarizePrompt)
	return client.Complete(ctx, llm.CompletionRequest{
		Messages:   []llm.Message{{Role: "user", Content: prompt}},
		JSONSchema: summarizeSchema,
	})
}

func TestSummarizeEval(t *testing.T) {
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

	threshold, err := getEnvFloat("VOX_EVAL_SUMMARIZE_THRESHOLD", 4.0)
	if err != nil {
		t.Fatalf("parse VOX_EVAL_SUMMARIZE_THRESHOLD: %v", err)
	}

	sampleSize, err := getEnvInt("VOX_EVAL_SAMPLE_SIZE", 8)
	if err != nil {
		t.Fatalf("parse VOX_EVAL_SAMPLE_SIZE: %v", err)
	}

	seed, err := getEnvInt64("VOX_EVAL_SAMPLE_SEED", eval.DefaultSeed())
	if err != nil {
		t.Fatalf("parse VOX_EVAL_SAMPLE_SEED: %v", err)
	}

	summarizePrompt, err := eval.LoadPrompt("summarize")
	if err != nil {
		t.Fatalf("load summarize prompt: %v", err)
	}

	judgePrompt := loadTestdataString(t, "summarize_judge.md")

	// Load test sets.
	regressionCases := loadCasesJSON[summarizeCase](t, "summarize_regression_cases.json")
	poolCases := loadCasesJSON[summarizeCase](t, "summarize_pool_cases.json")

	sampled := eval.Sample(poolCases, sampleSize, seed)
	sampledNames := make([]string, len(sampled))
	for i, c := range sampled {
		sampledNames[i] = c.Name
	}
	t.Logf("seed=%d, sampled generalization cases: %v", seed, sampledNames)

	// Bundle all cases: regression (all) + generalization (sampled).
	type evaluationCase struct {
		summarizeCase
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

		// Marshal article for use in summarize prompt and judge input.
		articleJSONBytes, err := json.Marshal(ec.Article)
		if err != nil {
			t.Fatalf("marshal article for case %s: %v", ec.Name, err)
		}
		articleJSON := string(articleJSONBytes)

		// Step 1: run summarize.
		raw, err := runSummarize(ctx, t, targetClient, summarizePrompt, articleJSON)
		if err != nil {
			if eval.IsInconclusive(err) {
				t.Skipf("summarize API call failed (inconclusive) for case %s: %v", ec.Name, err)
			}
			t.Fatalf("summarize failed for case %s: %v", ec.Name, err)
		}

		// Step 2: judge.
		scores, err := eval.Judge(ctx, judgeClient, judgePrompt, summarizeJudgeSchema, eval.JudgeInput{
			Placeholders: map[string]string{
				"article":        articleJSON,
				"summary_output": string(raw),
				"expectation":    eval.ResolveExpectation(ec.Expectation),
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
	for _, c := range eval.AllSummarizeCriteria {
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
