package write_test

import (
	"context"
	"encoding/json"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/canpok1/vox-radio/internal/cache"
	"github.com/canpok1/vox-radio/internal/config"
	"github.com/canpok1/vox-radio/internal/model"
	"github.com/canpok1/vox-radio/internal/script/llm"
	"github.com/canpok1/vox-radio/internal/script/write"
)

type mockClient struct {
	response  json.RawMessage
	err       error
	callCount int
	responses []json.RawMessage
	captured  []llm.CompletionRequest
}

func (m *mockClient) Complete(_ context.Context, req llm.CompletionRequest) (json.RawMessage, error) {
	m.captured = append(m.captured, req)
	if len(m.responses) > 0 {
		idx := m.callCount
		m.callCount++
		if idx < len(m.responses) {
			return m.responses[idx], m.err
		}
	}
	return m.response, m.err
}

var linesJSON = json.RawMessage(`{
  "lines": [
    {"speaker_role": "zundamon", "text": "こんにちは"},
    {"speaker_role": "metan", "text": "よろしく"}
  ]
}`)

func TestLLMWriter_Write_Success(t *testing.T) {
	mc := &mockClient{response: linesJSON}
	w := write.NewLLMWriter(mc, "corner={{corner}} articles={{articles}} flow={{flow}} cast={{cast_info}}", 0, nil)

	corner := config.CornerConfig{Title: "コーナー1", Content: "内容", LengthSec: 14}
	assignments := []write.CastAssignment{
		{CharacterID: "zundamon", Type: "regular", ProgramRole: "MC", CornerRole: "司会"},
	}
	articles := []model.RundownArticle{{URL: "https://example.com/1", Title: "記事1", Body: "本文", Points: []string{"p1"}}}
	chars := map[string]config.CharacterConfig{
		"zundamon": {Name: "ずんだもん", Pronoun: "ボク", SpeechSuffix: []string{"〜のだ"}, Personality: []string{"元気"}},
	}

	got, err := w.Write(context.Background(), config.ProgramConfig{}, corner, assignments, nil, nil, articles, "記事を紹介する", chars)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 2 {
		t.Errorf("Lines: got %d, want 2", len(got))
	}
	if got[0].SpeakerRole != "zundamon" {
		t.Errorf("SpeakerRole: got %q, want zundamon", got[0].SpeakerRole)
	}
}

func TestLLMWriter_Write_PromptContainsCornerAppearance(t *testing.T) {
	mc := &mockClient{response: linesJSON}
	w := write.NewLLMWriter(mc, "corner={{corner}}", 0, nil)
	w.SetCornerAppearance(5, 2) // 今回含め5回目・前回は第2回

	corner := config.CornerConfig{Title: "コーナー1", Content: "内容", LengthSec: 14}
	_, err := w.Write(context.Background(), config.ProgramConfig{}, corner, nil, nil, nil, nil, "", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	prompt := mc.captured[0].Messages[0].Content
	if !strings.Contains(prompt, `"appearance_count":5`) {
		t.Errorf("prompt should contain appearance_count:5, got: %s", prompt)
	}
	if !strings.Contains(prompt, `"last_episode_number":2`) {
		t.Errorf("prompt should contain last_episode_number:2, got: %s", prompt)
	}
}

func TestLLMWriter_Write_PromptContainsCornerAndCastInfo(t *testing.T) {
	mc := &mockClient{response: linesJSON}
	w := write.NewLLMWriter(mc, "c={{corner}} a={{articles}} f={{flow}} cast={{cast_info}}", 0, nil)

	corner := config.CornerConfig{Title: "AIコーナー", Content: "AI紹介", LengthSec: 14}
	assignments := []write.CastAssignment{
		{CharacterID: "zundamon", Type: "regular", ProgramRole: "MC", CornerRole: "司会"},
	}
	articles := []model.RundownArticle{{URL: "https://example.com/1", Title: "AI記事", Body: "AI記事の本文", Points: []string{"p1"}}}
	chars := map[string]config.CharacterConfig{
		"zundamon": {Name: "ずんだもん", Pronoun: "ボク", SpeechSuffix: []string{"〜のだ"}, Personality: []string{"元気"}},
	}

	_, _ = w.Write(context.Background(), config.ProgramConfig{}, corner, assignments, nil, nil, articles, "AI記事を紹介する", chars)

	if len(mc.captured) == 0 {
		t.Fatal("LLM was not called")
	}
	prompt := mc.captured[0].Messages[0].Content
	if !strings.Contains(prompt, "AIコーナー") {
		t.Errorf("prompt should contain corner title, got: %s", prompt)
	}
	if !strings.Contains(prompt, "AI記事の本文") {
		t.Errorf("prompt should contain article body, got: %s", prompt)
	}
	if !strings.Contains(prompt, "ずんだもん") {
		t.Errorf("prompt should contain character name, got: %s", prompt)
	}
	if !strings.Contains(prompt, "司会") {
		t.Errorf("prompt should contain role, got: %s", prompt)
	}
}

func TestLLMWriter_Write_PromptContainsFlow(t *testing.T) {
	mc := &mockClient{response: linesJSON}
	w := write.NewLLMWriter(mc, "flow={{flow}}", 0, nil)

	_, _ = w.Write(context.Background(), config.ProgramConfig{}, config.CornerConfig{}, nil, nil, nil, nil, "AIについて順に解説する", nil)

	if len(mc.captured) == 0 {
		t.Fatal("LLM was not called")
	}
	prompt := mc.captured[0].Messages[0].Content
	if !strings.Contains(prompt, "AIについて順に解説する") {
		t.Errorf("prompt should contain flow, got: %s", prompt)
	}
}

func TestLLMWriter_Write_LLMError(t *testing.T) {
	mc := &mockClient{err: context.Canceled}
	w := write.NewLLMWriter(mc, "{{corner}}", 0, nil)

	_, err := w.Write(context.Background(), config.ProgramConfig{}, config.CornerConfig{}, nil, nil, nil, nil, "", nil)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestLLMWriter_Write_PromptContainsStyles(t *testing.T) {
	mc := &mockClient{response: linesJSON}
	w := write.NewLLMWriter(mc, "cast={{cast_info}}", 0, nil)

	assignments := []write.CastAssignment{
		{CharacterID: "zundamon", Type: "regular", ProgramRole: "MC", CornerRole: "司会"},
	}
	chars := map[string]config.CharacterConfig{
		"zundamon": {
			Name: "ずんだもん", Pronoun: "ボク", SpeechSuffix: []string{"〜のだ"}, Personality: []string{"元気"},
			DefaultStyle: "ノーマル",
			Styles:       map[string]int{"ノーマル": 3, "なみだめ": 76},
		},
	}

	_, _ = w.Write(context.Background(), config.ProgramConfig{}, config.CornerConfig{}, assignments, nil, nil, nil, "", chars)

	if len(mc.captured) == 0 {
		t.Fatal("LLM was not called")
	}
	prompt := mc.captured[0].Messages[0].Content
	if !strings.Contains(prompt, "なみだめ") {
		t.Errorf("prompt should contain style name 'なみだめ', got: %s", prompt)
	}
	if !strings.Contains(prompt, "ノーマル") {
		t.Errorf("prompt should contain style name 'ノーマル', got: %s", prompt)
	}
}

func TestLLMWriter_Write_LineStyleParsed(t *testing.T) {
	linesWithStyleJSON := json.RawMessage(`{
		"lines": [
			{"speaker_role": "zundamon", "style": "なみだめ", "text": "ぐすん"},
			{"speaker_role": "metan", "text": "よろしく"}
		]
	}`)
	mc := &mockClient{response: linesWithStyleJSON}
	w := write.NewLLMWriter(mc, "{{corner}}", 0, nil)

	got, err := w.Write(context.Background(), config.ProgramConfig{}, config.CornerConfig{}, nil, nil, nil, nil, "", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("Lines: got %d, want 2", len(got))
	}
	if got[0].Style != "なみだめ" {
		t.Errorf("Style: got %q, want なみだめ", got[0].Style)
	}
	if got[1].Style != "" {
		t.Errorf("Style for line without style: got %q, want empty", got[1].Style)
	}
}

func TestLLMWriter_Write_SchemaExcludesPresetFields(t *testing.T) {
	mc := &mockClient{response: linesJSON}
	w := write.NewLLMWriter(mc, "{{corner}}", 0, nil)

	_, _ = w.Write(context.Background(), config.ProgramConfig{}, config.CornerConfig{}, nil, nil, nil, nil, "", nil)

	if len(mc.captured) == 0 {
		t.Fatal("LLM was not called")
	}
	schemaStr := string(mc.captured[0].JSONSchema)
	for _, field := range []string{"intonation", "pitch", "speed"} {
		if strings.Contains(schemaStr, field) {
			t.Errorf("schema should not contain %q (moved to direct step, ADR-0104), got: %s", field, schemaStr)
		}
	}
}

func TestLLMWriter_Write_PromptContainsConvertedTargetChars(t *testing.T) {
	mc := &mockClient{response: linesJSON}
	w := write.NewLLMWriter(mc, "c={{corner}}", 0, nil)

	// 14sec * 420chars/min / 60 = 98 chars
	corner := config.CornerConfig{Title: "Test", Content: "内容", LengthSec: 14}
	_, _ = w.Write(context.Background(), config.ProgramConfig{}, corner, nil, nil, nil, nil, "", nil)

	if len(mc.captured) == 0 {
		t.Fatal("LLM was not called")
	}
	prompt := mc.captured[0].Messages[0].Content
	if !strings.Contains(prompt, `"target_chars":98`) {
		t.Errorf("prompt should contain target_chars:98 (14sec*420/min), got: %s", prompt)
	}
	if strings.Contains(prompt, "length_sec") {
		t.Errorf("prompt should not expose length_sec to LLM, got: %s", prompt)
	}
}

func TestLLMWriter_Write_PromptContainsDirectTargetChars(t *testing.T) {
	mc := &mockClient{response: linesJSON}
	w := write.NewLLMWriter(mc, "c={{corner}}", 0, nil)

	corner := config.CornerConfig{Title: "Test", Content: "内容", TargetChars: 150}
	_, _ = w.Write(context.Background(), config.ProgramConfig{}, corner, nil, nil, nil, nil, "", nil)

	if len(mc.captured) == 0 {
		t.Fatal("LLM was not called")
	}
	prompt := mc.captured[0].Messages[0].Content
	if !strings.Contains(prompt, `"target_chars":150`) {
		t.Errorf("prompt should contain target_chars:150 (直接指定、換算なし), got: %s", prompt)
	}
}

func TestLLMWriter_Write_DirectionNotInPrompt(t *testing.T) {
	mc := &mockClient{response: linesJSON}
	w := write.NewLLMWriter(mc, "c={{corner}} p={{program}}", 0, nil)

	corner := config.CornerConfig{
		Title:     "オープニング",
		Content:   "番組の挨拶",
		Direction: "冒頭でジングルを流す演出をする。",
	}
	allCorners := []config.CornerConfig{corner}
	_, _ = w.Write(context.Background(), config.ProgramConfig{}, corner, nil, allCorners, nil, nil, "", nil)

	if len(mc.captured) == 0 {
		t.Fatal("LLM was not called")
	}
	prompt := mc.captured[0].Messages[0].Content
	if strings.Contains(prompt, "冒頭でジングルを流す演出をする。") {
		t.Errorf("direction value must not appear in write prompt, got: %s", prompt)
	}
	if strings.Contains(prompt, "direction") {
		t.Errorf("direction key must not appear in write prompt, got: %s", prompt)
	}
}

func TestLLMWriter_Write_RetroTryInjectedInPrompt(t *testing.T) {
	mc := &mockClient{response: linesJSON}
	w := write.NewLLMWriter(mc, "try={{retro_try}}", 0, nil)
	w.SetRetroTry(write.FormatRetroTry([]write.RetroTryItem{
		{Problem: "掛け合いが説明調", Action: "疑問形で崩す"},
	}))

	_, err := w.Write(context.Background(), config.ProgramConfig{}, config.CornerConfig{}, nil, nil, nil, nil, "", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	prompt := mc.captured[0].Messages[0].Content
	if !strings.Contains(prompt, "掛け合いが説明調") || !strings.Contains(prompt, "疑問形で崩す") {
		t.Errorf("prompt should contain the retro try text, got: %s", prompt)
	}
}

func TestLLMWriter_Write_RetroTryDefaultsToNone(t *testing.T) {
	mc := &mockClient{response: linesJSON}
	w := write.NewLLMWriter(mc, "try={{retro_try}}", 0, nil)

	_, err := w.Write(context.Background(), config.ProgramConfig{}, config.CornerConfig{}, nil, nil, nil, nil, "", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	prompt := mc.captured[0].Messages[0].Content
	if !strings.Contains(prompt, "try=（なし）") {
		t.Errorf("prompt should render （なし） when SetRetroTry was never called, got: %s", prompt)
	}
}

func TestFormatRetroTry_EmptyReturnsEmptyString(t *testing.T) {
	if got := write.FormatRetroTry(nil); got != "" {
		t.Errorf("FormatRetroTry(nil) = %q, want empty string", got)
	}
}

func TestFormatRetroTry_FormatsProblemAndAction(t *testing.T) {
	got := write.FormatRetroTry([]write.RetroTryItem{
		{Problem: "P1", Action: "A1"},
		{Problem: "P2", Action: "A2"},
	})
	if !strings.Contains(got, "P1") || !strings.Contains(got, "A1") || !strings.Contains(got, "P2") || !strings.Contains(got, "A2") {
		t.Errorf("FormatRetroTry should contain all problems/actions, got: %q", got)
	}
}

func TestLLMWriter_Write_RetroKeepInjectedInPrompt(t *testing.T) {
	mc := &mockClient{response: linesJSON}
	w := write.NewLLMWriter(mc, "keep={{retro_keep}}", 0, nil)
	w.SetRetroKeep(write.FormatRetroKeep([]write.RetroTryItem{
		{Problem: "掛け合いが説明調", Action: "疑問形で崩す"},
	}))

	_, err := w.Write(context.Background(), config.ProgramConfig{}, config.CornerConfig{}, nil, nil, nil, nil, "", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	prompt := mc.captured[0].Messages[0].Content
	if !strings.Contains(prompt, "掛け合いが説明調") || !strings.Contains(prompt, "疑問形で崩す") {
		t.Errorf("prompt should contain the retro keep text, got: %s", prompt)
	}
}

func TestLLMWriter_Write_RetroKeepDefaultsToNone(t *testing.T) {
	mc := &mockClient{response: linesJSON}
	w := write.NewLLMWriter(mc, "keep={{retro_keep}}", 0, nil)

	_, err := w.Write(context.Background(), config.ProgramConfig{}, config.CornerConfig{}, nil, nil, nil, nil, "", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	prompt := mc.captured[0].Messages[0].Content
	if !strings.Contains(prompt, "keep=（なし）") {
		t.Errorf("prompt should render （なし） when SetRetroKeep was never called, got: %s", prompt)
	}
}

func TestFormatRetroKeep_EmptyReturnsEmptyString(t *testing.T) {
	if got := write.FormatRetroKeep(nil); got != "" {
		t.Errorf("FormatRetroKeep(nil) = %q, want empty string", got)
	}
}

func TestFormatRetroKeep_FormatsProblemAndAction(t *testing.T) {
	got := write.FormatRetroKeep([]write.RetroTryItem{{Problem: "P1", Action: "A1"}})
	if !strings.Contains(got, "P1") || !strings.Contains(got, "A1") {
		t.Errorf("FormatRetroKeep should contain problem/action, got: %q", got)
	}
}

func TestLLMWriter_Write_PastEpisodesInjectedInPrompt(t *testing.T) {
	mc := &mockClient{response: linesJSON}
	w := write.NewLLMWriter(mc, "past={{past_episodes}}", 0, nil)
	w.SetPastEpisodes([]cache.Entry{
		{
			ProgramID: "tech-daily",
			Title:     "過去エピソード1",
			Datetime:  "2024-01-01T10:00:00Z",
			Summary:   "先週の要約",
			Corners: []cache.CornerEntry{
				{
					Title:   "コーナー1",
					Summary: "コーナー概要",
					Articles: []cache.ArticleEntry{
						{Title: "過去記事", URL: "https://example.com/old"},
					},
				},
			},
		},
	})

	_, err := w.Write(context.Background(), config.ProgramConfig{}, config.CornerConfig{}, nil, nil, nil, nil, "", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(mc.captured) == 0 {
		t.Fatal("LLM was not called")
	}
	prompt := mc.captured[0].Messages[0].Content
	if !strings.Contains(prompt, "先週の要約") {
		t.Errorf("prompt should contain past episode summary, got: %s", prompt)
	}
	if !strings.Contains(prompt, "コーナー1") {
		t.Errorf("prompt should contain past corner title, got: %s", prompt)
	}
	if strings.Contains(prompt, "過去エピソード1") {
		t.Errorf("prompt should NOT contain past episode title (Entry.Title excluded), got: %s", prompt)
	}
	if strings.Contains(prompt, "https://example.com/old") {
		t.Errorf("prompt should NOT contain article URL (Articles excluded), got: %s", prompt)
	}
}

func TestLLMWriter_Write_NoPastEpisodes_ShowsNone(t *testing.T) {
	mc := &mockClient{response: linesJSON}
	w := write.NewLLMWriter(mc, "past={{past_episodes}}", 0, nil)

	_, err := w.Write(context.Background(), config.ProgramConfig{}, config.CornerConfig{}, nil, nil, nil, nil, "", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(mc.captured) == 0 {
		t.Fatal("LLM was not called")
	}
	prompt := mc.captured[0].Messages[0].Content
	if !strings.Contains(prompt, "なし") {
		t.Errorf("prompt should indicate no past episodes, got: %s", prompt)
	}
}

func TestLLMWriter_Write_PromptContainsProgramInfo(t *testing.T) {
	mc := &mockClient{response: linesJSON}
	w := write.NewLLMWriter(mc, "program={{program}}", 0, nil)

	program := config.ProgramConfig{
		Title:       "今日のテックニュース",
		Description: "毎日5分のニュースラジオ",
	}
	allCorners := []config.CornerConfig{
		{Title: "オープニング", Content: "番組の挨拶"},
		{Title: "テックニュース", Content: "記事紹介"},
		{Title: "エンディング", Content: "まとめ"},
	}
	corner := config.CornerConfig{Title: "オープニング", Content: "番組の挨拶"}

	_, _ = w.Write(context.Background(), program, corner, nil, allCorners, nil, nil, "", nil)

	if len(mc.captured) == 0 {
		t.Fatal("LLM was not called")
	}
	prompt := mc.captured[0].Messages[0].Content
	if !strings.Contains(prompt, "今日のテックニュース") {
		t.Errorf("prompt should contain program title, got: %s", prompt)
	}
	if !strings.Contains(prompt, "毎日5分のニュースラジオ") {
		t.Errorf("prompt should contain program description, got: %s", prompt)
	}
	if !strings.Contains(prompt, "オープニング") {
		t.Errorf("prompt should contain corner title 'オープニング', got: %s", prompt)
	}
	if !strings.Contains(prompt, "テックニュース") {
		t.Errorf("prompt should contain corner title 'テックニュース', got: %s", prompt)
	}
	if !strings.Contains(prompt, "エンディング") {
		t.Errorf("prompt should contain corner title 'エンディング', got: %s", prompt)
	}
	// 他コーナーの content は {{program}} に露出してはならない（コーナー先取り防止）。
	// 当該コーナーの content は {{corner}} 側で渡されるため {{program}} には含まれない。
	for _, leaked := range []string{"番組の挨拶", "記事紹介", "まとめ"} {
		if strings.Contains(prompt, leaked) {
			t.Errorf("prompt {{program}} should NOT contain corner content %q, got: %s", leaked, prompt)
		}
	}
}

func TestLLMWriter_Write_PreviousCornersInjectedInPrompt(t *testing.T) {
	mc := &mockClient{response: linesJSON}
	w := write.NewLLMWriter(mc, "prev={{previous_corners}}", 0, nil)

	previousCorners := []model.CornerLines{
		{
			Title: "オープニング",
			Lines: []model.Line{
				{SpeakerRole: "zundamon", Text: "こんにちは！今日もよろしくのだ！", Style: "ノーマル"},
				{SpeakerRole: "metan", Text: "よろしくお願いします。"},
			},
		},
	}

	_, err := w.Write(context.Background(), config.ProgramConfig{}, config.CornerConfig{}, nil, nil, previousCorners, nil, "", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(mc.captured) == 0 {
		t.Fatal("LLM was not called")
	}
	prompt := mc.captured[0].Messages[0].Content
	if !strings.Contains(prompt, "オープニング") {
		t.Errorf("prompt should contain previous corner title, got: %s", prompt)
	}
	if !strings.Contains(prompt, "こんにちは") {
		t.Errorf("prompt should contain previous corner text, got: %s", prompt)
	}
	if !strings.Contains(prompt, "zundamon") {
		t.Errorf("prompt should contain previous corner speaker_role, got: %s", prompt)
	}
	if strings.Contains(prompt, "ノーマル") {
		t.Errorf("prompt should NOT contain style field from previous corner, got: %s", prompt)
	}
}

func TestLLMWriter_Write_NoPreviousCorners_ShowsNone(t *testing.T) {
	mc := &mockClient{response: linesJSON}
	w := write.NewLLMWriter(mc, "prev={{previous_corners}}", 0, nil)

	_, err := w.Write(context.Background(), config.ProgramConfig{}, config.CornerConfig{}, nil, nil, nil, nil, "", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(mc.captured) == 0 {
		t.Fatal("LLM was not called")
	}
	prompt := mc.captured[0].Messages[0].Content
	if !strings.Contains(prompt, "なし") {
		t.Errorf("prompt should show （なし） when no previous corners, got: %s", prompt)
	}
}

func TestLLMWriter_SetEpisodeNumber_InjectsSectionIntoPrompt(t *testing.T) {
	mc := &mockClient{response: linesJSON}
	w := write.NewLLMWriter(mc, "section={{episode_section}}", 0, nil)
	w.SetEpisodeNumber(5)

	_, err := w.Write(context.Background(), config.ProgramConfig{}, config.CornerConfig{}, nil, nil, nil, nil, "", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(mc.captured) == 0 {
		t.Fatal("LLM was not called")
	}
	prompt := mc.captured[0].Messages[0].Content
	if !strings.Contains(prompt, "第5回") {
		t.Errorf("prompt should contain 第5回, got: %s", prompt)
	}
	if !strings.Contains(prompt, "今回の放送回") {
		t.Errorf("prompt should contain the 今回の放送回 section, got: %s", prompt)
	}
}

// single_shot（episodeNumber <= 0）では「今回の放送回」セクションごと出力しない。
// 「（不明）」も出さない。
func TestLLMWriter_SetEpisodeNumber_Zero_OmitsSection(t *testing.T) {
	mc := &mockClient{response: linesJSON}
	w := write.NewLLMWriter(mc, "section=[{{episode_section}}]", 0, nil)
	// default is 0 (no SetEpisodeNumber call) = single-shot

	_, err := w.Write(context.Background(), config.ProgramConfig{}, config.CornerConfig{}, nil, nil, nil, nil, "", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(mc.captured) == 0 {
		t.Fatal("LLM was not called")
	}
	prompt := mc.captured[0].Messages[0].Content
	if strings.Contains(prompt, "今回の放送回") {
		t.Errorf("prompt should omit the 今回の放送回 section when episode number is 0, got: %s", prompt)
	}
	if strings.Contains(prompt, "（不明）") {
		t.Errorf("prompt should not contain （不明） when episode number is 0, got: %s", prompt)
	}
	if !strings.Contains(prompt, "section=[]") {
		t.Errorf("episode_section should be empty when episode number is 0, got: %s", prompt)
	}
}

func TestLLMWriter_Write_PromptContainsVarietyInstruction(t *testing.T) {
	templateBytes, err := os.ReadFile("../../cli/prompts/write.md")
	if err != nil {
		t.Fatalf("failed to read write.md: %v", err)
	}

	mc := &mockClient{response: linesJSON}
	w := write.NewLLMWriter(mc, string(templateBytes), 0, nil)
	w.SetPastEpisodes([]cache.Entry{
		{
			ProgramID: "tech-daily",
			Title:     "過去エピソード1",
			Datetime:  "2024-01-01T10:00:00Z",
			Summary:   "先週の要約",
			Corners:   []cache.CornerEntry{{Title: "コーナー1", Summary: "コーナー概要"}},
		},
	})

	_, err = w.Write(context.Background(), config.ProgramConfig{}, config.CornerConfig{}, nil, nil, nil, nil, "", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(mc.captured) == 0 {
		t.Fatal("LLM was not called")
	}
	prompt := mc.captured[0].Messages[0].Content
	if !strings.Contains(prompt, "意図せず似た切り口・ネタ・オチを繰り返さないこと") {
		t.Errorf("prompt should contain unintentional repetition avoidance instruction, got: %s", prompt)
	}
	if !strings.Contains(prompt, "反復を自覚したセリフ") {
		t.Errorf("prompt should contain intentional repetition instruction, got: %s", prompt)
	}
	if !strings.Contains(prompt, "オチ・リアクションのパターンをワンパターンにせず") {
		t.Errorf("prompt should contain reaction variety instruction, got: %s", prompt)
	}
}

func TestLLMWriter_SetCasts_GuestInjectedIntoPrompt(t *testing.T) {
	mc := &mockClient{response: linesJSON}
	w := write.NewLLMWriter(mc, "{{guest_info}}", 0, nil)

	w.SetCasts([]model.RundownCast{
		{CharacterID: "guest_char", Role: "古参リスナー出身の常連ゲスト", Type: "guest"},
	})

	_, err := w.Write(context.Background(), config.ProgramConfig{}, config.CornerConfig{}, nil, nil, nil, nil, "", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	prompt := mc.captured[0].Messages[0].Content
	if !strings.Contains(prompt, "guest_char") {
		t.Errorf("prompt should contain guest character ID, got: %s", prompt)
	}
	if !strings.Contains(prompt, "古参リスナー出身の常連ゲスト") {
		t.Errorf("prompt should contain guest role, got: %s", prompt)
	}
}

func TestLLMWriter_NoCasts_InformsLLM(t *testing.T) {
	mc := &mockClient{response: linesJSON}
	w := write.NewLLMWriter(mc, "{{guest_info}}", 0, nil)
	// SetCasts を呼ばない（デフォルトはキャストなし）

	_, err := w.Write(context.Background(), config.ProgramConfig{}, config.CornerConfig{}, nil, nil, nil, nil, "", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	prompt := mc.captured[0].Messages[0].Content
	// ゲストなし回であることを LLM に伝えること
	if !strings.Contains(prompt, "ゲストのいない通常回") {
		t.Errorf("prompt should inform LLM of no-guest episode, got: %s", prompt)
	}
}

func TestLLMWriter_SetCasts_OnlyRegular_InformsLLM(t *testing.T) {
	mc := &mockClient{response: linesJSON}
	w := write.NewLLMWriter(mc, "{{guest_info}}", 0, nil)

	w.SetCasts([]model.RundownCast{
		{CharacterID: "zundamon", Role: "MC", Type: "regular"},
	})

	_, err := w.Write(context.Background(), config.ProgramConfig{}, config.CornerConfig{}, nil, nil, nil, nil, "", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	prompt := mc.captured[0].Messages[0].Content
	// レギュラーのみの場合もゲストなし扱い
	if !strings.Contains(prompt, "ゲストなし") {
		t.Errorf("prompt should inform LLM of no-guest episode (regular only), got: %s", prompt)
	}
}

func TestLLMWriter_CastInfo_BothRoles(t *testing.T) {
	mc := &mockClient{response: linesJSON}
	w := write.NewLLMWriter(mc, "cast={{cast_info}}", 0, nil)

	assignments := []write.CastAssignment{
		{CharacterID: "zundamon", Type: "regular", ProgramRole: "番組MC。進行役。", CornerRole: "ボケ担当"},
	}
	chars := map[string]config.CharacterConfig{
		"zundamon": {Name: "ずんだもん", Pronoun: "ボク", SpeechSuffix: []string{"〜のだ"}, Personality: []string{"元気"}},
	}

	_, _ = w.Write(context.Background(), config.ProgramConfig{}, config.CornerConfig{}, assignments, nil, nil, nil, "", chars)

	if len(mc.captured) == 0 {
		t.Fatal("LLM was not called")
	}
	prompt := mc.captured[0].Messages[0].Content
	if !strings.Contains(prompt, "番組MC。進行役。") {
		t.Errorf("prompt should contain program role, got: %s", prompt)
	}
	if !strings.Contains(prompt, "ボケ担当") {
		t.Errorf("prompt should contain corner role, got: %s", prompt)
	}
}

func TestLLMWriter_CastInfo_ProgramRoleOnly_WhenNoCornerRole(t *testing.T) {
	mc := &mockClient{response: linesJSON}
	w := write.NewLLMWriter(mc, "cast={{cast_info}}", 0, nil)

	assignments := []write.CastAssignment{
		{CharacterID: "zundamon", Type: "regular", ProgramRole: "番組MC。進行役。", CornerRole: ""},
	}
	chars := map[string]config.CharacterConfig{
		"zundamon": {Name: "ずんだもん"},
	}

	_, _ = w.Write(context.Background(), config.ProgramConfig{}, config.CornerConfig{}, assignments, nil, nil, nil, "", chars)

	if len(mc.captured) == 0 {
		t.Fatal("LLM was not called")
	}
	prompt := mc.captured[0].Messages[0].Content
	if !strings.Contains(prompt, "番組MC。進行役。") {
		t.Errorf("prompt should contain program role, got: %s", prompt)
	}
	// コーナーロール未指定時はコーナーロール記述がないこと
	if strings.Contains(prompt, "コーナーロール") {
		t.Errorf("prompt should NOT contain 'コーナーロール' when corner role is empty, got: %s", prompt)
	}
}

func TestLLMWriter_CastInfo_AnonymousCharacter_HidesNameAndAvoidsAddressing(t *testing.T) {
	templateBytes, err := os.ReadFile("../../cli/prompts/write.md")
	if err != nil {
		t.Fatalf("failed to read write.md: %v", err)
	}

	mc := &mockClient{response: linesJSON}
	w := write.NewLLMWriter(mc, string(templateBytes), 0, nil)

	assignments := []write.CastAssignment{
		{CharacterID: "guest_a", Type: "guest", ProgramRole: "ゲスト"},
	}
	chars := map[string]config.CharacterConfig{
		"guest_a": {Name: "", Pronoun: "わたし", SpeechSuffix: []string{"です"}, Personality: []string{"物静か"}},
	}

	_, err = w.Write(context.Background(), config.ProgramConfig{}, config.CornerConfig{}, assignments, nil, nil, nil, "", chars)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(mc.captured) == 0 {
		t.Fatal("LLM was not called")
	}
	prompt := mc.captured[0].Messages[0].Content
	if !strings.Contains(prompt, "名前=非公開") {
		t.Errorf("prompt should mark anonymous character's name as undisclosed, got: %s", prompt)
	}
	if !strings.Contains(prompt, "他の出演者もその人物への呼びかけ自体を行わないでください") {
		t.Errorf("prompt should instruct not to address the anonymous character, got: %s", prompt)
	}
	if !strings.Contains(prompt, "わたし") {
		t.Errorf("prompt should still contain pronoun for anonymous character, got: %s", prompt)
	}
}

func TestLLMWriter_CastInfo_MixedNamedAndAnonymousCharacters(t *testing.T) {
	mc := &mockClient{response: linesJSON}
	w := write.NewLLMWriter(mc, "cast={{cast_info}}", 0, nil)

	assignments := []write.CastAssignment{
		{CharacterID: "zundamon", Type: "regular", ProgramRole: "MC"},
		{CharacterID: "guest_a", Type: "guest", ProgramRole: "ゲスト"},
	}
	chars := map[string]config.CharacterConfig{
		"zundamon": {Name: "ずんだもん", Pronoun: "ボク"},
		"guest_a":  {Name: "", Pronoun: "わたし"},
	}

	_, _ = w.Write(context.Background(), config.ProgramConfig{}, config.CornerConfig{}, assignments, nil, nil, nil, "", chars)

	if len(mc.captured) == 0 {
		t.Fatal("LLM was not called")
	}
	prompt := mc.captured[0].Messages[0].Content
	if !strings.Contains(prompt, "名前=ずんだもん") {
		t.Errorf("prompt should contain named character's name, got: %s", prompt)
	}
	if !strings.Contains(prompt, "名前=非公開") {
		t.Errorf("prompt should mark anonymous character's name as undisclosed, got: %s", prompt)
	}
}

func TestLLMWriter_SetRecordedAt_InjectsIntoPrompt(t *testing.T) {
	mc := &mockClient{response: linesJSON}
	w := write.NewLLMWriter(mc, "recorded={{recorded_at}} tz={{timezone}}", 0, nil)

	loc, err := time.LoadLocation("Asia/Tokyo")
	if err != nil {
		t.Fatalf("time.LoadLocation: %v", err)
	}
	recordedAt := time.Date(2026, 6, 6, 19, 0, 0, 0, loc)
	w.SetRecordedAt(recordedAt, loc)

	_, _ = w.Write(context.Background(), config.ProgramConfig{}, config.CornerConfig{LengthSec: 14}, nil, nil, nil, nil, "", nil)

	if len(mc.captured) == 0 {
		t.Fatal("LLM was not called")
	}
	prompt := mc.captured[0].Messages[0].Content
	if !strings.Contains(prompt, "2026-06-06T19:00:00+09:00") {
		t.Errorf("prompt should contain recorded_at RFC3339, got: %s", prompt)
	}
	if !strings.Contains(prompt, "Asia/Tokyo") {
		t.Errorf("prompt should contain timezone name, got: %s", prompt)
	}
}

func TestLLMWriter_SetRecordedAt_Unset_UsesPlaceholder(t *testing.T) {
	mc := &mockClient{response: linesJSON}
	w := write.NewLLMWriter(mc, "recorded={{recorded_at}} tz={{timezone}}", 0, nil)

	_, _ = w.Write(context.Background(), config.ProgramConfig{}, config.CornerConfig{LengthSec: 14}, nil, nil, nil, nil, "", nil)

	if len(mc.captured) == 0 {
		t.Fatal("LLM was not called")
	}
	prompt := mc.captured[0].Messages[0].Content
	if !strings.Contains(prompt, "（不明）") {
		t.Errorf("unset recorded_at/timezone should show placeholder, got: %s", prompt)
	}
}

func TestLLMWriter_Write_ProgramScriptNoteInPrompt(t *testing.T) {
	mc := &mockClient{response: linesJSON}
	w := write.NewLLMWriter(mc, "{{program_script_note}}", 0, nil)

	program := config.ProgramConfig{ScriptNote: "記事タイトルを正確に伝えること"}
	_, _ = w.Write(context.Background(), program, config.CornerConfig{}, nil, nil, nil, nil, "", nil)

	if len(mc.captured) == 0 {
		t.Fatal("LLM was not called")
	}
	prompt := mc.captured[0].Messages[0].Content
	if !strings.Contains(prompt, "記事タイトルを正確に伝えること") {
		t.Errorf("program_script_note should appear in prompt, got: %s", prompt)
	}
}

func TestLLMWriter_Write_ProgramScriptNoteEmptyUsesNone(t *testing.T) {
	mc := &mockClient{response: linesJSON}
	w := write.NewLLMWriter(mc, "{{program_script_note}}", 0, nil)

	_, _ = w.Write(context.Background(), config.ProgramConfig{}, config.CornerConfig{}, nil, nil, nil, nil, "", nil)

	if len(mc.captured) == 0 {
		t.Fatal("LLM was not called")
	}
	prompt := mc.captured[0].Messages[0].Content
	if !strings.Contains(prompt, "（なし）") {
		t.Errorf("empty program_script_note should be rendered as （なし）, got: %s", prompt)
	}
}

func TestLLMWriter_Write_CornerScriptNoteInCornerJSON(t *testing.T) {
	mc := &mockClient{response: linesJSON}
	w := write.NewLLMWriter(mc, "{{corner}}", 0, nil)

	corner := config.CornerConfig{Title: "テスト", Content: "内容", ScriptNote: "コーナー台本指示", LengthSec: 14}
	_, _ = w.Write(context.Background(), config.ProgramConfig{}, corner, nil, nil, nil, nil, "", nil)

	if len(mc.captured) == 0 {
		t.Fatal("LLM was not called")
	}
	prompt := mc.captured[0].Messages[0].Content
	if !strings.Contains(prompt, "コーナー台本指示") {
		t.Errorf("corner script_note should appear in {{corner}} prompt, got: %s", prompt)
	}
}

func TestLLMWriter_Write_ProgramDirectionNotInPrompt(t *testing.T) {
	mc := &mockClient{response: linesJSON}
	w := write.NewLLMWriter(mc, "{{program}} {{corner}}", 0, nil)

	program := config.ProgramConfig{Title: "テスト", Direction: "番組演出方針（direct専用）"}
	_, _ = w.Write(context.Background(), program, config.CornerConfig{LengthSec: 14}, nil, nil, nil, nil, "", nil)

	if len(mc.captured) == 0 {
		t.Fatal("LLM was not called")
	}
	prompt := mc.captured[0].Messages[0].Content
	if strings.Contains(prompt, "番組演出方針（direct専用）") {
		t.Errorf("program.Direction must not leak into write prompt, got: %s", prompt)
	}
}

func TestBuildLinesSchema_ExcludesPresetFields(t *testing.T) {
	schema := write.BuildLinesSchema()

	schemaStr := string(schema)
	for _, field := range []string{"intonation", "pitch", "speed"} {
		if strings.Contains(schemaStr, field) {
			t.Errorf("schema should not contain %q (moved to direct step, ADR-0104), got: %s", field, schemaStr)
		}
	}
	for _, want := range []string{"speaker_role", "style", "text"} {
		if !strings.Contains(schemaStr, want) {
			t.Errorf("schema should contain %q, got: %s", want, schemaStr)
		}
	}
}

func TestLLMWriter_Write_CornerDirectionNotInCornerJSON(t *testing.T) {
	mc := &mockClient{response: linesJSON}
	w := write.NewLLMWriter(mc, "{{corner}}", 0, nil)

	corner := config.CornerConfig{Title: "テスト", Content: "内容", Direction: "コーナー演出方針（direct専用）", LengthSec: 14}
	_, _ = w.Write(context.Background(), config.ProgramConfig{}, corner, nil, nil, nil, nil, "", nil)

	if len(mc.captured) == 0 {
		t.Fatal("LLM was not called")
	}
	prompt := mc.captured[0].Messages[0].Content
	if strings.Contains(prompt, "コーナー演出方針（direct専用）") {
		t.Errorf("corner.Direction must not leak into write prompt, got: %s", prompt)
	}
}
