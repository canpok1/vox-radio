package eval

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/canpok1/vox-radio/internal/script/llm"
)

// judgeResponse is the parsed judge LLM output.
type judgeResponse struct {
	Scores []ScoreEntry `json:"scores"`
}

// JudgeInput holds template placeholders for a single judge call.
// Each key maps to a value that replaces {{key}} in the judge prompt.
type JudgeInput struct {
	Placeholders map[string]string
}

// Judge calls the LLM with the judge prompt and returns scored criteria.
// schema is the JSON schema for the expected judge output.
// judgePrompt may contain {{key}} placeholders replaced by input.Placeholders.
func Judge(ctx context.Context, client llm.Client, judgePrompt string, schema json.RawMessage, input JudgeInput) ([]ScoreEntry, error) {
	pairs := make([]string, 0, len(input.Placeholders)*2)
	for k, v := range input.Placeholders {
		pairs = append(pairs, "{{"+k+"}}", v)
	}
	prompt := strings.NewReplacer(pairs...).Replace(judgePrompt)

	raw, err := client.Complete(ctx, llm.CompletionRequest{
		Messages:   []llm.Message{{Role: "user", Content: prompt}},
		JSONSchema: schema,
	})
	if err != nil {
		return nil, fmt.Errorf("judge llm: %w", err)
	}

	var resp judgeResponse
	if err := json.Unmarshal(raw, &resp); err != nil {
		return nil, fmt.Errorf("unmarshal judge response: %w", err)
	}
	return resp.Scores, nil
}
