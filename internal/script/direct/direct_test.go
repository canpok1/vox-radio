package direct_test

import (
	"context"
	"encoding/json"
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

func TestLLMDirector_Direct_NoSEInsertions(t *testing.T) {
	mc := &mockClient{
		response: json.RawMessage(`{"se_insertions":[]}`),
	}
	d := direct.NewLLMDirector(mc, "lines={{lines}} se={{se_catalog}}", 0)

	lines := []model.Line{
		{SpeakerRole: "host", Text: "こんにちは"},
		{SpeakerRole: "guest", Text: "よろしく"},
	}
	se := model.SECatalog{Names: []string{"chime"}}

	got, err := d.Direct(context.Background(), lines, se)
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

func TestLLMDirector_Direct_WithSEInsertions(t *testing.T) {
	mc := &mockClient{
		response: json.RawMessage(`{"se_insertions":[{"after_line_index":0,"se_name":"chime","reason":"コーナー開始"}]}`),
	}
	d := direct.NewLLMDirector(mc, "{{lines}}", 0)

	lines := []model.Line{
		{SpeakerRole: "host", Text: "開始"},
		{SpeakerRole: "guest", Text: "続き"},
	}
	se := model.SECatalog{Names: []string{"chime"}}

	got, err := d.Direct(context.Background(), lines, se)
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
	if got.Segments[1].SEName != "chime" {
		t.Errorf("Segment[1].SEName: got %q, want chime", got.Segments[1].SEName)
	}
	if got.Segments[2].Type != model.SegmentTypeSpeech {
		t.Errorf("Segment[2].Type: got %q, want speech", got.Segments[2].Type)
	}
}

func TestLLMDirector_Direct_SEAfterLastLine(t *testing.T) {
	mc := &mockClient{
		response: json.RawMessage(`{"se_insertions":[{"after_line_index":1,"se_name":"transition"}]}`),
	}
	d := direct.NewLLMDirector(mc, "{{lines}}", 0)

	lines := []model.Line{
		{SpeakerRole: "host", Text: "A"},
		{SpeakerRole: "guest", Text: "B"},
	}
	se := model.SECatalog{Names: []string{"transition"}}

	got, err := d.Direct(context.Background(), lines, se)
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
		response: json.RawMessage(`{"se_insertions":[]}`),
	}
	d := direct.NewLLMDirector(mc, "{{lines}}", 0)

	lines := []model.Line{
		{SpeakerRole: "zundamon", Style: "なみだめ", Text: "ぐすん"},
		{SpeakerRole: "metan", Style: "", Text: "大丈夫？"},
	}

	got, err := d.Direct(context.Background(), lines, model.SECatalog{})
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

	_, err := d.Direct(context.Background(), nil, model.SECatalog{})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestLLMDirector_Direct_SpeechSegmentFields(t *testing.T) {
	mc := &mockClient{
		response: json.RawMessage(`{"se_insertions":[]}`),
	}
	d := direct.NewLLMDirector(mc, "{{lines}}", 0)

	lines := []model.Line{
		{SpeakerRole: "host", Text: "テストテキスト"},
	}

	got, err := d.Direct(context.Background(), lines, model.SECatalog{})
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
