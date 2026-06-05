package flow

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/canpok1/vox-radio/internal/config"
	"github.com/canpok1/vox-radio/internal/model"
	"github.com/canpok1/vox-radio/internal/script/llm"
)

var flowSchema = json.RawMessage(`{
  "type": "object",
  "required": ["flow"],
  "properties": {
    "flow": {"type": "string"}
  },
  "additionalProperties": false
}`)

// Position はコーナーの番組内位置（構造的ロールヒント）。
type Position string

const (
	PositionOpening Position = "opening" // 先頭=導入
	PositionEnding  Position = "ending"  // 末尾=締め
	PositionMiddle  Position = "middle"  // 中間=つなぎ
)

// Designer は1コーナーの flow を番組構成全体の文脈から設計する。
type Designer interface {
	DesignFlow(ctx context.Context, corner config.CornerConfig, position Position, target model.RundownCorner, rundown model.Rundown) (string, error)
}

// cornerForPrompt はプロンプトに渡すコーナー情報（body 除外）。
type cornerForPrompt struct {
	Title                 string `json:"title"`
	Content               string `json:"content"`
	TargetDurationSeconds int    `json:"target_duration_seconds"`
}

// articleForProgram はプロンプトに渡す記事情報（body 除外）。
type articleForProgram struct {
	URL     string   `json:"url"`
	Title   string   `json:"title"`
	Summary string   `json:"summary"`
	Points  []string `json:"points"`
}

// cornerForProgram はプロンプトに渡す番組全体コーナー情報。
type cornerForProgram struct {
	Title           string              `json:"title"`
	SelectionReason string              `json:"selection_reason"`
	Articles        []articleForProgram `json:"articles"`
}

// programForPrompt はプロンプトに渡す番組全体情報。
type programForPrompt struct {
	Corners []cornerForProgram  `json:"corners"`
	Casts   []model.RundownCast `json:"casts"`
}

type flowResponse struct {
	Flow string `json:"flow"`
}

// LLMDesigner は LLM を使ってコーナーの flow を設計する。
type LLMDesigner struct {
	client         llm.Client
	promptTemplate string
	temperature    float64
}

func NewLLMDesigner(client llm.Client, promptTemplate string, temperature float64) *LLMDesigner {
	return &LLMDesigner{client: client, promptTemplate: promptTemplate, temperature: temperature}
}

func (d *LLMDesigner) DesignFlow(ctx context.Context, corner config.CornerConfig, position Position, target model.RundownCorner, rundown model.Rundown) (string, error) {
	cp := cornerForPrompt{
		Title:                 corner.Title,
		Content:               corner.Content,
		TargetDurationSeconds: corner.LengthSec,
	}
	cornerJSON, err := json.Marshal(cp)
	if err != nil {
		return "", fmt.Errorf("marshal corner: %w", err)
	}

	articles := make([]articleForProgram, len(target.Articles))
	for i, a := range target.Articles {
		articles[i] = articleForProgram{
			URL:     a.URL,
			Title:   a.Title,
			Summary: a.Summary,
			Points:  a.Points,
		}
	}
	articlesJSON, err := json.Marshal(articles)
	if err != nil {
		return "", fmt.Errorf("marshal articles: %w", err)
	}

	programCorners := make([]cornerForProgram, len(rundown.Corners))
	for i, c := range rundown.Corners {
		pas := make([]articleForProgram, len(c.Articles))
		for j, a := range c.Articles {
			pas[j] = articleForProgram{
				URL:     a.URL,
				Title:   a.Title,
				Summary: a.Summary,
				Points:  a.Points,
			}
		}
		programCorners[i] = cornerForProgram{
			Title:           c.Title,
			SelectionReason: c.SelectionReason,
			Articles:        pas,
		}
	}
	casts := rundown.Casts
	if casts == nil {
		casts = make([]model.RundownCast, 0)
	}
	prog := programForPrompt{
		Corners: programCorners,
		Casts:   casts,
	}
	programJSON, err := json.Marshal(prog)
	if err != nil {
		return "", fmt.Errorf("marshal program: %w", err)
	}

	prompt := strings.NewReplacer(
		"{{corner}}", string(cornerJSON),
		"{{position}}", string(position),
		"{{articles}}", string(articlesJSON),
		"{{selection_reason}}", target.SelectionReason,
		"{{program}}", string(programJSON),
	).Replace(d.promptTemplate)

	raw, err := d.client.Complete(ctx, llm.CompletionRequest{
		Messages:    []llm.Message{{Role: "user", Content: prompt}},
		JSONSchema:  flowSchema,
		Temperature: d.temperature,
	})
	if err != nil {
		return "", fmt.Errorf("llm complete: %w", err)
	}

	var resp flowResponse
	if err := json.Unmarshal(raw, &resp); err != nil {
		return "", fmt.Errorf("unmarshal response: %w", err)
	}

	return resp.Flow, nil
}
