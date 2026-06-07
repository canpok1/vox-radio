package eval

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/canpok1/vox-radio/internal/script/llm"
)

// mockLLMClient is a minimal llm.Client stub for unit tests.
type mockLLMClient struct {
	response json.RawMessage
	err      error
}

func (m *mockLLMClient) Complete(_ context.Context, _ llm.CompletionRequest) (json.RawMessage, error) {
	return m.response, m.err
}

var testSchema = json.RawMessage(`{
  "type": "object",
  "required": ["scores"],
  "properties": {
    "scores": {
      "type": "array",
      "items": {
        "type": "object",
        "required": ["criterion", "score", "reason"],
        "properties": {
          "criterion": {"type": "string"},
          "score": {"type": "integer", "minimum": 1, "maximum": 5},
          "reason": {"type": "string"}
        }
      }
    }
  }
}`)

func TestJudge_ParsesScores(t *testing.T) {
	resp := json.RawMessage(`{"scores":[
		{"criterion":"detection_recall","score":5,"reason":"全て検出"},
		{"criterion":"false_positive_suppression","score":4,"reason":"誤検出なし"},
		{"criterion":"correction_accuracy","score":5,"reason":"正確"},
		{"criterion":"reason_validity","score":4,"reason":"妥当"}
	]}`)
	client := &mockLLMClient{response: resp}

	scores, err := Judge(context.Background(), client, "{{lines}} {{corrections}} {{expectation}}", testSchema, JudgeInput{
		Placeholders: map[string]string{
			"lines":       `[]`,
			"corrections": `{"corrections":[]}`,
			"expectation": "corrections は空であるべき",
		},
	})
	if err != nil {
		t.Fatalf("Judge: %v", err)
	}
	if len(scores) != 4 {
		t.Errorf("len(scores) = %d, want 4", len(scores))
	}
	if scores[0].Criterion != CriterionDetectionRecall {
		t.Errorf("scores[0].Criterion = %q, want %q", scores[0].Criterion, CriterionDetectionRecall)
	}
	if scores[0].Score != 5 {
		t.Errorf("scores[0].Score = %d, want 5", scores[0].Score)
	}
}

func TestJudge_PlaceholderReplacement(t *testing.T) {
	var capturedPrompt string
	client := &capturingClient{
		response: json.RawMessage(`{"scores":[]}`),
		onComplete: func(req llm.CompletionRequest) {
			capturedPrompt = req.Messages[0].Content
		},
	}

	_, err := Judge(context.Background(), client, "input={{article}} expectation={{expectation}}", testSchema, JudgeInput{
		Placeholders: map[string]string{
			"article":     `{"title":"test"}`,
			"expectation": "良い要約であるべき",
		},
	})
	if err != nil {
		t.Fatalf("Judge: %v", err)
	}
	if want := `input={"title":"test"} expectation=良い要約であるべき`; capturedPrompt != want {
		t.Errorf("prompt = %q, want %q", capturedPrompt, want)
	}
}

func TestJudge_LLMError(t *testing.T) {
	client := &mockLLMClient{err: errors.New("http do: connection refused")}
	_, err := Judge(context.Background(), client, "prompt", testSchema, JudgeInput{
		Placeholders: map[string]string{},
	})
	if err == nil {
		t.Fatal("expected error from Judge when LLM fails")
	}
}

// capturingClient captures the CompletionRequest for inspection in tests.
type capturingClient struct {
	response   json.RawMessage
	err        error
	onComplete func(llm.CompletionRequest)
}

func (c *capturingClient) Complete(_ context.Context, req llm.CompletionRequest) (json.RawMessage, error) {
	if c.onComplete != nil {
		c.onComplete(req)
	}
	return c.response, c.err
}
