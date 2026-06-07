//go:build eval

package eval_test

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
	"testing"

	"github.com/canpok1/vox-radio/internal/eval"
	"github.com/canpok1/vox-radio/internal/script/llm"
)

// summaryScriptLine mirrors one element of the {{script_lines}} input for summary.md.
type summaryScriptLine struct {
	Speaker string `json:"speaker"`
	Text    string `json:"text"`
}

// summaryCase is one entry in the summary testdata files.
type summaryCase struct {
	Name          string              `json:"name"`
	Category      string              `json:"category"`
	ScriptLines   []summaryScriptLine `json:"script_lines"`
	SummaryLength int                 `json:"summary_length"`
	Expectation   string              `json:"expectation,omitempty"`
}

// summaryOutputSchema is the JSON schema for the summary.md output.
var summaryOutputSchema = json.RawMessage(`{
  "type": "object",
  "required": ["summary", "episode_title", "conversation_notes"],
  "properties": {
    "summary": {"type": "string"},
    "episode_title": {"type": "string"},
    "conversation_notes": {
      "type": "array",
      "items": {
        "type": "object",
        "required": ["category", "character_ids", "note"],
        "properties": {
          "category": {"type": "string"},
          "character_ids": {"type": "array", "items": {"type": "string"}},
          "note": {"type": "string"}
        },
        "additionalProperties": false
      }
    }
  },
  "additionalProperties": false
}`)

// summaryJudgeSchema is the JSON schema for the summary judge LLM output.
var summaryJudgeSchema = json.RawMessage(`{
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
            "enum": ["summary_quality", "episode_title_quality", "notes_faithfulness", "notes_coverage"]
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

func runSummary(ctx context.Context, t *testing.T, client llm.Client, promptTemplate string, ec summaryCase, linesJSON string) (json.RawMessage, error) {
	t.Helper()
	prompt := strings.NewReplacer(
		"{{script_lines}}", linesJSON,
		"{{summary_length}}", strconv.Itoa(ec.SummaryLength),
	).Replace(promptTemplate)
	return client.Complete(ctx, llm.CompletionRequest{
		Messages:   []llm.Message{{Role: "user", Content: prompt}},
		JSONSchema: summaryOutputSchema,
	})
}

func TestSummaryEval(t *testing.T) {
	if os.Getenv("GEMINI_API_KEY") == "" {
		t.Skip("GEMINI_API_KEY not set")
	}

	targetClient, judgeClient := buildEvalClients(t)

	threshold, err := getEnvFloat("VOX_EVAL_SUMMARY_THRESHOLD", 4.0)
	if err != nil {
		t.Fatalf("parse VOX_EVAL_SUMMARY_THRESHOLD: %v", err)
	}

	sampleSize, err := getEnvInt("VOX_EVAL_SAMPLE_SIZE", 8)
	if err != nil {
		t.Fatalf("parse VOX_EVAL_SAMPLE_SIZE: %v", err)
	}

	seed, err := getEnvInt64("VOX_EVAL_SAMPLE_SEED", eval.DefaultSeed())
	if err != nil {
		t.Fatalf("parse VOX_EVAL_SAMPLE_SEED: %v", err)
	}

	summaryPrompt, err := eval.LoadPrompt("summary")
	if err != nil {
		t.Fatalf("load summary prompt: %v", err)
	}

	judgePrompt := loadTestdataString(t, "summary_judge.md")

	regressionCases := loadCasesJSON[summaryCase](t, "summary_regression_cases.json")
	poolCases := loadCasesJSON[summaryCase](t, "summary_pool_cases.json")

	sampled := eval.Sample(poolCases, sampleSize, seed)
	sampledNames := make([]string, len(sampled))
	for i, c := range sampled {
		sampledNames[i] = c.Name
	}
	t.Logf("seed=%d, sampled generalization cases: %v", seed, sampledNames)

	caseByName := make(map[string]summaryCase, len(regressionCases)+len(sampled))
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
		Criteria:    eval.AllSummaryCriteria,
		JudgeClient: judgeClient,
		JudgePrompt: judgePrompt,
		JudgeSchema: summaryJudgeSchema,
		Threshold:   threshold,
		RunCase: func(ctx context.Context, t *testing.T, c harnessCase) (map[string]string, error) {
			ec := caseByName[c.Name]
			linesJSONBytes, err := json.Marshal(ec.ScriptLines)
			if err != nil {
				return nil, fmt.Errorf("marshal script_lines for case %s: %w", c.Name, err)
			}
			linesJSON := string(linesJSONBytes)

			raw, err := runSummary(ctx, t, targetClient, summaryPrompt, ec, linesJSON)
			if err != nil {
				return nil, err
			}

			return map[string]string{
				"script_lines":   linesJSON,
				"summary_output": string(raw),
				"expectation":    eval.ResolveExpectation(ec.Expectation),
			}, nil
		},
	})
}
