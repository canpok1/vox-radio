//go:build eval

package eval_test

import (
	"context"
	"encoding/json"
	"fmt"
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

	targetClient, judgeClient := buildEvalClients(t)

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

	regressionCases := loadCasesJSON[summarizeCase](t, "summarize_regression_cases.json")
	poolCases := loadCasesJSON[summarizeCase](t, "summarize_pool_cases.json")

	sampled := eval.Sample(poolCases, sampleSize, seed)
	sampledNames := make([]string, len(sampled))
	for i, c := range sampled {
		sampledNames[i] = c.Name
	}
	t.Logf("seed=%d, sampled generalization cases: %v", seed, sampledNames)

	caseByName := make(map[string]summarizeCase, len(regressionCases)+len(sampled))
	for _, c := range regressionCases {
		caseByName[c.Name] = c
	}
	for _, c := range sampled {
		caseByName[c.Name] = c
	}

	var allCases []harnessCase
	for _, c := range regressionCases {
		allCases = append(allCases, harnessCase{c.Name, "regression"})
	}
	for _, c := range sampled {
		allCases = append(allCases, harnessCase{c.Name, "generalization"})
	}

	ctx := context.Background()

	runEvalHarness(ctx, t, allCases, harnessConfig{
		Criteria:    eval.AllSummarizeCriteria,
		JudgeClient: judgeClient,
		JudgePrompt: judgePrompt,
		JudgeSchema: summarizeJudgeSchema,
		Threshold:   threshold,
		RunCase: func(ctx context.Context, t *testing.T, c harnessCase) (map[string]string, error) {
			ec := caseByName[c.Name]
			articleJSONBytes, err := json.Marshal(ec.Article)
			if err != nil {
				return nil, fmt.Errorf("marshal article for case %s: %w", c.Name, err)
			}
			articleJSON := string(articleJSONBytes)

			raw, err := runSummarize(ctx, t, targetClient, summarizePrompt, articleJSON)
			if err != nil {
				return nil, err
			}

			return map[string]string{
				"article":        articleJSON,
				"summary_output": string(raw),
				"expectation":    eval.ResolveExpectation(ec.Expectation),
			}, nil
		},
	})
}
