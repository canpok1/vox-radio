package llm

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	dify "github.com/safejob/dify-sdk-go"
	"github.com/safejob/dify-sdk-go/base"
	"github.com/safejob/dify-sdk-go/types"
)

const difyTemperaturePlaceholder = "${temperature}"

type difyChatClient struct {
	cfg        Config
	dcCfg      DifyChatClientConfig
	baseClient *base.Client
	throttler
}

func newDifyChatClient(cfg Config) Client {
	dc := cfg.DifyChat
	if dc == nil {
		dc = &DifyChatClientConfig{}
	}
	user := dc.User
	if user == "" {
		user = "vox-radio"
	}

	bc, _ := dify.NewClient(dify.ClientConfig{
		ApiServer: dc.BaseURL,
		ApiKey:    dc.APIKey,
		User:      user,
		Timeout:   60 * time.Second,
	})

	return &difyChatClient{
		cfg:        cfg,
		dcCfg:      DifyChatClientConfig{BaseURL: dc.BaseURL, APIKey: dc.APIKey, User: user, Inputs: dc.Inputs},
		baseClient: bc,
		throttler:  throttler{minIntervalMS: cfg.MinRequestIntervalMS},
	}
}

func (c *difyChatClient) Complete(ctx context.Context, req CompletionRequest) (json.RawMessage, error) {
	temperature := req.Temperature
	if temperature == 0 {
		temperature = c.cfg.Temperature
	}

	query := buildDifyQuery(req.Messages)
	inputs := buildDifyInputs(c.dcCfg.Inputs, temperature)

	if len(req.JSONSchema) == 0 {
		raw, _, err := c.callDify(ctx, query, "", inputs)
		return raw, err
	}

	maxRetries := c.cfg.MaxRetries
	if maxRetries < 0 {
		maxRetries = 0
	}

	conversationID := ""
	currentQuery := query
	for attempt := 0; attempt <= maxRetries; attempt++ {
		raw, convID, err := c.callDify(ctx, currentQuery, conversationID, inputs)
		if err != nil {
			return nil, err
		}
		if conversationID == "" {
			conversationID = convID
		}

		errs, err := validateAgainstSchema(raw, req.JSONSchema)
		if err != nil {
			return nil, fmt.Errorf("validate schema: %w", err)
		}
		if len(errs) == 0 {
			return raw, nil
		}

		if attempt < maxRetries {
			currentQuery = buildRepairPrompt(errs)
		}
	}

	return nil, fmt.Errorf("validation failed after %d retries", c.cfg.MaxRetries)
}

func (c *difyChatClient) callDify(ctx context.Context, query, conversationID string, inputs map[string]any) (json.RawMessage, string, error) {
	if err := c.throttle(ctx); err != nil {
		return nil, "", err
	}

	if c.baseClient == nil {
		return nil, "", fmt.Errorf("dify base client not initialized")
	}

	app := c.baseClient.ChatbotApp()
	resp, err := app.RunBlock(ctx, types.ChatRequest{
		Query:          query,
		Inputs:         inputs,
		ConversationId: conversationID,
	})
	if err != nil {
		return nil, "", fmt.Errorf("dify run block: %w", err)
	}

	raw := json.RawMessage(resp.Answer)
	return raw, resp.ConversationId, nil
}

// buildDifyQuery joins all message contents into a single query string.
func buildDifyQuery(msgs []Message) string {
	parts := make([]string, 0, len(msgs))
	for _, m := range msgs {
		if m.Content != "" {
			parts = append(parts, m.Content)
		}
	}
	return strings.Join(parts, "\n\n")
}

// buildDifyInputs resolves ${temperature} placeholders in input values and returns map[string]any.
// Exact match → float64 (JSON number); partial match → string interpolation; no match → original string.
func buildDifyInputs(inputs map[string]string, temperature float64) map[string]any {
	result := make(map[string]any, len(inputs))
	tempStr := strconv.FormatFloat(temperature, 'f', -1, 64)
	for k, v := range inputs {
		if v == difyTemperaturePlaceholder {
			result[k] = temperature
		} else if strings.Contains(v, difyTemperaturePlaceholder) {
			result[k] = strings.ReplaceAll(v, difyTemperaturePlaceholder, tempStr)
		} else {
			result[k] = v
		}
	}
	return result
}
