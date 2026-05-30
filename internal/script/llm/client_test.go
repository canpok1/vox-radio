package llm_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/canpok1/vox-radio/internal/script/llm"
)

func makeAPIResponse(t *testing.T, content string) []byte {
	t.Helper()
	type msgWrapper struct {
		Content string `json:"content"`
	}
	type choice struct {
		Message msgWrapper `json:"message"`
	}
	type resp struct {
		Choices []choice `json:"choices"`
	}
	r := resp{Choices: []choice{{Message: msgWrapper{Content: content}}}}
	b, err := json.Marshal(r)
	if err != nil {
		t.Fatalf("marshal api response: %v", err)
	}
	return b
}

func TestComplete_Success(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(makeAPIResponse(t, `{"value":"hello"}`))
	}))
	defer ts.Close()

	c := llm.NewClient(llm.Config{
		BaseURL:    ts.URL,
		APIKey:     "test-key",
		Model:      "test-model",
		MaxRetries: 0,
	})

	result, err := c.Complete(context.Background(), llm.CompletionRequest{
		Messages: []llm.Message{{Role: "user", Content: "hello"}},
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !json.Valid(result) {
		t.Fatalf("result is not valid JSON: %s", string(result))
	}
}

func TestComplete_SetsAuthorizationHeader(t *testing.T) {
	var gotAuth string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(makeAPIResponse(t, `{}`))
	}))
	defer ts.Close()

	c := llm.NewClient(llm.Config{
		BaseURL: ts.URL,
		APIKey:  "my-secret-key",
		Model:   "test-model",
	})

	_, _ = c.Complete(context.Background(), llm.CompletionRequest{
		Messages: []llm.Message{{Role: "user", Content: "hello"}},
	})

	if gotAuth != "Bearer my-secret-key" {
		t.Errorf("expected Authorization: Bearer my-secret-key, got %q", gotAuth)
	}
}

func TestComplete_WithSchema_Success(t *testing.T) {
	schema := json.RawMessage(`{
		"type": "object",
		"properties": {
			"name": {"type": "string"}
		},
		"required": ["name"]
	}`)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(makeAPIResponse(t, `{"name":"hello"}`))
	}))
	defer ts.Close()

	c := llm.NewClient(llm.Config{
		BaseURL:    ts.URL,
		APIKey:     "test-key",
		Model:      "test-model",
		MaxRetries: 0,
	})

	result, err := c.Complete(context.Background(), llm.CompletionRequest{
		Messages:   []llm.Message{{Role: "user", Content: "hello"}},
		JSONSchema: schema,
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var got map[string]string
	if err := json.Unmarshal(result, &got); err != nil {
		t.Fatalf("unmarshal result: %v", err)
	}
	if got["name"] != "hello" {
		t.Errorf("expected name=hello, got %q", got["name"])
	}
}

func TestComplete_ValidationRetrySuccess(t *testing.T) {
	schema := json.RawMessage(`{
		"type": "object",
		"properties": {
			"name": {"type": "string"}
		},
		"required": ["name"]
	}`)

	var callCount atomic.Int32
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		n := callCount.Add(1)
		if n == 1 {
			_, _ = w.Write(makeAPIResponse(t, `{"wrong":"value"}`))
		} else {
			_, _ = w.Write(makeAPIResponse(t, `{"name":"hello"}`))
		}
	}))
	defer ts.Close()

	c := llm.NewClient(llm.Config{
		BaseURL:    ts.URL,
		APIKey:     "test-key",
		Model:      "test-model",
		MaxRetries: 3,
	})

	result, err := c.Complete(context.Background(), llm.CompletionRequest{
		Messages:   []llm.Message{{Role: "user", Content: "hello"}},
		JSONSchema: schema,
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var got map[string]string
	if err := json.Unmarshal(result, &got); err != nil {
		t.Fatalf("unmarshal result: %v", err)
	}
	if got["name"] != "hello" {
		t.Errorf("expected name=hello, got %q", got["name"])
	}
	if callCount.Load() != 2 {
		t.Errorf("expected 2 API calls, got %d", callCount.Load())
	}
}

func TestComplete_RetryIncludesRepairContext(t *testing.T) {
	schema := json.RawMessage(`{
		"type": "object",
		"properties": {
			"name": {"type": "string"}
		},
		"required": ["name"]
	}`)

	var requestBodies [][]byte
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		requestBodies = append(requestBodies, body)
		w.Header().Set("Content-Type", "application/json")
		if len(requestBodies) == 1 {
			_, _ = w.Write(makeAPIResponse(t, `{"wrong":"value"}`))
		} else {
			_, _ = w.Write(makeAPIResponse(t, `{"name":"hello"}`))
		}
	}))
	defer ts.Close()

	c := llm.NewClient(llm.Config{
		BaseURL:    ts.URL,
		APIKey:     "test-key",
		Model:      "test-model",
		MaxRetries: 2,
	})

	_, err := c.Complete(context.Background(), llm.CompletionRequest{
		Messages:   []llm.Message{{Role: "user", Content: "hello"}},
		JSONSchema: schema,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(requestBodies) != 2 {
		t.Fatalf("expected 2 requests, got %d", len(requestBodies))
	}

	type apiReq struct {
		Messages []struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		} `json:"messages"`
	}

	var secondReq apiReq
	if err := json.Unmarshal(requestBodies[1], &secondReq); err != nil {
		t.Fatalf("unmarshal second request: %v", err)
	}

	// original message + assistant (invalid response) + user (repair prompt) = 3 messages
	if len(secondReq.Messages) != 3 {
		t.Errorf("expected 3 messages in retry request, got %d", len(secondReq.Messages))
	}
	if secondReq.Messages[1].Role != "assistant" {
		t.Errorf("expected messages[1].role=assistant, got %q", secondReq.Messages[1].Role)
	}
	if secondReq.Messages[2].Role != "user" {
		t.Errorf("expected messages[2].role=user, got %q", secondReq.Messages[2].Role)
	}
}

func TestComplete_ExhaustsRetries(t *testing.T) {
	schema := json.RawMessage(`{
		"type": "object",
		"properties": {
			"name": {"type": "string"}
		},
		"required": ["name"]
	}`)

	var callCount atomic.Int32
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		callCount.Add(1)
		_, _ = w.Write(makeAPIResponse(t, `{"wrong":"value"}`))
	}))
	defer ts.Close()

	maxRetries := 2
	c := llm.NewClient(llm.Config{
		BaseURL:    ts.URL,
		APIKey:     "test-key",
		Model:      "test-model",
		MaxRetries: maxRetries,
	})

	_, err := c.Complete(context.Background(), llm.CompletionRequest{
		Messages:   []llm.Message{{Role: "user", Content: "hello"}},
		JSONSchema: schema,
	})

	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if int(callCount.Load()) != maxRetries+1 {
		t.Errorf("expected %d API calls, got %d", maxRetries+1, callCount.Load())
	}
}

func TestComplete_APIError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"error":{"message":"internal error","type":"server_error"}}`))
	}))
	defer ts.Close()

	c := llm.NewClient(llm.Config{
		BaseURL:    ts.URL,
		APIKey:     "test-key",
		Model:      "test-model",
		MaxRetries: 3,
	})

	_, err := c.Complete(context.Background(), llm.CompletionRequest{
		Messages: []llm.Message{{Role: "user", Content: "hello"}},
	})

	if err == nil {
		t.Fatal("expected error, got nil")
	}
}
