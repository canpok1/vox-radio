package sel

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/canpok1/vox-radio/internal/config"
	"github.com/canpok1/vox-radio/internal/model"
	"github.com/canpok1/vox-radio/internal/rundown/prompt"
	"github.com/canpok1/vox-radio/internal/script/llm"
)

var selectSchema = json.RawMessage(`{
  "type": "object",
  "required": ["selected_ids", "selection_reason"],
  "properties": {
    "selected_ids": {"type": "array", "items": {"type": "string"}, "minItems": 1},
    "selection_reason": {"type": "string"}
  },
  "additionalProperties": false
}`)

// SelectResult holds the output of a selection operation.
type SelectResult struct {
	SelectedIDs     []string
	SelectionReason string
}

// Selector selects articles from candidates and designs the talk flow for a corner.
type Selector interface {
	Select(ctx context.Context, corner config.CornerConfig, articles []model.Article) (SelectResult, error)
}

// CornerAppearanceSetter is an optional supplementary interface for selectors that accept
// the current corner's appearance context (count including this episode, last episode number).
// rundown.Run sets these per corner before each Select so the value reflects the corner in hand.
type CornerAppearanceSetter interface {
	SetCornerAppearance(appearanceCount, lastEpisodeNumber int)
}

// articleForPrompt is the subset of article data passed to the LLM (body excluded to save tokens).
type articleForPrompt struct {
	ID    string `json:"id"`            // DedupKey: 選別結果の id として返却される
	URL   string `json:"url,omitempty"` // 表示用（空可）
	Title string `json:"title"`
}

type selectResponse struct {
	SelectedIDs     []string `json:"selected_ids"`
	SelectionReason string   `json:"selection_reason"`
}

// LLMSelector uses an LLM to select articles and design a talk flow.
type LLMSelector struct {
	client                  llm.Client
	promptTemplate          string
	temperature             float64
	casts                   []model.RundownCast
	cornerAppearanceCount   int
	cornerLastEpisodeNumber int
}

func NewLLMSelector(client llm.Client, promptTemplate string, temperature float64) *LLMSelector {
	return &LLMSelector{client: client, promptTemplate: promptTemplate, temperature: temperature}
}

// SetCasts configures the confirmed cast members to inject into the selection prompt.
func (s *LLMSelector) SetCasts(casts []model.RundownCast) {
	s.casts = casts
}

// SetCornerAppearance configures the current corner's appearance context injected into the
// selection prompt. appearanceCount includes this episode (1 = new corner); lastEpisodeNumber is
// the most recent past episode in which the corner appeared (0 = none).
func (s *LLMSelector) SetCornerAppearance(appearanceCount, lastEpisodeNumber int) {
	s.cornerAppearanceCount = appearanceCount
	s.cornerLastEpisodeNumber = lastEpisodeNumber
}

func (s *LLMSelector) Select(ctx context.Context, corner config.CornerConfig, articles []model.Article) (SelectResult, error) {
	cp := prompt.NewCornerForPrompt(corner)
	cp.AppearanceCount = s.cornerAppearanceCount
	cp.LastEpisodeNumber = s.cornerLastEpisodeNumber
	cornerJSON, err := json.Marshal(cp)
	if err != nil {
		return SelectResult{}, fmt.Errorf("marshal corner: %w", err)
	}

	aps := make([]articleForPrompt, len(articles))
	for i, a := range articles {
		aps[i] = articleForPrompt{ID: a.DedupKey, URL: a.URL, Title: a.Title}
	}
	articlesJSON, err := json.Marshal(aps)
	if err != nil {
		return SelectResult{}, fmt.Errorf("marshal articles: %w", err)
	}

	castsJSON, err := json.Marshal(model.CastsForLLM(s.casts))
	if err != nil {
		return SelectResult{}, fmt.Errorf("marshal casts: %w", err)
	}

	prompt := strings.NewReplacer(
		"{{corner}}", string(cornerJSON),
		"{{articles}}", string(articlesJSON),
		"{{casts}}", string(castsJSON),
	).Replace(s.promptTemplate)

	raw, err := s.client.Complete(ctx, llm.CompletionRequest{
		Messages:    []llm.Message{{Role: "user", Content: prompt}},
		JSONSchema:  selectSchema,
		Temperature: s.temperature,
	})
	if err != nil {
		return SelectResult{}, fmt.Errorf("llm complete: %w", err)
	}

	var resp selectResponse
	if err := json.Unmarshal(raw, &resp); err != nil {
		return SelectResult{}, fmt.Errorf("unmarshal response: %w", err)
	}

	ids := resp.SelectedIDs
	if ids == nil {
		ids = make([]string, 0)
	}
	return SelectResult{SelectedIDs: ids, SelectionReason: resp.SelectionReason}, nil
}
