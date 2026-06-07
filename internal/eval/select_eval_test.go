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

// selectCast mirrors the cast element passed to select.md.
type selectCast struct {
	CharacterID     string `json:"character_id"`
	Role            string `json:"role"`
	Type            string `json:"type"`
	AppearanceCount int    `json:"appearance_count"`
}

// selectCorner mirrors the CornerForPrompt passed to select.md.
type selectCorner struct {
	Title                 string `json:"title"`
	Content               string `json:"content"`
	TargetDurationSeconds int    `json:"target_duration_seconds"`
}

// selectArticle mirrors the articleForPrompt passed to select.md.
type selectArticle struct {
	URL   string `json:"url"`
	Title string `json:"title"`
}

// selectCase is one entry in the select testdata files.
type selectCase struct {
	Name        string          `json:"name"`
	Category    string          `json:"category"`
	Casts       []selectCast    `json:"casts"`
	Corner      selectCorner    `json:"corner"`
	Articles    []selectArticle `json:"articles"`
	Expectation string          `json:"expectation,omitempty"`
}

// selectOutputSchema is the JSON schema for the select.md output.
var selectOutputSchema = json.RawMessage(`{
  "type": "object",
  "required": ["selected_urls", "selection_reason"],
  "properties": {
    "selected_urls": {
      "type": "array",
      "items": {"type": "string"},
      "minItems": 1
    },
    "selection_reason": {"type": "string"}
  },
  "additionalProperties": false
}`)

// selectJudgeSchema is the JSON schema for the select judge LLM output.
var selectJudgeSchema = json.RawMessage(`{
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
            "enum": ["relevance", "constraint_compliance", "ordering_quality", "reason_validity"]
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

// selectOutput mirrors the select.md output for mechanical verification.
type selectOutput struct {
	SelectedURLs    []string `json:"selected_urls"`
	SelectionReason string   `json:"selection_reason"`
}

func runSelect(ctx context.Context, t *testing.T, client llm.Client, promptTemplate string, castsJSON, cornerJSON, articlesJSON string) (json.RawMessage, error) {
	t.Helper()
	prompt := strings.NewReplacer(
		"{{casts}}", castsJSON,
		"{{corner}}", cornerJSON,
		"{{articles}}", articlesJSON,
	).Replace(promptTemplate)
	return client.Complete(ctx, llm.CompletionRequest{
		Messages:   []llm.Message{{Role: "user", Content: prompt}},
		JSONSchema: selectOutputSchema,
	})
}

func TestSelectEval(t *testing.T) {
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

	threshold, err := getEnvFloat("VOX_EVAL_SELECT_THRESHOLD", 4.0)
	if err != nil {
		t.Fatalf("parse VOX_EVAL_SELECT_THRESHOLD: %v", err)
	}

	sampleSize, err := getEnvInt("VOX_EVAL_SAMPLE_SIZE", 8)
	if err != nil {
		t.Fatalf("parse VOX_EVAL_SAMPLE_SIZE: %v", err)
	}

	seed, err := getEnvInt64("VOX_EVAL_SAMPLE_SEED", eval.DefaultSeed())
	if err != nil {
		t.Fatalf("parse VOX_EVAL_SAMPLE_SEED: %v", err)
	}

	selectPrompt, err := eval.LoadPrompt("select")
	if err != nil {
		t.Fatalf("load select prompt: %v", err)
	}

	judgePrompt := loadTestdataString(t, "select_judge.md")

	// Load test sets.
	regressionCases := loadCasesJSON[selectCase](t, "select_regression_cases.json")
	poolCases := loadCasesJSON[selectCase](t, "select_pool_cases.json")

	sampled := eval.Sample(poolCases, sampleSize, seed)
	sampledNames := make([]string, len(sampled))
	for i, c := range sampled {
		sampledNames[i] = c.Name
	}
	t.Logf("seed=%d, sampled generalization cases: %v", seed, sampledNames)

	// Bundle all cases: regression (all) + generalization (sampled).
	type evaluationCase struct {
		selectCase
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

		// Marshal inputs for use in prompt and judge.
		castsJSONBytes, err := json.Marshal(ec.Casts)
		if err != nil {
			t.Fatalf("marshal casts for case %s: %v", ec.Name, err)
		}
		cornerJSONBytes, err := json.Marshal(ec.Corner)
		if err != nil {
			t.Fatalf("marshal corner for case %s: %v", ec.Name, err)
		}
		articlesJSONBytes, err := json.Marshal(ec.Articles)
		if err != nil {
			t.Fatalf("marshal articles for case %s: %v", ec.Name, err)
		}
		castsJSON := string(castsJSONBytes)
		cornerJSON := string(cornerJSONBytes)
		articlesJSON := string(articlesJSONBytes)

		// Step 1: run select.
		raw, err := runSelect(ctx, t, targetClient, selectPrompt, castsJSON, cornerJSON, articlesJSON)
		if err != nil {
			if eval.IsInconclusive(err) {
				t.Skipf("select API call failed (inconclusive) for case %s: %v", ec.Name, err)
			}
			t.Fatalf("select failed for case %s: %v", ec.Name, err)
		}

		// Step 2: mechanical verification of constraint_compliance.
		var output selectOutput
		if err := json.Unmarshal(raw, &output); err != nil {
			t.Fatalf("unmarshal select output for case %s: %v", ec.Name, err)
		}
		candidateURLs := make(map[string]bool, len(ec.Articles))
		for _, a := range ec.Articles {
			candidateURLs[a.URL] = true
		}
		if len(output.SelectedURLs) == 0 {
			t.Errorf("*** CONSTRAINT VIOLATION *** [%s] selected_urls is empty (min 1 required)", ec.Name)
		}
		for _, u := range output.SelectedURLs {
			if !candidateURLs[u] {
				t.Errorf("*** CONSTRAINT VIOLATION *** [%s] selected URL %q is not in candidate set", ec.Name, u)
			}
		}

		// Step 3: judge.
		scores, err := eval.Judge(ctx, judgeClient, judgePrompt, selectJudgeSchema, eval.JudgeInput{
			Placeholders: map[string]string{
				"casts":         castsJSON,
				"corner":        cornerJSON,
				"articles":      articlesJSON,
				"select_output": string(raw),
				"expectation":   eval.ResolveExpectation(ec.Expectation),
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
	for _, c := range eval.AllSelectCriteria {
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
