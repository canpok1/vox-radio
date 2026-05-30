package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/xeipuuv/gojsonschema"
)

const (
	DefaultBaseURL = "https://generativelanguage.googleapis.com/v1beta/openai"
	DefaultModel   = "gemini-3.1-flash-lite"
)

// Config holds the settings for the OpenAI-compatible LLM client.
type Config struct {
	BaseURL     string
	APIKey      string
	Model       string
	MaxRetries  int
	Temperature float64
}

type openAIClient struct {
	cfg      Config
	hc       *http.Client
	endpoint string
}

// NewClient creates a new OpenAI-compatible LLM client.
func NewClient(cfg Config) Client {
	if cfg.BaseURL == "" {
		cfg.BaseURL = DefaultBaseURL
	}
	if cfg.Model == "" {
		cfg.Model = DefaultModel
	}
	return &openAIClient{
		cfg:      cfg,
		hc:       &http.Client{Timeout: 60 * time.Second},
		endpoint: strings.TrimRight(cfg.BaseURL, "/") + "/chat/completions",
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

	apiReq.ResponseFormat = &responseFormat{Type: "json_object"}
	if len(req.JSONSchema) > 0 {
		apiReq.ResponseFormat.Type = "json_schema"
		apiReq.ResponseFormat.JSONSchema = &schemaSpec{
			Name:   "output",
			Schema: req.JSONSchema,
			Strict: true,
		}
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
