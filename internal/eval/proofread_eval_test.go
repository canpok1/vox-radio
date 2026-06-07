//go:build eval

package eval_test

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"runtime"
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

func loadCases(t *testing.T, filename string) []proofreadCase {
	t.Helper()
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	path := filepath.Join(filepath.Dir(thisFile), "testdata", filename)
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", filename, err)
	}
	var cases []proofreadCase
	if err := json.Unmarshal(data, &cases); err != nil {
		t.Fatalf("parse %s: %v", filename, err)
	}
	return cases
}

func loadJudgePrompt(t *testing.T) string {
	t.Helper()
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	path := filepath.Join(filepath.Dir(thisFile), "testdata", "proofread_judge.md")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read judge prompt: %v", err)
	}
	return string(data)
}

func mustJSONString(t *testing.T, v any) string {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	return string(b)
}

func runProofread(ctx context.Context, t *testing.T, client llm.Client, proofreadPrompt string, lines []proofreadLine, model string) json.RawMessage {
	t.Helper()

	linesJSON := mustJSONString(t, lines)
	prompt := strings.NewReplacer("{{lines}}", linesJSON).Replace(proofreadPrompt)

	correctionsSchema := json.RawMessage(`{
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

	req := llm.CompletionRequest{
		Messages:   []llm.Message{{Role: "user", Content: prompt}},
		JSONSchema: correctionsSchema,
	}
	if model != "" {
		req.Model = model
	}

	raw, err := client.Complete(ctx, req)
	if err != nil {
		return nil
	}
	return raw
}

func TestProofreadEval(t *testing.T) {
	apiKey := os.Getenv("GEMINI_API_KEY")
	if apiKey == "" {
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

	// Threshold.
	threshold := 4.0
	if v := os.Getenv("VOX_EVAL_PROOFREAD_THRESHOLD"); v != "" {
		if _, err := fmt.Sscanf(v, "%f", &threshold); err != nil {
			t.Fatalf("parse VOX_EVAL_PROOFREAD_THRESHOLD: %v", err)
		}
	}

	// Sample size.
	sampleSize := 8
	if v := os.Getenv("VOX_EVAL_SAMPLE_SIZE"); v != "" {
		if _, err := fmt.Sscanf(v, "%d", &sampleSize); err != nil {
			t.Fatalf("parse VOX_EVAL_SAMPLE_SIZE: %v", err)
		}
	}

	// Seed.
	seed := eval.DefaultSeed()
	if v := os.Getenv("VOX_EVAL_SAMPLE_SEED"); v != "" {
		if _, err := fmt.Sscanf(v, "%d", &seed); err != nil {
			t.Fatalf("parse VOX_EVAL_SAMPLE_SEED: %v", err)
		}
	}

	proofreadPrompt, err := eval.LoadPrompt("proofread")
	if err != nil {
		t.Fatalf("load proofread prompt: %v", err)
	}

	judgePrompt := loadJudgePrompt(t)

	// Load test sets.
	regressionCases := loadCases(t, "proofread_regression_cases.json")
	poolCases := loadCases(t, "proofread_pool_cases.json")

	sampled := eval.Sample(poolCases, sampleSize, seed, func(c proofreadCase) string { return c.Name })
	sampledNames := make([]string, len(sampled))
	for i, c := range sampled {
		sampledNames[i] = c.Name
	}
	t.Logf("seed=%d, sampled generalization cases: %v", seed, sampledNames)

	// Bundle all cases: regression (all) + generalization (sampled).
	type evaluationCase struct {
		proofreadCase
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

		// Step 1: run proofread.
		raw := runProofread(ctx, t, targetClient, proofreadPrompt, ec.Lines, targetCfg.Model)
		if raw == nil {
			// runProofread logged the error; check if inconclusive.
			t.Skipf("proofread API call failed for case %s (inconclusive)", ec.Name)
		}

		correctionsJSON := string(raw)

		// Step 2: judge.
		linesJSON := mustJSONString(t, ec.Lines)
		judgeInput := eval.JudgeInput{
			LinesJSON:       linesJSON,
			CorrectionsJSON: correctionsJSON,
			Expectation:     ec.Expectation,
		}

		judgeModel := ""
		if v := os.Getenv("VOX_EVAL_JUDGE_MODEL"); v != "" {
			judgeModel = v
		}
		scores, err := eval.Judge(ctx, judgeClient, judgePrompt, judgeInput, judgeModel)
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

		// Log per-case scores.
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
	for _, c := range eval.AllCriteria {
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
