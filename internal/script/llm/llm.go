package llm

import (
	"context"
	"encoding/json"
)

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type CompletionRequest struct {
	Model       string          `json:"model"`
	Messages    []Message       `json:"messages"`
	JSONSchema  json.RawMessage `json:"json_schema,omitempty"`
	Temperature float64         `json:"temperature"`
}

type Client interface {
	Complete(ctx context.Context, req CompletionRequest) (json.RawMessage, error)
}
