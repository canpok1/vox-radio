package direct_test

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/canpok1/vox-radio/internal/model"
	"github.com/canpok1/vox-radio/internal/script/direct"
	"github.com/canpok1/vox-radio/internal/script/llm"
)

type mockClient struct {
	response json.RawMessage
	err      error
}

func (m *mockClient) Complete(_ context.Context, _ llm.CompletionRequest) (json.RawMessage, error) {
	return m.response, m.err
}

type capturingClient struct {
	response       json.RawMessage
	err            error
	capturedPrompt *string
}

func (c *capturingClient) Complete(_ context.Context, req llm.CompletionRequest) (json.RawMessage, error) {
	if len(req.Messages) > 0 {
		*c.capturedPrompt = req.Messages[0].Content
	}
	return c.response, c.err
}

func TestLLMDirector_Direct_NoInsertions(t *testing.T) {
	mc := &mockClient{
		response: json.RawMessage(`{"insertions":[]}`),
	}
	d := direct.NewLLMDirector(mc, "lines={{lines}} catalog={{asset_catalog}}", 0)

	lines := []model.Line{
		{SpeakerRole: "host", Text: "こんにちは"},
		{SpeakerRole: "guest", Text: "よろしく"},
	}
	catalog := model.AssetCatalog{
		SE:     []model.AssetCatalogEntry{{Name: "chime"}},
		BGM:    []model.AssetCatalogEntry{{Name: "talk_bgm"}},
		Jingle: []model.AssetCatalogEntry{},
	}

	got, err := d.Direct(context.Background(), lines, catalog)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got.Segments) != 2 {
		t.Errorf("Segments: got %d, want 2", len(got.Segments))
	}
	for i, seg := range got.Segments {
		if seg.Type != model.SegmentTypeSpeech {
			t.Errorf("Segment[%d].Type: got %q, want speech", i, seg.Type)
		}
	}
}

func TestLLMDirector_Direct_WithSEInsertion(t *testing.T) {
	mc := &mockClient{
		response: json.RawMessage(`{"insertions":[{"after_line_index":0,"type":"se","asset_name":"chime","reason":"コーナー開始"}]}`),
	}
	d := direct.NewLLMDirector(mc, "{{lines}}", 0)

	lines := []model.Line{
		{SpeakerRole: "host", Text: "開始"},
		{SpeakerRole: "guest", Text: "続き"},
	}
	catalog := model.AssetCatalog{
		SE:     []model.AssetCatalogEntry{{Name: "chime"}},
		BGM:    []model.AssetCatalogEntry{},
		Jingle: []model.AssetCatalogEntry{},
	}

	got, err := d.Direct(context.Background(), lines, catalog)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// [speech(host), se(chime), speech(guest)]
	if len(got.Segments) != 3 {
		t.Fatalf("Segments: got %d, want 3", len(got.Segments))
	}
	if got.Segments[0].Type != model.SegmentTypeSpeech {
		t.Errorf("Segment[0].Type: got %q, want speech", got.Segments[0].Type)
	}
	if got.Segments[1].Type != model.SegmentTypeSE {
		t.Errorf("Segment[1].Type: got %q, want se", got.Segments[1].Type)
	}
	if got.Segments[1].AssetName != "chime" {
		t.Errorf("Segment[1].AssetName: got %q, want chime", got.Segments[1].AssetName)
	}
	if got.Segments[2].Type != model.SegmentTypeSpeech {
		t.Errorf("Segment[2].Type: got %q, want speech", got.Segments[2].Type)
	}
}

func TestLLMDirector_Direct_WithBGMInsertion(t *testing.T) {
	mc := &mockClient{
		response: json.RawMessage(`{"insertions":[{"after_line_index":0,"type":"bgm","asset_name":"talk_bgm"},{"after_line_index":1,"type":"bgm","asset_name":""}]}`),
	}
	d := direct.NewLLMDirector(mc, "{{lines}}", 0)

	lines := []model.Line{
		{SpeakerRole: "host", Text: "BGM開始"},
		{SpeakerRole: "guest", Text: "BGM停止"},
	}
	catalog := model.AssetCatalog{
		SE:     []model.AssetCatalogEntry{},
		BGM:    []model.AssetCatalogEntry{{Name: "talk_bgm"}},
		Jingle: []model.AssetCatalogEntry{},
	}

	got, err := d.Direct(context.Background(), lines, catalog)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// [speech, bgm(start), speech, bgm(stop)]
	if len(got.Segments) != 4 {
		t.Fatalf("Segments: got %d, want 4", len(got.Segments))
	}
	if got.Segments[1].Type != model.SegmentTypeBGM {
		t.Errorf("Segment[1].Type: got %q, want bgm", got.Segments[1].Type)
	}
	if got.Segments[1].AssetName != "talk_bgm" {
		t.Errorf("Segment[1].AssetName: got %q, want talk_bgm", got.Segments[1].AssetName)
	}
	if got.Segments[3].Type != model.SegmentTypeBGM {
		t.Errorf("Segment[3].Type: got %q, want bgm", got.Segments[3].Type)
	}
	if got.Segments[3].AssetName != "" {
		t.Errorf("Segment[3].AssetName: got %q, want empty (stop)", got.Segments[3].AssetName)
	}
}

func TestLLMDirector_Direct_WithJingleInsertion(t *testing.T) {
	mc := &mockClient{
		response: json.RawMessage(`{"insertions":[{"after_line_index":0,"type":"jingle","asset_name":"eyecatch"}]}`),
	}
	d := direct.NewLLMDirector(mc, "{{lines}}", 0)

	lines := []model.Line{
		{SpeakerRole: "host", Text: "コーナー1"},
		{SpeakerRole: "guest", Text: "コーナー2"},
	}
	catalog := model.AssetCatalog{
		SE:     []model.AssetCatalogEntry{},
		BGM:    []model.AssetCatalogEntry{},
		Jingle: []model.AssetCatalogEntry{{Name: "eyecatch"}},
	}

	got, err := d.Direct(context.Background(), lines, catalog)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// [speech, jingle(eyecatch), speech]
	if len(got.Segments) != 3 {
		t.Fatalf("Segments: got %d, want 3", len(got.Segments))
	}
	if got.Segments[1].Type != model.SegmentTypeJingle {
		t.Errorf("Segment[1].Type: got %q, want jingle", got.Segments[1].Type)
	}
	if got.Segments[1].AssetName != "eyecatch" {
		t.Errorf("Segment[1].AssetName: got %q, want eyecatch", got.Segments[1].AssetName)
	}
}

func TestLLMDirector_Direct_InsertionAfterLastLine(t *testing.T) {
	mc := &mockClient{
		response: json.RawMessage(`{"insertions":[{"after_line_index":1,"type":"se","asset_name":"transition"}]}`),
	}
	d := direct.NewLLMDirector(mc, "{{lines}}", 0)

	lines := []model.Line{
		{SpeakerRole: "host", Text: "A"},
		{SpeakerRole: "guest", Text: "B"},
	}
	catalog := model.AssetCatalog{
		SE:     []model.AssetCatalogEntry{{Name: "transition"}},
		BGM:    []model.AssetCatalogEntry{},
		Jingle: []model.AssetCatalogEntry{},
	}

	got, err := d.Direct(context.Background(), lines, catalog)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// [speech, speech, se]
	if len(got.Segments) != 3 {
		t.Fatalf("Segments: got %d, want 3", len(got.Segments))
	}
	if got.Segments[2].Type != model.SegmentTypeSE {
		t.Errorf("Segment[2].Type: got %q, want se", got.Segments[2].Type)
	}
}

func TestLLMDirector_Direct_StylePropagated(t *testing.T) {
	mc := &mockClient{
		response: json.RawMessage(`{"insertions":[]}`),
	}
	d := direct.NewLLMDirector(mc, "{{lines}}", 0)

	lines := []model.Line{
		{SpeakerRole: "zundamon", Style: "なみだめ", Text: "ぐすん"},
		{SpeakerRole: "metan", Style: "", Text: "大丈夫？"},
	}

	got, err := d.Direct(context.Background(), lines, model.AssetCatalog{SE: []model.AssetCatalogEntry{}, BGM: []model.AssetCatalogEntry{}, Jingle: []model.AssetCatalogEntry{}})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got.Segments) != 2 {
		t.Fatalf("Segments: got %d, want 2", len(got.Segments))
	}
	if got.Segments[0].Style != "なみだめ" {
		t.Errorf("Segment[0].Style: got %q, want なみだめ", got.Segments[0].Style)
	}
	if got.Segments[1].Style != "" {
		t.Errorf("Segment[1].Style: got %q, want empty", got.Segments[1].Style)
	}
}

func TestLLMDirector_Direct_LLMError(t *testing.T) {
	mc := &mockClient{err: context.Canceled}
	d := direct.NewLLMDirector(mc, "{{lines}}", 0)

	_, err := d.Direct(context.Background(), nil, model.AssetCatalog{SE: []model.AssetCatalogEntry{}, BGM: []model.AssetCatalogEntry{}, Jingle: []model.AssetCatalogEntry{}})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestLLMDirector_Direct_SpeechSegmentFields(t *testing.T) {
	mc := &mockClient{
		response: json.RawMessage(`{"insertions":[]}`),
	}
	d := direct.NewLLMDirector(mc, "{{lines}}", 0)

	lines := []model.Line{
		{SpeakerRole: "host", Text: "テストテキスト"},
	}

	got, err := d.Direct(context.Background(), lines, model.AssetCatalog{SE: []model.AssetCatalogEntry{}, BGM: []model.AssetCatalogEntry{}, Jingle: []model.AssetCatalogEntry{}})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got.Segments) != 1 {
		t.Fatalf("Segments: got %d, want 1", len(got.Segments))
	}
	seg := got.Segments[0]
	if seg.SpeakerRole != "host" {
		t.Errorf("SpeakerRole: got %q, want host", seg.SpeakerRole)
	}
	if seg.Text != "テストテキスト" {
		t.Errorf("Text: got %q, want テストテキスト", seg.Text)
	}
}

func TestLLMDirector_Direct_CatalogDescriptionPassedToPrompt(t *testing.T) {
	var capturedPrompt string
	mc := &capturingClient{
		response:       json.RawMessage(`{"insertions":[]}`),
		capturedPrompt: &capturedPrompt,
	}
	d := direct.NewLLMDirector(mc, "{{asset_catalog}}", 0)

	catalog := model.AssetCatalog{
		SE:     []model.AssetCatalogEntry{{Name: "chime", Description: "コーナー開始時のチャイム音"}},
		BGM:    []model.AssetCatalogEntry{},
		Jingle: []model.AssetCatalogEntry{},
	}

	_, err := d.Direct(context.Background(), []model.Line{}, catalog)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(capturedPrompt, "コーナー開始時のチャイム音") {
		t.Errorf("description should appear in prompt, got: %s", capturedPrompt)
	}
}

func TestLLMDirector_Direct_CatalogNoInternalFieldsInPrompt(t *testing.T) {
	var capturedPrompt string
	mc := &capturingClient{
		response:       json.RawMessage(`{"insertions":[]}`),
		capturedPrompt: &capturedPrompt,
	}
	d := direct.NewLLMDirector(mc, "{{asset_catalog}}", 0)

	catalog := model.AssetCatalog{
		SE:     []model.AssetCatalogEntry{{Name: "chime", Description: "テスト"}},
		BGM:    []model.AssetCatalogEntry{},
		Jingle: []model.AssetCatalogEntry{},
	}

	_, err := d.Direct(context.Background(), []model.Line{}, catalog)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	for _, field := range []string{`"file"`, `"volume"`, `"duck_ratio"`, `"loop"`, `"fade_in"`, `"fade_out"`} {
		if strings.Contains(capturedPrompt, field) {
			t.Errorf("internal config field %s should not appear in prompt, got: %s", field, capturedPrompt)
		}
	}
}
