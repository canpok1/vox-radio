//go:build eval

package eval_test

import (
	"context"
	"encoding/json"
	"log/slog"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"testing"

	"github.com/canpok1/vox-radio/internal/eval"
	"github.com/canpok1/vox-radio/internal/script/llm"
)

var testdataDir string

func init() {
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		panic("runtime.Caller failed")
	}
	testdataDir = filepath.Join(filepath.Dir(thisFile), "testdata")
}

func testdataPath(filename string) string {
	return filepath.Join(testdataDir, filename)
}

func loadTestdataString(t *testing.T, filename string) string {
	t.Helper()
	data, err := os.ReadFile(testdataPath(filename))
	if err != nil {
		t.Fatalf("read %s: %v", filename, err)
	}
	return string(data)
}

func loadCasesJSON[T any](t *testing.T, filename string) []T {
	t.Helper()
	data, err := os.ReadFile(testdataPath(filename))
	if err != nil {
		t.Fatalf("read %s: %v", filename, err)
	}
	var cases []T
	if err := json.Unmarshal(data, &cases); err != nil {
		t.Fatalf("parse %s: %v", filename, err)
	}
	return cases
}

func getEnvFloat(key string, defaultVal float64) (float64, error) {
	v := os.Getenv(key)
	if v == "" {
		return defaultVal, nil
	}
	return strconv.ParseFloat(v, 64)
}

func getEnvInt(key string, defaultVal int) (int, error) {
	v := os.Getenv(key)
	if v == "" {
		return defaultVal, nil
	}
	n, err := strconv.Atoi(v)
	return n, err
}

func getEnvInt64(key string, defaultVal int64) (int64, error) {
	v := os.Getenv(key)
	if v == "" {
		return defaultVal, nil
	}
	return strconv.ParseInt(v, 10, 64)
}

// buildEvalClients creates target and judge LLM clients from env vars.
func buildEvalClients(t *testing.T) (llm.Client, llm.Client) {
	t.Helper()

	targetCfg := eval.BuildLLMConfig("GEMINI_API_KEY", "VOX_EVAL_MODEL", "VOX_EVAL_MIN_INTERVAL_MS")
	targetCfg.MaxRetries = 2
	targetClient := llm.NewClient(targetCfg)

	judgeCfg := eval.BuildLLMConfig("GEMINI_API_KEY", "VOX_EVAL_MODEL", "VOX_EVAL_MIN_INTERVAL_MS")
	if jm := os.Getenv("VOX_EVAL_JUDGE_MODEL"); jm != "" {
		judgeCfg.Model = jm
	}
	judgeCfg.MaxRetries = 2
	judgeClient := llm.NewClient(judgeCfg)

	return targetClient, judgeClient
}

// harnessCase holds the per-case info needed by the eval harness.
type harnessCase struct {
	Name    string
	SetType string
}

// harnessConfig configures a call to runEvalHarness.
type harnessConfig struct {
	Criteria    []eval.Criterion
	JudgeClient llm.Client
	JudgePrompt string
	JudgeSchema json.RawMessage
	Threshold   float64
	// RunCase runs the target prompt and returns judge placeholders.
	// It may call t.Errorf for non-fatal constraint violations.
	RunCase func(ctx context.Context, t *testing.T, c harnessCase) (map[string]string, error)
}

// runEvalHarness runs the common eval loop: invoke target → judge → aggregate → check.
func runEvalHarness(ctx context.Context, t *testing.T, cases []harnessCase, cfg harnessConfig) {
	t.Helper()

	var results []eval.CaseResult

	for _, c := range cases {
		t.Logf("evaluating [%s] %s ...", c.SetType, c.Name)

		placeholders, err := cfg.RunCase(ctx, t, c)
		if err != nil {
			if eval.IsInconclusive(err) {
				t.Skipf("API call failed (inconclusive) for case %s: %v", c.Name, err)
			}
			t.Fatalf("case %s failed: %v", c.Name, err)
		}

		scores, err := eval.Judge(ctx, cfg.JudgeClient, cfg.JudgePrompt, cfg.JudgeSchema, eval.JudgeInput{
			Placeholders: placeholders,
		})
		if err != nil {
			if eval.IsInconclusive(err) {
				t.Skipf("judge API call failed (inconclusive): %v", err)
			}
			t.Fatalf("judge failed for case %s: %v", c.Name, err)
		}

		results = append(results, eval.CaseResult{
			CaseName: c.Name,
			SetType:  c.SetType,
			Scores:   scores,
		})

		for _, s := range scores {
			t.Logf("  [%s] %s: %s=%d (%s)", c.SetType, c.Name, s.Criterion, s.Score, s.Reason)
		}
	}

	if len(results) == 0 {
		t.Skip("no results collected (all cases inconclusive)")
	}

	agg := eval.AggregateScores(results)

	t.Logf("=== Aggregated scores ===")
	t.Logf("overall average: %.2f (threshold: %.2f)", agg.Overall, cfg.Threshold)
	for _, c := range cfg.Criteria {
		t.Logf("  %s: %.2f", c, agg.ByCriterion[c])
	}

	for _, r := range results {
		if r.SetType != "regression" {
			continue
		}
		caseAgg := eval.AggregateScores([]eval.CaseResult{r})
		if caseAgg.Overall < cfg.Threshold {
			t.Errorf("*** REGRESSION CASE FAILED *** [%s] overall=%.2f < threshold=%.2f", r.CaseName, caseAgg.Overall, cfg.Threshold)
			for _, s := range r.Scores {
				slog.Warn("regression failure detail", "case", r.CaseName, "criterion", s.Criterion, "score", s.Score, "reason", s.Reason)
			}
		}
	}

	if agg.Overall < cfg.Threshold {
		t.Errorf("overall average %.2f is below threshold %.2f", agg.Overall, cfg.Threshold)
	}
}
