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
	requireGeminiKey(t)

	targetClient, judgeClient := buildEvalClients(t)

	threshold, err := getEnvFloat("VOX_EVAL_SELECT_THRESHOLD", 4.0)
	if err != nil {
		t.Fatalf("parse VOX_EVAL_SELECT_THRESHOLD: %v", err)
	}

	sampleSize, seed := loadSampleParams(t)

	selectPrompt, err := eval.LoadPrompt("select")
	if err != nil {
		t.Fatalf("load select prompt: %v", err)
	}

	judgePrompt := loadTestdataString(t, "select_judge.md")

	regressionCases := loadCasesJSON[selectCase](t, "select_regression_cases.json")
	poolCases := loadCasesJSON[selectCase](t, "select_pool_cases.json")

	sampled := eval.Sample(poolCases, sampleSize, seed)
	sampledNames := make([]string, len(sampled))
	for i, c := range sampled {
		sampledNames[i] = c.Name
	}
	t.Logf("seed=%d, sampled generalization cases: %v", seed, sampledNames)

	caseByName := make(map[string]selectCase, len(regressionCases)+len(sampled))
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
		Criteria:    eval.AllSelectCriteria,
		JudgeClient: judgeClient,
		JudgePrompt: judgePrompt,
		JudgeSchema: selectJudgeSchema,
		Threshold:   threshold,
		RunCase: func(ctx context.Context, t *testing.T, c harnessCase) (map[string]string, error) {
			ec := caseByName[c.Name]

			castsJSONBytes, err := json.Marshal(ec.Casts)
			if err != nil {
				return nil, fmt.Errorf("marshal casts for case %s: %w", c.Name, err)
			}
			cornerJSONBytes, err := json.Marshal(ec.Corner)
			if err != nil {
				return nil, fmt.Errorf("marshal corner for case %s: %w", c.Name, err)
			}
			articlesJSONBytes, err := json.Marshal(ec.Articles)
			if err != nil {
				return nil, fmt.Errorf("marshal articles for case %s: %w", c.Name, err)
			}
			castsJSON := string(castsJSONBytes)
			cornerJSON := string(cornerJSONBytes)
			articlesJSON := string(articlesJSONBytes)

			raw, err := runSelect(ctx, t, targetClient, selectPrompt, castsJSON, cornerJSON, articlesJSON)
			if err != nil {
				return nil, err
			}

			// Mechanical verification of constraint_compliance.
			var output selectOutput
			if err := json.Unmarshal(raw, &output); err != nil {
				return nil, fmt.Errorf("unmarshal select output for case %s: %w", c.Name, err)
			}
			candidateURLs := make(map[string]bool, len(ec.Articles))
			for _, a := range ec.Articles {
				candidateURLs[a.URL] = true
			}
			if len(output.SelectedURLs) == 0 {
				t.Errorf("*** CONSTRAINT VIOLATION *** [%s] selected_urls is empty (min 1 required)", c.Name)
			}
			for _, u := range output.SelectedURLs {
				if !candidateURLs[u] {
					t.Errorf("*** CONSTRAINT VIOLATION *** [%s] selected URL %q is not in candidate set", c.Name, u)
				}
			}

			return map[string]string{
				"casts":         castsJSON,
				"corner":        cornerJSON,
				"articles":      articlesJSON,
				"select_output": string(raw),
				"expectation":   eval.ResolveExpectation(ec.Expectation),
			}, nil
		},
	})
}
