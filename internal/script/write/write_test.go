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
	articles := []model.RundownArticle{{URL: "https://example.com/1", Title: "記事1", Summary: "要約", Points: []string{"p1"}}}
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
	articles := []model.RundownArticle{{URL: "https://example.com/1", Title: "AI記事", Summary: "AI要約", Points: []string{"p1"}}}
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
	if !strings.Contains(prompt, "AI要約") {
		t.Errorf("prompt should contain summary, got: %s", prompt)
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

func TestLLMWriter_Write_LinePresetFieldsParsed(t *testing.T) {
	linesWithPresetsJSON := json.RawMessage(`{
		"lines": [
			{"speaker_role": "zundamon", "intonation": "表現豊か", "pitch": "高め", "speed": "早口", "text": "テスト"}
		]
	}`)
	mc := &mockClient{response: linesWithPresetsJSON}
	w := write.NewLLMWriter(mc, "{{corner}}", 0, nil)

	got, err := w.Write(context.Background(), config.ProgramConfig{}, config.CornerConfig{}, nil, nil, nil, nil, "", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("Lines: got %d, want 1", len(got))
	}
	if got[0].Intonation != "表現豊か" {
		t.Errorf("Intonation: got %q, want 表現豊か", got[0].Intonation)
	}
	if got[0].Pitch != "高め" {
		t.Errorf("Pitch: got %q, want 高め", got[0].Pitch)
	}
	if got[0].Speed != "早口" {
		t.Errorf("Speed: got %q, want 早口", got[0].Speed)
	}
}

func TestLLMWriter_Write_SchemaIncludesPresetEnums(t *testing.T) {
	mc := &mockClient{response: linesJSON}
	cfg := &config.Config{
		Voicevox: config.VoicevoxConfig{
			Presets: &config.VoicevoxPresets{
				Intonation: map[string]float64{"棒読み": 0.0, "標準": 1.0},
				Pitch:      map[string]float64{"低め": -0.05, "標準": 0.0},
				Speed:      map[string]float64{"ゆっくり": 0.8, "標準": 1.0},
			},
		},
	}
	w := write.NewLLMWriter(mc, "{{corner}}", 0, cfg)

	_, _ = w.Write(context.Background(), config.ProgramConfig{}, config.CornerConfig{}, nil, nil, nil, nil, "", nil)

	if len(mc.captured) == 0 {
		t.Fatal("LLM was not called")
	}
	schemaStr := string(mc.captured[0].JSONSchema)
	if !strings.Contains(schemaStr, "棒読み") {
		t.Errorf("schema should contain intonation preset name '棒読み', got: %s", schemaStr)
	}
	if !strings.Contains(schemaStr, "低め") {
		t.Errorf("schema should contain pitch preset name '低め', got: %s", schemaStr)
	}
	if !strings.Contains(schemaStr, "ゆっくり") {
		t.Errorf("schema should contain speed preset name 'ゆっくり', got: %s", schemaStr)
	}
}

func TestLLMWriter_Write_PromptContainsPresetInfo(t *testing.T) {
	mc := &mockClient{response: linesJSON}
	cfg := &config.Config{
		Voicevox: config.VoicevoxConfig{
			Presets: &config.VoicevoxPresets{
				Intonation: map[string]float64{"棒読み": 0.0, "標準": 1.0},
				Pitch:      map[string]float64{"低め": -0.05, "標準": 0.0},
				Speed:      map[string]float64{"ゆっくり": 0.8, "標準": 1.0},
			},
		},
	}
	w := write.NewLLMWriter(mc, "preset={{preset_info}}", 0, cfg)

	_, _ = w.Write(context.Background(), config.ProgramConfig{}, config.CornerConfig{}, nil, nil, nil, nil, "", nil)

	if len(mc.captured) == 0 {
		t.Fatal("LLM was not called")
	}
	prompt := mc.captured[0].Messages[0].Content
	if !strings.Contains(prompt, "棒読み") {
		t.Errorf("prompt should contain intonation preset info, got: %s", prompt)
	}
	if !strings.Contains(prompt, "低め") {
		t.Errorf("prompt should contain pitch preset info, got: %s", prompt)
	}
	if !strings.Contains(prompt, "ゆっくり") {
		t.Errorf("prompt should contain speed preset info, got: %s", prompt)
	}
}

func TestLLMWriter_Write_NoConfigUsesDefaultPresetSchema(t *testing.T) {
	mc := &mockClient{response: linesJSON}
	w := write.NewLLMWriter(mc, "{{corner}}", 0, nil) // no config

	_, _ = w.Write(context.Background(), config.ProgramConfig{}, config.CornerConfig{}, nil, nil, nil, nil, "", nil)

	if len(mc.captured) == 0 {
		t.Fatal("LLM was not called")
	}
	schemaStr := string(mc.captured[0].JSONSchema)
	// Default presets should include 標準
	if !strings.Contains(schemaStr, "標準") {
		t.Errorf("schema should contain default preset name '標準', got: %s", schemaStr)
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
				{SpeakerRole: "zundamon", Text: "こんにちは！今日もよろしくのだ！", Style: "ノーマル", Intonation: "標準"},
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
	if strings.Contains(prompt, "標準") {
		t.Errorf("prompt should NOT contain intonation field from previous corner, got: %s", prompt)
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

func TestLLMWriter_SetEpisodeNumber_InjectsNumberIntoPrompt(t *testing.T) {
	mc := &mockClient{response: linesJSON}
	w := write.NewLLMWriter(mc, "episode={{episode_number}}", 0, nil)
	w.SetEpisodeNumber(5)

	_, err := w.Write(context.Background(), config.ProgramConfig{}, config.CornerConfig{}, nil, nil, nil, nil, "", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(mc.captured) == 0 {
		t.Fatal("LLM was not called")
	}
	prompt := mc.captured[0].Messages[0].Content
	if !strings.Contains(prompt, "5") {
		t.Errorf("prompt should contain episode number 5, got: %s", prompt)
	}
}

func TestLLMWriter_SetEpisodeNumber_Zero_InjectsUnknown(t *testing.T) {
	mc := &mockClient{response: linesJSON}
	w := write.NewLLMWriter(mc, "episode={{episode_number}}", 0, nil)
	// default is 0 (no SetEpisodeNumber call)

	_, err := w.Write(context.Background(), config.ProgramConfig{}, config.CornerConfig{}, nil, nil, nil, nil, "", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(mc.captured) == 0 {
		t.Fatal("LLM was not called")
	}
	prompt := mc.captured[0].Messages[0].Content
	if !strings.Contains(prompt, "（不明）") {
		t.Errorf("prompt should contain （不明） when episode number is 0, got: %s", prompt)
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

func TestBuildLinesSchema_EnumValuesMatchInput(t *testing.T) {
	schema := write.BuildLinesSchema(
		[]string{"表現豊か", "棒読み"},
		[]string{"高め", "低め"},
		[]string{"早口", "ゆっくり"},
	)

	schemaStr := string(schema)
	for _, want := range []string{"表現豊か", "棒読み", "高め", "低め", "早口", "ゆっくり"} {
		if !strings.Contains(schemaStr, want) {
			t.Errorf("schema should contain %q, got: %s", want, schemaStr)
		}
	}
}

func TestBuildLinesSchema_SortsEnumValues(t *testing.T) {
	schema := write.BuildLinesSchema(
		[]string{"z値", "a値"},
		[]string{"b値"},
		[]string{"c値"},
	)

	schemaStr := string(schema)
	// "a値" must appear before "z値" in the JSON
	aPos := strings.Index(schemaStr, "a値")
	zPos := strings.Index(schemaStr, "z値")
	if aPos == -1 || zPos == -1 {
		t.Fatalf("schema should contain 'a値' and 'z値', got: %s", schemaStr)
	}
	if aPos > zPos {
		t.Errorf("enum values should be sorted: 'a値' should appear before 'z値', got: %s", schemaStr)
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
