package write_test

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

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
	w := write.NewLLMWriter(mc, "corner={{corner}} summaries={{summary}} cast={{cast_info}}", 0, nil)

	corner := config.CornerConfig{Title: "コーナー1", Content: "内容", Cast: map[string]string{"zundamon": "司会"}, TargetDurationSec: 14}
	summaries := []model.Summary{{URL: "https://example.com/1", Summary: "要約", Points: []string{"p1"}}}
	chars := map[string]config.CharacterConfig{
		"zundamon": {Name: "ずんだもん", Pronoun: "ボク", SpeechSuffix: []string{"〜のだ"}, Personality: []string{"元気"}},
	}

	got, err := w.Write(context.Background(), corner, summaries, chars)
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

func TestLLMWriter_Write_PromptContainsCornerAndCastInfo(t *testing.T) {
	mc := &mockClient{response: linesJSON}
	w := write.NewLLMWriter(mc, "c={{corner}} s={{summary}} cast={{cast_info}}", 0, nil)

	corner := config.CornerConfig{Title: "AIコーナー", Content: "AI紹介", Cast: map[string]string{"zundamon": "司会"}, TargetDurationSec: 14}
	summaries := []model.Summary{{URL: "https://example.com/1", Summary: "AI要約", Points: []string{"p1"}}}
	chars := map[string]config.CharacterConfig{
		"zundamon": {Name: "ずんだもん", Pronoun: "ボク", SpeechSuffix: []string{"〜のだ"}, Personality: []string{"元気"}},
	}

	_, _ = w.Write(context.Background(), corner, summaries, chars)

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

func TestLLMWriter_Write_LLMError(t *testing.T) {
	mc := &mockClient{err: context.Canceled}
	w := write.NewLLMWriter(mc, "{{corner}}", 0, nil)

	_, err := w.Write(context.Background(), config.CornerConfig{}, nil, nil)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestLLMWriter_Write_PromptContainsStyles(t *testing.T) {
	mc := &mockClient{response: linesJSON}
	w := write.NewLLMWriter(mc, "cast={{cast_info}}", 0, nil)

	corner := config.CornerConfig{Cast: map[string]string{"zundamon": "司会"}}
	chars := map[string]config.CharacterConfig{
		"zundamon": {
			Name: "ずんだもん", Pronoun: "ボク", SpeechSuffix: []string{"〜のだ"}, Personality: []string{"元気"},
			DefaultStyle: "ノーマル",
			Styles:       map[string]int{"ノーマル": 3, "なみだめ": 76},
		},
	}

	_, _ = w.Write(context.Background(), corner, nil, chars)

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

	got, err := w.Write(context.Background(), config.CornerConfig{}, nil, nil)
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

	got, err := w.Write(context.Background(), config.CornerConfig{}, nil, nil)
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

	_, _ = w.Write(context.Background(), config.CornerConfig{}, nil, nil)

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

	_, _ = w.Write(context.Background(), config.CornerConfig{}, nil, nil)

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

	_, _ = w.Write(context.Background(), config.CornerConfig{}, nil, nil)

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

	// 14sec * 7chars/sec = 98 chars
	corner := config.CornerConfig{Title: "Test", Content: "内容", Cast: map[string]string{"zundamon": "司会"}, TargetDurationSec: 14}
	_, _ = w.Write(context.Background(), corner, nil, nil)

	if len(mc.captured) == 0 {
		t.Fatal("LLM was not called")
	}
	prompt := mc.captured[0].Messages[0].Content
	if !strings.Contains(prompt, `"target_chars":98`) {
		t.Errorf("prompt should contain target_chars:98 (14sec*7), got: %s", prompt)
	}
	if strings.Contains(prompt, "target_duration_sec") {
		t.Errorf("prompt should not expose target_duration_sec to LLM, got: %s", prompt)
	}
}
