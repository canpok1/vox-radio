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

// evalCorner mirrors the CornerForPrompt shared across prompt eval tests.
type evalCorner struct {
	Title                 string `json:"title"`
	Content               string `json:"content"`
	TargetDurationSeconds int    `json:"target_duration_seconds"`
}

// evalCast mirrors the RundownCast (LLM-facing) shared across prompt eval tests.
type evalCast struct {
	CharacterID     string `json:"character_id"`
	Role            string `json:"role"`
	Type            string `json:"type"`
	AppearanceCount int    `json:"appearance_count"`
}

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

// requireGeminiKey skips the test if GEMINI_API_KEY is not set.
func requireGeminiKey(t *testing.T) {
	t.Helper()
	if os.Getenv("GEMINI_API_KEY") == "" {
		t.Skip("GEMINI_API_KEY not set")
	}
}

// loadSampleParams reads VOX_EVAL_SAMPLE_SIZE and VOX_EVAL_SAMPLE_SEED from env.
func loadSampleParams(t *testing.T) (sampleSize int, seed int64) {
	t.Helper()
	var err error
	sampleSize, err = getEnvInt("VOX_EVAL_SAMPLE_SIZE", 8)
	if err != nil {
		t.Fatalf("parse VOX_EVAL_SAMPLE_SIZE: %v", err)
	}
	seed, err = getEnvInt64("VOX_EVAL_SAMPLE_SEED", eval.DefaultSeed())
	if err != nil {
		t.Fatalf("parse VOX_EVAL_SAMPLE_SEED: %v", err)
	}
	return
}

// buildEvalClients creates target and judge LLM clients from env vars.
// Both clients share a single Throttler so their combined request rate stays within API RPM limits.
func buildEvalClients(t *testing.T) (llm.Client, llm.Client) {
	t.Helper()

	targetCfg := eval.BuildLLMConfig("GEMINI_API_KEY", "VOX_EVAL_MODEL", "VOX_EVAL_MIN_INTERVAL_MS")
	targetCfg.MaxRetries = 2

	judgeCfg := targetCfg
	if jm := os.Getenv("VOX_EVAL_JUDGE_MODEL"); jm != "" {
		judgeCfg.Model = jm
	}

	shared := llm.NewThrottler(targetCfg.MinRequestIntervalMS)
	targetCfg.SharedThrottler = shared
	judgeCfg.SharedThrottler = shared

	return llm.NewClient(targetCfg), llm.NewClient(judgeCfg)
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

// buildHarnessCases samples from pool, logs sampled names, and returns the combined
// harness case list and a lookup map keyed by name.
// Regression cases come first, followed by sampled generalization cases.
func buildHarnessCases[T any](t *testing.T, regression, pool []T, sampleSize int, seed int64, getName func(T) string) ([]harnessCase, map[string]T) {
	t.Helper()

	sampled := eval.Sample(pool, sampleSize, seed)
	sampledNames := make([]string, len(sampled))
	for i, c := range sampled {
		sampledNames[i] = getName(c)
	}
	t.Logf("seed=%d, sampled generalization cases: %v", seed, sampledNames)

	caseByName := make(map[string]T, len(regression)+len(sampled))
	var allCases []harnessCase
	for _, c := range regression {
		name := getName(c)
		caseByName[name] = c
		allCases = append(allCases, harnessCase{name, "regression"})
	}
	for _, c := range sampled {
		name := getName(c)
		caseByName[name] = c
		allCases = append(allCases, harnessCase{name, "generalization"})
	}

	return allCases, caseByName
}

// TestBuildHarnessCases verifies that buildHarnessCases correctly labels regression
// and generalization cases and populates the lookup map.
func TestBuildHarnessCases(t *testing.T) {
	type item struct {
		Name string
		Val  int
	}
	getName := func(c item) string { return c.Name }

	regression := []item{{Name: "r1", Val: 1}, {Name: "r2", Val: 2}}
	pool := []item{{Name: "p1", Val: 10}, {Name: "p2", Val: 20}, {Name: "p3", Val: 30}}

	allCases, caseByName := buildHarnessCases(t, regression, pool, 10, 42, getName)

	// Regression cases come first, labeled "regression".
	if len(allCases) < 2 {
		t.Fatalf("allCases len = %d, want >= 2", len(allCases))
	}
	for i, c := range allCases[:2] {
		if c.SetType != "regression" {
			t.Errorf("allCases[%d].SetType = %q, want regression", i, c.SetType)
		}
	}
	if allCases[0].Name != "r1" {
		t.Errorf("allCases[0].Name = %q, want r1", allCases[0].Name)
	}
	if allCases[1].Name != "r2" {
		t.Errorf("allCases[1].Name = %q, want r2", allCases[1].Name)
	}

	// Generalization cases follow, labeled "generalization".
	for i := 2; i < len(allCases); i++ {
		if allCases[i].SetType != "generalization" {
			t.Errorf("allCases[%d].SetType = %q, want generalization", i, allCases[i].SetType)
		}
	}

	// caseByName contains all regression cases.
	for _, name := range []string{"r1", "r2"} {
		if _, ok := caseByName[name]; !ok {
			t.Errorf("caseByName missing regression case %q", name)
		}
	}
	if caseByName["r1"].Val != 1 {
		t.Errorf("caseByName[r1].Val = %d, want 1", caseByName["r1"].Val)
	}

	// caseByName contains sampled pool cases (sampleSize > len(pool), so all pool items appear).
	for _, name := range []string{"p1", "p2", "p3"} {
		if _, ok := caseByName[name]; !ok {
			t.Errorf("caseByName missing pool case %q", name)
		}
	}
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
