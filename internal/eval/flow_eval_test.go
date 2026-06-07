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

// flowArticle mirrors the RundownArticle passed to flow.md.
type flowArticle struct {
	URL     string   `json:"url"`
	Title   string   `json:"title"`
	Summary string   `json:"summary"`
	Points  []string `json:"points"`
}

// flowCornerForProgram mirrors the cornerForProgram passed as part of {{program}}.
type flowCornerForProgram struct {
	Title           string        `json:"title"`
	SelectionReason string        `json:"selection_reason"`
	Articles        []flowArticle `json:"articles"`
}

// flowProgram mirrors the programForPrompt passed to flow.md.
type flowProgram struct {
	Corners []flowCornerForProgram `json:"corners"`
	Casts   []evalCast             `json:"casts"`
}

// flowCase is one entry in the flow testdata files.
type flowCase struct {
	Name            string        `json:"name"`
	Category        string        `json:"category"`
	Position        string        `json:"position"`
	Corner          evalCorner    `json:"corner"`
	Articles        []flowArticle `json:"articles"`
	SelectionReason string        `json:"selection_reason"`
	Program         flowProgram   `json:"program"`
	Expectation     string        `json:"expectation,omitempty"`
}

// flowOutputSchema is the JSON schema for the flow.md output.
var flowOutputSchema = json.RawMessage(`{
  "type": "object",
  "required": ["flow"],
  "properties": {
    "flow": {"type": "string"}
  },
  "additionalProperties": false
}`)

// flowJudgeSchema is the JSON schema for the flow judge LLM output.
var flowJudgeSchema = json.RawMessage(`{
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
            "enum": ["position_role_fit", "consistency", "article_alignment", "actionability"]
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

// flowOutput mirrors the flow.md output.
type flowOutput struct {
	Flow string `json:"flow"`
}

func runFlow(ctx context.Context, t *testing.T, client llm.Client, promptTemplate string, cornerJSON, articlesJSON, programJSON, position, selectionReason string) (json.RawMessage, error) {
	t.Helper()
	prompt := strings.NewReplacer(
		"{{corner}}", cornerJSON,
		"{{position}}", position,
		"{{articles}}", articlesJSON,
		"{{selection_reason}}", selectionReason,
		"{{program}}", programJSON,
	).Replace(promptTemplate)
	return client.Complete(ctx, llm.CompletionRequest{
		Messages:   []llm.Message{{Role: "user", Content: prompt}},
		JSONSchema: flowOutputSchema,
	})
}

func TestFlowEval(t *testing.T) {
	requireGeminiKey(t)

	targetClient, judgeClient := buildEvalClients(t)

	threshold, err := getEnvFloat("VOX_EVAL_FLOW_THRESHOLD", 4.0)
	if err != nil {
		t.Fatalf("parse VOX_EVAL_FLOW_THRESHOLD: %v", err)
	}

	sampleSize, seed := loadSampleParams(t)

	flowPrompt, err := eval.LoadPrompt("flow")
	if err != nil {
		t.Fatalf("load flow prompt: %v", err)
	}

	judgePrompt := loadTestdataString(t, "flow_judge.md")

	regressionCases := loadCasesJSON[flowCase](t, "flow_regression_cases.json")
	poolCases := loadCasesJSON[flowCase](t, "flow_pool_cases.json")

	allCases, caseByName := buildHarnessCases(t, regressionCases, poolCases, sampleSize, seed, func(c flowCase) string { return c.Name })

	ctx := context.Background()

	runEvalHarness(ctx, t, allCases, harnessConfig{
		Criteria:    eval.AllFlowCriteria,
		JudgeClient: judgeClient,
		JudgePrompt: judgePrompt,
		JudgeSchema: flowJudgeSchema,
		Threshold:   threshold,
		RunCase: func(ctx context.Context, t *testing.T, c harnessCase) (map[string]string, error) {
			ec := caseByName[c.Name]

			cornerJSONBytes, err := json.Marshal(ec.Corner)
			if err != nil {
				return nil, fmt.Errorf("marshal corner for case %s: %w", c.Name, err)
			}
			articlesJSONBytes, err := json.Marshal(ec.Articles)
			if err != nil {
				return nil, fmt.Errorf("marshal articles for case %s: %w", c.Name, err)
			}
			programJSONBytes, err := json.Marshal(ec.Program)
			if err != nil {
				return nil, fmt.Errorf("marshal program for case %s: %w", c.Name, err)
			}
			cornerJSON := string(cornerJSONBytes)
			articlesJSON := string(articlesJSONBytes)
			programJSON := string(programJSONBytes)

			raw, err := runFlow(ctx, t, targetClient, flowPrompt, cornerJSON, articlesJSON, programJSON, ec.Position, ec.SelectionReason)
			if err != nil {
				return nil, err
			}

			// Mechanical verification: flow must be non-empty.
			var output flowOutput
			if err := json.Unmarshal(raw, &output); err != nil {
				return nil, fmt.Errorf("unmarshal flow output for case %s: %w", c.Name, err)
			}
			if strings.TrimSpace(output.Flow) == "" {
				t.Errorf("*** CONSTRAINT VIOLATION *** [%s] flow is empty", c.Name)
			}

			return map[string]string{
				"corner":           cornerJSON,
				"position":         ec.Position,
				"articles":         articlesJSON,
				"selection_reason": ec.SelectionReason,
				"program":          programJSON,
				"flow_output":      output.Flow,
				"expectation":      eval.ResolveExpectation(ec.Expectation),
			}, nil
		},
	})
}
