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

func TestJudge_ParsesScores(t *testing.T) {
	resp := json.RawMessage(`{"scores":[
		{"criterion":"detection_recall","score":5,"reason":"全て検出"},
		{"criterion":"false_positive_suppression","score":4,"reason":"誤検出なし"},
		{"criterion":"correction_accuracy","score":5,"reason":"正確"},
		{"criterion":"reason_validity","score":4,"reason":"妥当"}
	]}`)
	client := &mockLLMClient{response: resp}

	scores, err := Judge(context.Background(), client, "{{lines}} {{corrections}} {{expectation}}", JudgeInput{
		LinesJSON:       `[]`,
		CorrectionsJSON: `{"corrections":[]}`,
		Expectation:     "corrections は空であるべき",
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

func TestJudge_EmptyExpectationFilled(t *testing.T) {
	client := &mockLLMClient{
		response: json.RawMessage(`{"scores":[
			{"criterion":"detection_recall","score":5,"reason":"ok"},
			{"criterion":"false_positive_suppression","score":5,"reason":"ok"},
			{"criterion":"correction_accuracy","score":5,"reason":"ok"},
			{"criterion":"reason_validity","score":5,"reason":"ok"}
		]}`),
	}

	// Empty Expectation should use "（なし）" as fallback without error.
	scores, err := Judge(context.Background(), client, "{{expectation}}", JudgeInput{
		LinesJSON:       `[]`,
		CorrectionsJSON: `{}`,
		Expectation:     "",
	})
	if err != nil {
		t.Fatalf("Judge with empty expectation: %v", err)
	}
	if len(scores) == 0 {
		t.Error("expected scores from judge")
	}
}

func TestJudge_LLMError(t *testing.T) {
	client := &mockLLMClient{err: errors.New("http do: connection refused")}
	_, err := Judge(context.Background(), client, "prompt", JudgeInput{
		LinesJSON:       `[]`,
		CorrectionsJSON: `{}`,
	})
	if err == nil {
		t.Fatal("expected error from Judge when LLM fails")
	}
}
