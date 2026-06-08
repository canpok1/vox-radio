package flow

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

// PositionFor は index（0始まり）と last（最後のインデックス）から Position を返す。
func PositionFor(index, last int) Position {
	switch index {
	case 0:
		return PositionOpening
	case last:
		return PositionEnding
	default:
		return PositionMiddle
	}
}

// Designer は1コーナーの flow を番組構成全体の文脈から設計する。
type Designer interface {
	DesignFlow(ctx context.Context, corner config.CornerConfig, position Position, target model.RundownCorner, rundown model.Rundown) (string, error)
}

// cornerForProgram はプロンプトに渡す番組全体コーナー情報。
// AppearanceCount は今回を含む扱い回数（1 = 新コーナー）。LastEpisodeNumber は前回扱った回番号（0 = 過去になし）。
type cornerForProgram struct {
	Title             string                 `json:"title"`
	SelectionReason   string                 `json:"selection_reason"`
	Articles          []model.RundownArticle `json:"articles"`
	AppearanceCount   int                    `json:"appearance_count,omitempty"`
	LastEpisodeNumber int                    `json:"last_episode_number,omitempty"`
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
	cp := prompt.NewCornerForPrompt(corner)
	cp.AppearanceCount = target.AppearanceCount
	cp.LastEpisodeNumber = target.LastEpisodeNumber
	cornerJSON, err := json.Marshal(cp)
	if err != nil {
		return "", fmt.Errorf("marshal corner: %w", err)
	}

	articlesJSON, err := json.Marshal(target.Articles)
	if err != nil {
		return "", fmt.Errorf("marshal articles: %w", err)
	}

	programCorners := make([]cornerForProgram, len(rundown.Corners))
	for i, c := range rundown.Corners {
		programCorners[i] = cornerForProgram{
			Title:             c.Title,
			SelectionReason:   c.SelectionReason,
			Articles:          c.Articles,
			AppearanceCount:   c.AppearanceCount,
			LastEpisodeNumber: c.LastEpisodeNumber,
		}
	}
	prog := programForPrompt{
		Corners: programCorners,
		Casts:   model.CastsForLLM(rundown.Casts),
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
