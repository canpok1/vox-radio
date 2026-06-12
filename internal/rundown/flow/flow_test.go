package flow_test

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/canpok1/vox-radio/internal/config"
	"github.com/canpok1/vox-radio/internal/model"
	"github.com/canpok1/vox-radio/internal/rundown/flow"
	"github.com/canpok1/vox-radio/internal/script/llm"
)

type mockClient struct {
	response json.RawMessage
	err      error
	captured []llm.CompletionRequest
}

func (m *mockClient) Complete(_ context.Context, req llm.CompletionRequest) (json.RawMessage, error) {
	m.captured = append(m.captured, req)
	return m.response, m.err
}

func TestPositionFor(t *testing.T) {
	tests := []struct {
		index   int
		last    int
		wantPos flow.Position
	}{
		{0, 2, flow.PositionOpening},
		{1, 2, flow.PositionMiddle},
		{2, 2, flow.PositionEnding},
		{0, 0, flow.PositionOpening}, // single-corner: index==0 matches first → opening
	}
	for _, tc := range tests {
		got := flow.PositionFor(tc.index, tc.last)
		if got != tc.wantPos {
			t.Errorf("PositionFor(%d, %d) = %q, want %q", tc.index, tc.last, got, tc.wantPos)
		}
	}
}

func TestLLMDesigner_DesignFlow_Success(t *testing.T) {
	mc := &mockClient{
		response: json.RawMessage(`{"flow":"ニュースを2本紹介して締める"}`),
	}
	d := flow.NewLLMDesigner(mc, "コーナー: {{corner}} 位置: {{position}} 記事: {{articles}} 選別理由: {{selection_reason}} 番組: {{program}}", 0)

	corner := config.CornerConfig{Title: "テックニュース", Content: "最新技術を紹介", LengthSec: 60}
	target := model.RundownCorner{
		Title:           "テックニュース",
		SelectionReason: "AIチップ記事が最も関連性高い",
		Articles: []model.RundownArticle{
			{URL: "https://example.com/1", Title: "記事1", Body: "記事1の本文", Points: []string{"p1"}},
		},
	}
	rd := model.Rundown{
		Corners: []model.RundownCorner{target},
		Casts:   []model.RundownCast{{CharacterID: "zundamon", Role: "MC", Type: "regular"}},
	}

	got, err := d.DesignFlow(context.Background(), corner, flow.PositionOpening, target, rd)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "ニュースを2本紹介して締める" {
		t.Errorf("flow: got %q, want %q", got, "ニュースを2本紹介して締める")
	}
}

func TestLLMDesigner_DesignFlow_PromptContainsCornerAppearance(t *testing.T) {
	mc := &mockClient{response: json.RawMessage(`{"flow":"フロー"}`)}
	d := flow.NewLLMDesigner(mc, "コーナー: {{corner}} 番組: {{program}}", 0)

	corner := config.CornerConfig{Title: "テック", Content: "内容", LengthSec: 60}
	target := model.RundownCorner{Title: "テック", AppearanceCount: 3, LastEpisodeNumber: 2}
	rd := model.Rundown{
		Corners: []model.RundownCorner{
			target,
			{Title: "別コーナー", AppearanceCount: 1}, // 番組内の別コーナー（新コーナー）
		},
		Casts: make([]model.RundownCast, 0),
	}

	_, err := d.DesignFlow(context.Background(), corner, flow.PositionOpening, target, rd)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	prompt := mc.captured[0].Messages[0].Content
	// 対象コーナー（{{corner}}）に扱い回数・前回出演回番号が出る
	if !strings.Contains(prompt, `"appearance_count":3`) {
		t.Errorf("corner prompt should contain appearance_count:3, got: %s", prompt)
	}
	if !strings.Contains(prompt, `"last_episode_number":2`) {
		t.Errorf("corner prompt should contain last_episode_number:2, got: %s", prompt)
	}
	// 番組構成全体（{{program}}）にも各コーナーの扱い回数が出る
	if !strings.Contains(prompt, `"appearance_count":1`) {
		t.Errorf("program prompt should contain other corner appearance_count:1, got: %s", prompt)
	}
}

func TestLLMDesigner_DesignFlow_PromptContainsCorner(t *testing.T) {
	mc := &mockClient{
		response: json.RawMessage(`{"flow":"フロー"}`),
	}
	d := flow.NewLLMDesigner(mc, "コーナー: {{corner}} 位置: {{position}} 記事: {{articles}} 選別理由: {{selection_reason}} 番組: {{program}}", 0)

	corner := config.CornerConfig{Title: "テック", Content: "内容", LengthSec: 60}
	target := model.RundownCorner{Title: "テック", SelectionReason: "理由"}
	rd := model.Rundown{Corners: []model.RundownCorner{target}, Casts: make([]model.RundownCast, 0)}

	_, _ = d.DesignFlow(context.Background(), corner, flow.PositionOpening, target, rd)

	if len(mc.captured) == 0 {
		t.Fatal("LLM was not called")
	}
	prompt := mc.captured[0].Messages[0].Content
	if !strings.Contains(prompt, "テック") {
		t.Errorf("prompt should contain corner title, got: %s", prompt)
	}
}

func TestLLMDesigner_DesignFlow_PromptContainsPosition(t *testing.T) {
	mc := &mockClient{
		response: json.RawMessage(`{"flow":"フロー"}`),
	}
	d := flow.NewLLMDesigner(mc, "{{position}}", 0)

	corner := config.CornerConfig{Title: "テック", Content: "内容", LengthSec: 60}
	target := model.RundownCorner{Title: "テック"}
	rd := model.Rundown{Corners: []model.RundownCorner{target}, Casts: make([]model.RundownCast, 0)}

	_, _ = d.DesignFlow(context.Background(), corner, flow.PositionMiddle, target, rd)

	if len(mc.captured) == 0 {
		t.Fatal("LLM was not called")
	}
	prompt := mc.captured[0].Messages[0].Content
	if !strings.Contains(prompt, string(flow.PositionMiddle)) {
		t.Errorf("prompt should contain position, got: %s", prompt)
	}
}

func TestLLMDesigner_DesignFlow_PromptContainsSelectionReason(t *testing.T) {
	mc := &mockClient{
		response: json.RawMessage(`{"flow":"フロー"}`),
	}
	d := flow.NewLLMDesigner(mc, "{{selection_reason}}", 0)

	corner := config.CornerConfig{Title: "テック", Content: "内容", LengthSec: 60}
	target := model.RundownCorner{Title: "テック", SelectionReason: "AI記事が重要だから"}
	rd := model.Rundown{Corners: []model.RundownCorner{target}, Casts: make([]model.RundownCast, 0)}

	_, _ = d.DesignFlow(context.Background(), corner, flow.PositionOpening, target, rd)

	if len(mc.captured) == 0 {
		t.Fatal("LLM was not called")
	}
	prompt := mc.captured[0].Messages[0].Content
	if !strings.Contains(prompt, "AI記事が重要だから") {
		t.Errorf("prompt should contain selection_reason, got: %s", prompt)
	}
}

func TestLLMDesigner_DesignFlow_PromptContainsProgram(t *testing.T) {
	mc := &mockClient{
		response: json.RawMessage(`{"flow":"フロー"}`),
	}
	d := flow.NewLLMDesigner(mc, "{{program}}", 0)

	corner := config.CornerConfig{Title: "テック", Content: "内容", LengthSec: 60}
	target := model.RundownCorner{Title: "テック"}
	rd := model.Rundown{
		Corners: []model.RundownCorner{
			{Title: "オープニング", Articles: make([]model.RundownArticle, 0)},
			target,
		},
		Casts: []model.RundownCast{{CharacterID: "zundamon", Role: "MC", Type: "regular"}},
	}

	_, _ = d.DesignFlow(context.Background(), corner, flow.PositionMiddle, target, rd)

	if len(mc.captured) == 0 {
		t.Fatal("LLM was not called")
	}
	prompt := mc.captured[0].Messages[0].Content
	if !strings.Contains(prompt, "オープニング") {
		t.Errorf("prompt should contain program corners, got: %s", prompt)
	}
	if !strings.Contains(prompt, "zundamon") {
		t.Errorf("prompt should contain cast info, got: %s", prompt)
	}
}

func TestLLMDesigner_DesignFlow_LLMError(t *testing.T) {
	mc := &mockClient{err: context.Canceled}
	d := flow.NewLLMDesigner(mc, "{{corner}} {{position}} {{articles}} {{selection_reason}} {{program}}", 0)

	corner := config.CornerConfig{Title: "テック"}
	target := model.RundownCorner{Title: "テック"}
	rd := model.Rundown{Corners: []model.RundownCorner{target}, Casts: make([]model.RundownCast, 0)}

	_, err := d.DesignFlow(context.Background(), corner, flow.PositionOpening, target, rd)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestLLMDesigner_DesignFlow_ArticleBodySerializedToProgram(t *testing.T) {
	mc := &mockClient{
		response: json.RawMessage(`{"flow":"フロー"}`),
	}
	d := flow.NewLLMDesigner(mc, "{{program}}", 0)

	corner := config.CornerConfig{Title: "テック"}
	target := model.RundownCorner{
		Title: "テック",
		Articles: []model.RundownArticle{
			{URL: "u1", Title: "記事1", Body: "ユニークな本文テキスト", Points: []string{"p1"}},
		},
	}
	rd := model.Rundown{
		Corners: []model.RundownCorner{target},
		Casts:   make([]model.RundownCast, 0),
	}

	_, _ = d.DesignFlow(context.Background(), corner, flow.PositionOpening, target, rd)

	if len(mc.captured) == 0 {
		t.Fatal("LLM was not called")
	}
	prompt := mc.captured[0].Messages[0].Content
	if !strings.Contains(prompt, "ユニークな本文テキスト") {
		t.Errorf("prompt should contain article body via program serialization, got: %s", prompt)
	}
}

func TestLLMDesigner_DesignFlow_NoArticleCorner_EmptyArticlesInPrompt(t *testing.T) {
	mc := &mockClient{
		response: json.RawMessage(`{"flow":"番組全体の導入です"}`),
	}
	d := flow.NewLLMDesigner(mc, "{{articles}}", 0)

	corner := config.CornerConfig{Title: "オープニング", Content: "導入", LengthSec: 30}
	target := model.RundownCorner{Title: "オープニング", Articles: make([]model.RundownArticle, 0)}
	rd := model.Rundown{Corners: []model.RundownCorner{target}, Casts: make([]model.RundownCast, 0)}

	got, err := d.DesignFlow(context.Background(), corner, flow.PositionOpening, target, rd)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "番組全体の導入です" {
		t.Errorf("flow: got %q, want %q", got, "番組全体の導入です")
	}
}

func TestLLMDesigner_DesignFlow_AppearanceCountIncludedInProgram(t *testing.T) {
	mc := &mockClient{
		response: json.RawMessage(`{"flow":"フロー"}`),
	}
	d := flow.NewLLMDesigner(mc, "{{program}}", 0)

	corner := config.CornerConfig{Title: "テック"}
	target := model.RundownCorner{Title: "テック"}
	rd := model.Rundown{
		Corners: []model.RundownCorner{target},
		Casts: []model.RundownCast{
			{CharacterID: "zundamon", Role: "MC", Type: "regular", AppearanceCount: 5},
		},
	}

	_, _ = d.DesignFlow(context.Background(), corner, flow.PositionOpening, target, rd)

	if len(mc.captured) == 0 {
		t.Fatal("LLM was not called")
	}
	prompt := mc.captured[0].Messages[0].Content
	if !strings.Contains(prompt, "appearance_count") {
		t.Errorf("program JSON should contain appearance_count field for cast, got: %s", prompt)
	}
}

func TestLLMDesigner_DesignFlow_AppearanceCountIsConverted(t *testing.T) {
	// appearance_count in prompt should be PastAppearanceCount() (new_count - 1), not raw value
	mc := &mockClient{
		response: json.RawMessage(`{"flow":"フロー"}`),
	}
	d := flow.NewLLMDesigner(mc, "{{program}}", 0)

	corner := config.CornerConfig{Title: "テック"}
	target := model.RundownCorner{Title: "テック"}
	rd := model.Rundown{
		Corners: []model.RundownCorner{target},
		Casts: []model.RundownCast{
			{CharacterID: "zundamon", Role: "MC", Type: "regular", AppearanceCount: 5},
		},
	}

	_, _ = d.DesignFlow(context.Background(), corner, flow.PositionOpening, target, rd)

	if len(mc.captured) == 0 {
		t.Fatal("LLM was not called")
	}
	prompt := mc.captured[0].Messages[0].Content
	// AppearanceCount=5 → PastAppearanceCount()=4 should appear in prompt
	if !strings.Contains(prompt, `"appearance_count":4`) {
		t.Errorf("program JSON should contain appearance_count:4 (converted from 5), got: %s", prompt)
	}
}
