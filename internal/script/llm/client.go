package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/xeipuuv/gojsonschema"
)

const (
	DefaultBaseURL = "https://generativelanguage.googleapis.com/v1beta/openai"
	DefaultModel   = "gemini-3.1-flash-lite"
)

// DifyChatClientConfig holds connection settings for the Dify chat-messages provider.
type DifyChatClientConfig struct {
	BaseURL string
	APIKey  string
	User    string
	Inputs  map[string]string
}

// Config holds the settings for the LLM client factory.
type Config struct {
	Provider             string
	BaseURL              string
	APIKey               string
	Model                string
	MaxRetries           int
	Temperature          float64
	MinRequestIntervalMS int
	DifyChat             *DifyChatClientConfig
	// SharedThrottler, if non-nil, is used instead of creating a new throttler from MinRequestIntervalMS.
	// Use NewThrottler and share the pointer across multiple clients to enforce a combined rate limit.
	SharedThrottler *Throttler
}

// Throttler manages rate limiting for LLM API calls and can be shared across multiple clients.
type Throttler struct {
	minIntervalMS int
	mu            sync.Mutex
	lastTime      time.Time
}

// NewThrottler creates a Throttler with the given minimum interval between requests.
// Pass the returned pointer to Config.SharedThrottler to share it across multiple clients.
func NewThrottler(minIntervalMS int) *Throttler {
	return &Throttler{minIntervalMS: minIntervalMS}
}

func (t *Throttler) throttle(ctx context.Context) error {
	if t.minIntervalMS <= 0 {
		return nil
	}
	interval := time.Duration(t.minIntervalMS) * time.Millisecond

	t.mu.Lock()
	now := time.Now()
	next := t.lastTime.Add(interval)
	if next.Before(now) {
		next = now
	}
	t.lastTime = next
	t.mu.Unlock()

	waitDur := time.Until(next)
	if waitDur <= 0 {
		return nil
	}
	select {
	case <-time.After(waitDur):
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

type openAIClient struct {
	cfg      Config
	hc       *http.Client
	endpoint string
	t        *Throttler
}

// NewClient creates a new LLM client, selecting the implementation based on Config.Provider.
// Empty Provider defaults to the OpenAI-compatible implementation.
func NewClient(cfg Config) Client {
	switch cfg.Provider {
	case "dify-chat":
		return newDifyChatClient(cfg)
	default:
		return newOpenAIClient(cfg)
	}
}

func newOpenAIClient(cfg Config) Client {
	if cfg.BaseURL == "" {
		cfg.BaseURL = DefaultBaseURL
	}
	if cfg.Model == "" {
		cfg.Model = DefaultModel
	}
	t := cfg.SharedThrottler
	if t == nil {
		t = &Throttler{minIntervalMS: cfg.MinRequestIntervalMS}
	}
	return &openAIClient{
		cfg:      cfg,
		hc:       &http.Client{Timeout: 60 * time.Second},
		endpoint: strings.TrimRight(cfg.BaseURL, "/") + "/chat/completions",
		t:        t,
	}
}

type apiRequest struct {
	Model          string          `json:"model"`
	Messages       []Message       `json:"messages"`
	ResponseFormat *responseFormat `json:"response_format,omitempty"`
	Temperature    float64         `json:"temperature,omitempty"`
}

type responseFormat struct {
	Type       string      `json:"type"`
	JSONSchema *schemaSpec `json:"json_schema,omitempty"`
}

type schemaSpec struct {
	Name   string          `json:"name"`
	Schema json.RawMessage `json:"schema"`
	Strict bool            `json:"strict"`
}

type apiResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
		Type    string `json:"type"`
	} `json:"error,omitempty"`
}

func (c *openAIClient) Complete(ctx context.Context, req CompletionRequest) (json.RawMessage, error) {
	msgs := make([]Message, len(req.Messages))
	copy(msgs, req.Messages)

	if len(req.JSONSchema) == 0 {
		return c.callAPI(ctx, req, msgs)
	}

	maxRetries := c.cfg.MaxRetries
	if maxRetries < 0 {
		maxRetries = 0
	}

	for attempt := 0; attempt <= maxRetries; attempt++ {
		raw, err := c.callAPI(ctx, req, msgs)
		if err != nil {
			return nil, err
		}

		errs, err := validateAgainstSchema(raw, req.JSONSchema)
		if err != nil {
			return nil, fmt.Errorf("validate schema: %w", err)
		}
		if len(errs) == 0 {
			return raw, nil
		}

		if attempt < maxRetries {
			msgs = append(msgs,
				Message{Role: "assistant", Content: string(raw)},
				Message{Role: "user", Content: buildRepairPrompt(errs)},
			)
		}
	}

	return nil, fmt.Errorf("validation failed after %d retries", c.cfg.MaxRetries)
}

func (c *openAIClient) callAPI(ctx context.Context, req CompletionRequest, msgs []Message) (json.RawMessage, error) {
	if err := c.t.throttle(ctx); err != nil {
		return nil, err
	}

	model := req.Model
	if model == "" {
		model = c.cfg.Model
	}

	temperature := req.Temperature
	if temperature == 0 {
		temperature = c.cfg.Temperature
	}

	apiReq := apiRequest{
		Model:       model,
		Messages:    msgs,
		Temperature: temperature,
	}

	if len(req.JSONSchema) > 0 {
		apiReq.ResponseFormat = &responseFormat{
			Type: "json_schema",
			JSONSchema: &schemaSpec{
				Name:   "output",
				Schema: req.JSONSchema,
				Strict: true,
			},
		}
	} else {
		apiReq.ResponseFormat = &responseFormat{Type: "json_object"}
	}

	body, err := json.Marshal(apiReq)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+c.cfg.APIKey)

	resp, err := c.hc.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("http do: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("api error (status %d): %s", resp.StatusCode, string(respBody))
	}

	var apiResp apiResponse
	if err := json.Unmarshal(respBody, &apiResp); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}

	if apiResp.Error != nil {
		return nil, fmt.Errorf("api error: %s", apiResp.Error.Message)
	}

	if len(apiResp.Choices) == 0 {
		return nil, fmt.Errorf("empty choices in response")
	}

	return json.RawMessage(apiResp.Choices[0].Message.Content), nil
}

func validateAgainstSchema(data json.RawMessage, schema json.RawMessage) ([]string, error) {
	if !json.Valid(data) {
		return []string{"response is not valid JSON"}, nil
	}

	schemaLoader := gojsonschema.NewBytesLoader(schema)
	dataLoader := gojsonschema.NewBytesLoader(data)

	result, err := gojsonschema.Validate(schemaLoader, dataLoader)
	if err != nil {
		return nil, err
	}

	var errs []string
	for _, e := range result.Errors() {
		errs = append(errs, e.String())
	}
	return errs, nil
}

func buildRepairPrompt(errs []string) string {
	return "Your previous response did not conform to the required JSON schema. Validation errors:\n" +
		strings.Join(errs, "\n") +
		"\nPlease provide a corrected JSON response that strictly follows the schema."
}
