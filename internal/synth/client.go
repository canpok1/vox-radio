package synth

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
)

// VoicevoxClient is the interface for the VOICEVOX HTTP API
type VoicevoxClient interface {
	AudioQuery(ctx context.Context, text string, speaker int) (*AudioQuery, error)
	Synthesis(ctx context.Context, query *AudioQuery, speaker int) ([]byte, error)
}

type httpVoicevoxClient struct {
	baseURL    string
	httpClient *http.Client
}

// NewClient creates a new VOICEVOX HTTP API client
func NewClient(baseURL string) VoicevoxClient {
	return &httpVoicevoxClient{
		baseURL:    baseURL,
		httpClient: &http.Client{},
	}
}

func (c *httpVoicevoxClient) AudioQuery(ctx context.Context, text string, speaker int) (*AudioQuery, error) {
	u, err := url.Parse(c.baseURL + "/audio_query")
	if err != nil {
		return nil, fmt.Errorf("parse URL: %w", err)
	}
	q := u.Query()
	q.Set("text", text)
	q.Set("speaker", fmt.Sprintf("%d", speaker))
	u.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, u.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("do request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("audio_query returned %d", resp.StatusCode)
	}

	var query AudioQuery
	if err := json.NewDecoder(resp.Body).Decode(&query); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}
	return &query, nil
}

func (c *httpVoicevoxClient) Synthesis(ctx context.Context, query *AudioQuery, speaker int) ([]byte, error) {
	body, err := json.Marshal(query)
	if err != nil {
		return nil, fmt.Errorf("marshal query: %w", err)
	}

	u, err := url.Parse(c.baseURL + "/synthesis")
	if err != nil {
		return nil, fmt.Errorf("parse URL: %w", err)
	}
	q := u.Query()
	q.Set("speaker", fmt.Sprintf("%d", speaker))
	u.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, u.String(), bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("do request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("synthesis returned %d", resp.StatusCode)
	}

	wavBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}
	return wavBytes, nil
}
