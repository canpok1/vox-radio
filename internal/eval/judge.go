package eval

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/canpok1/vox-radio/internal/script/llm"
)

// judgeSchema is the JSON schema for the judge LLM output.
var judgeSchema = json.RawMessage(`{
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
            "enum": ["detection_recall", "false_positive_suppression", "correction_accuracy", "reason_validity"]
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

// judgeResponse is the parsed judge LLM output.
type judgeResponse struct {
	Scores []ScoreEntry `json:"scores"`
}

// JudgeInput holds the inputs for a single judge call.
type JudgeInput struct {
	LinesJSON       string // JSON representation of the input lines
	CorrectionsJSON string // JSON representation of proofread output
	Expectation     string // optional expected result (empty if generalization set)
}

// Judge calls the LLM with the judge prompt and returns scored criteria.
// judgePrompt may contain {{lines}}, {{corrections}}, and {{expectation}} placeholders.
func Judge(ctx context.Context, client llm.Client, judgePrompt string, input JudgeInput, model string) ([]ScoreEntry, error) {
	expectation := input.Expectation
	if expectation == "" {
		expectation = "（なし）"
	}

	prompt := strings.NewReplacer(
		"{{lines}}", input.LinesJSON,
		"{{corrections}}", input.CorrectionsJSON,
		"{{expectation}}", expectation,
	).Replace(judgePrompt)

	req := llm.CompletionRequest{
		Messages:   []llm.Message{{Role: "user", Content: prompt}},
		JSONSchema: judgeSchema,
	}
	if model != "" {
		req.Model = model
	}

	raw, err := client.Complete(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("judge llm: %w", err)
	}

	var resp judgeResponse
	if err := json.Unmarshal(raw, &resp); err != nil {
		return nil, fmt.Errorf("unmarshal judge response: %w", err)
	}
	return resp.Scores, nil
}
