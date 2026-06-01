package llm_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/canpok1/vox-radio/internal/script/llm"
)

// makeDifyResponse builds a minimal Dify chat-messages blocking response JSON.
func makeDifyResponse(t *testing.T, answer, conversationID string) []byte {
	t.Helper()
	type resp struct {
		Event          string `json:"event"`
		Answer         string `json:"answer"`
		ConversationId string `json:"conversation_id,omitempty"`
		MessageId      string `json:"message_id"`
		TaskId         string `json:"task_id"`
		CreatedAt      int64  `json:"created_at"`
	}
	r := resp{
		Event:          "message",
		Answer:         answer,
		ConversationId: conversationID,
		MessageId:      "msg-001",
		TaskId:         "task-001",
		CreatedAt:      1700000000,
	}
	b, err := json.Marshal(r)
	if err != nil {
		t.Fatalf("marshal dify response: %v", err)
	}
	return b
}

func newDifyTestConfig(ts *httptest.Server, intervalMS int) llm.Config {
	return llm.Config{
		Provider: "dify-chat",
		DifyChat: &llm.DifyChatClientConfig{
			BaseURL: ts.URL,
			APIKey:  "test-key",
			User:    "test-user",
			Inputs:  nil,
		},
		MaxRetries:           3,
		MinRequestIntervalMS: intervalMS,
	}
}

func TestDifyChatComplete_Success(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(makeDifyResponse(t, `{"name":"hello"}`, "conv-001"))
	}))
	defer ts.Close()

	c := llm.NewClient(newDifyTestConfig(ts, 0))
	result, err := c.Complete(context.Background(), llm.CompletionRequest{
		Messages: []llm.Message{
			{Role: "system", Content: "You are helpful."},
			{Role: "user", Content: "Say hello"},
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(result) != `{"name":"hello"}` {
		t.Errorf("result = %s, want %s", string(result), `{"name":"hello"}`)
	}
}

func TestDifyChatComplete_WithSchema_Success(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(makeDifyResponse(t, `{"name":"hello"}`, "conv-001"))
	}))
	defer ts.Close()

	c := llm.NewClient(newDifyTestConfig(ts, 0))
	result, err := c.Complete(context.Background(), llm.CompletionRequest{
		Messages:   []llm.Message{{Role: "user", Content: "hello"}},
		JSONSchema: testRequiredNameSchema,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	var got map[string]string
	if err := json.Unmarshal(result, &got); err != nil {
		t.Fatalf("unmarshal result: %v", err)
	}
	if got["name"] != "hello" {
		t.Errorf("name = %q, want %q", got["name"], "hello")
	}
}

func TestDifyChatComplete_SchemaRetry_ConversationIDCarried(t *testing.T) {
	var callCount atomic.Int32
	var receivedConvIDs []string

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		n := callCount.Add(1)

		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err == nil {
			if convID, ok := body["conversation_id"].(string); ok {
				receivedConvIDs = append(receivedConvIDs, convID)
			} else {
				receivedConvIDs = append(receivedConvIDs, "")
			}
		}

		if n == 1 {
			_, _ = w.Write(makeDifyResponse(t, `{"wrong":"value"}`, "conv-abc"))
		} else {
			_, _ = w.Write(makeDifyResponse(t, `{"name":"hello"}`, "conv-abc"))
		}
	}))
	defer ts.Close()

	cfg := newDifyTestConfig(ts, 0)
	cfg.MaxRetries = 3
	c := llm.NewClient(cfg)

	result, err := c.Complete(context.Background(), llm.CompletionRequest{
		Messages:   []llm.Message{{Role: "user", Content: "hello"}},
		JSONSchema: testRequiredNameSchema,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	var got map[string]string
	if err := json.Unmarshal(result, &got); err != nil {
		t.Fatalf("unmarshal result: %v", err)
	}
	if got["name"] != "hello" {
		t.Errorf("name = %q, want %q", got["name"], "hello")
	}
	if callCount.Load() != 2 {
		t.Errorf("expected 2 calls, got %d", callCount.Load())
	}
	// Second call should carry the conversation_id from the first response.
	if len(receivedConvIDs) >= 2 && receivedConvIDs[1] != "conv-abc" {
		t.Errorf("second call conversation_id = %q, want %q", receivedConvIDs[1], "conv-abc")
	}
}

func TestDifyChatComplete_ExhaustsRetries(t *testing.T) {
	var callCount atomic.Int32
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		callCount.Add(1)
		_, _ = w.Write(makeDifyResponse(t, `{"wrong":"value"}`, "conv-001"))
	}))
	defer ts.Close()

	maxRetries := 2
	cfg := newDifyTestConfig(ts, 0)
	cfg.MaxRetries = maxRetries
	c := llm.NewClient(cfg)

	_, err := c.Complete(context.Background(), llm.CompletionRequest{
		Messages:   []llm.Message{{Role: "user", Content: "hello"}},
		JSONSchema: testRequiredNameSchema,
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if int(callCount.Load()) != maxRetries+1 {
		t.Errorf("expected %d calls, got %d", maxRetries+1, callCount.Load())
	}
}

func TestDifyChatComplete_APIError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"code":"internal_server_error","message":"Internal server error","status":500}`))
	}))
	defer ts.Close()

	cfg := newDifyTestConfig(ts, 0)
	cfg.MaxRetries = 0
	c := llm.NewClient(cfg)

	_, err := c.Complete(context.Background(), llm.CompletionRequest{
		Messages: []llm.Message{{Role: "user", Content: "hello"}},
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestDifyChatComplete_Inputs_TemperatureExact(t *testing.T) {
	var receivedBody map[string]any

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewDecoder(r.Body).Decode(&receivedBody)
		_, _ = w.Write(makeDifyResponse(t, `{}`, "conv-001"))
	}))
	defer ts.Close()

	cfg := llm.Config{
		Provider: "dify-chat",
		DifyChat: &llm.DifyChatClientConfig{
			BaseURL: ts.URL,
			APIKey:  "test-key",
			User:    "test-user",
			Inputs:  map[string]string{"temperature": "${temperature}", "lang": "ja"},
		},
		MaxRetries:           0,
		MinRequestIntervalMS: 0,
	}
	c := llm.NewClient(cfg)

	_, err := c.Complete(context.Background(), llm.CompletionRequest{
		Messages:    []llm.Message{{Role: "user", Content: "hello"}},
		Temperature: 0.7,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	inputs, ok := receivedBody["inputs"].(map[string]any)
	if !ok {
		t.Fatalf("inputs not found in request body")
	}
	// Exact match: should be float64
	tempVal, ok := inputs["temperature"]
	if !ok {
		t.Fatal("temperature not found in inputs")
	}
	if _, ok := tempVal.(float64); !ok {
		t.Errorf("temperature should be float64 (JSON number), got %T", tempVal)
	}
	if tempVal.(float64) != 0.7 {
		t.Errorf("temperature = %v, want 0.7", tempVal)
	}
	// lang should be string
	if inputs["lang"] != "ja" {
		t.Errorf("lang = %v, want %q", inputs["lang"], "ja")
	}
}

func TestDifyChatComplete_Inputs_TemperaturePartial(t *testing.T) {
	var receivedBody map[string]any

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewDecoder(r.Body).Decode(&receivedBody)
		_, _ = w.Write(makeDifyResponse(t, `{}`, "conv-001"))
	}))
	defer ts.Close()

	cfg := llm.Config{
		Provider: "dify-chat",
		DifyChat: &llm.DifyChatClientConfig{
			BaseURL: ts.URL,
			APIKey:  "test-key",
			User:    "test-user",
			Inputs:  map[string]string{"param": "temp=${temperature}"},
		},
		MaxRetries:           0,
		MinRequestIntervalMS: 0,
	}
	c := llm.NewClient(cfg)

	_, err := c.Complete(context.Background(), llm.CompletionRequest{
		Messages:    []llm.Message{{Role: "user", Content: "hello"}},
		Temperature: 0.5,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	inputs, ok := receivedBody["inputs"].(map[string]any)
	if !ok {
		t.Fatalf("inputs not found in request body")
	}
	// Partial match: should be string with interpolation
	paramVal, ok := inputs["param"]
	if !ok {
		t.Fatal("param not found in inputs")
	}
	if _, ok := paramVal.(string); !ok {
		t.Errorf("param should be string, got %T", paramVal)
	}
	if paramVal.(string) != "temp=0.5" {
		t.Errorf("param = %q, want %q", paramVal.(string), "temp=0.5")
	}
}

func TestDifyChatComplete_Inputs_NoTemperaturePlaceholder(t *testing.T) {
	var receivedBody map[string]any

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewDecoder(r.Body).Decode(&receivedBody)
		_, _ = w.Write(makeDifyResponse(t, `{}`, "conv-001"))
	}))
	defer ts.Close()

	cfg := llm.Config{
		Provider: "dify-chat",
		DifyChat: &llm.DifyChatClientConfig{
			BaseURL: ts.URL,
			APIKey:  "test-key",
			User:    "test-user",
			Inputs:  map[string]string{"lang": "ja"},
		},
		MaxRetries:           0,
		MinRequestIntervalMS: 0,
	}
	c := llm.NewClient(cfg)

	_, err := c.Complete(context.Background(), llm.CompletionRequest{
		Messages:    []llm.Message{{Role: "user", Content: "hello"}},
		Temperature: 0.7,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	inputs, ok := receivedBody["inputs"].(map[string]any)
	if !ok {
		t.Fatalf("inputs not found in request body")
	}
	// No temperature placeholder: temperature should not be in inputs
	if _, ok := inputs["temperature"]; ok {
		t.Error("temperature should not be in inputs when no placeholder")
	}
	if inputs["lang"] != "ja" {
		t.Errorf("lang = %v, want %q", inputs["lang"], "ja")
	}
}

func TestDifyChatComplete_QueryBuildsFromMessages(t *testing.T) {
	var receivedQuery string

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err == nil {
			if q, ok := body["query"].(string); ok {
				receivedQuery = q
			}
		}
		_, _ = w.Write(makeDifyResponse(t, `{}`, "conv-001"))
	}))
	defer ts.Close()

	c := llm.NewClient(newDifyTestConfig(ts, 0))
	_, err := c.Complete(context.Background(), llm.CompletionRequest{
		Messages: []llm.Message{
			{Role: "system", Content: "You are helpful."},
			{Role: "user", Content: "Say hello"},
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	want := "You are helpful.\n\nSay hello"
	if receivedQuery != want {
		t.Errorf("query = %q, want %q", receivedQuery, want)
	}
}

func TestDifyChatComplete_Throttle(t *testing.T) {
	var callCount atomic.Int32
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount.Add(1)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(makeDifyResponse(t, `{}`, "conv-001"))
	}))
	defer ts.Close()

	cfg := newDifyTestConfig(ts, 50)
	c := llm.NewClient(cfg)

	req := llm.CompletionRequest{Messages: []llm.Message{{Role: "user", Content: "hello"}}}

	_, err := c.Complete(context.Background(), req)
	if err != nil {
		t.Fatalf("first request failed: %v", err)
	}
	_, err = c.Complete(context.Background(), req)
	if err != nil {
		t.Fatalf("second request failed: %v", err)
	}

	if callCount.Load() != 2 {
		t.Errorf("expected 2 requests, got %d", callCount.Load())
	}
}
