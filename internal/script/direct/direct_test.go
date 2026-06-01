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

// helper: wrap lines into a single-corner CornerLines slice
func oneCorner(title string, lines ...model.Line) []model.CornerLines {
	return []model.CornerLines{{Title: title, Lines: lines}}
}

// helper: single empty catalog
func emptyCatalog() model.AssetCatalog {
	return model.AssetCatalog{
		SE: []model.AssetCatalogEntry{},
	}
}

func TestLLMDirector_Direct_NoInsertions(t *testing.T) {
	mc := &mockClient{
		response: json.RawMessage(`{"insertions":[]}`),
	}
	d := direct.NewLLMDirector(mc, "corners={{corners}} catalog={{asset_catalog}}", 0)

	corners := oneCorner("C1",
		model.Line{SpeakerRole: "host", Text: "こんにちは"},
		model.Line{SpeakerRole: "guest", Text: "よろしく"},
	)
	catalog := model.AssetCatalog{
		SE: []model.AssetCatalogEntry{{Name: "chime"}},
	}

	got, err := d.Direct(context.Background(), corners, catalog)
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
		response: json.RawMessage(`{"insertions":[{"corner_index":0,"after_line_index":0,"type":"se","asset_name":"chime","reason":"コーナー開始"}]}`),
	}
	d := direct.NewLLMDirector(mc, "{{corners}}", 0)

	corners := oneCorner("C1",
		model.Line{SpeakerRole: "host", Text: "開始"},
		model.Line{SpeakerRole: "guest", Text: "続き"},
	)
	catalog := model.AssetCatalog{
		SE: []model.AssetCatalogEntry{{Name: "chime"}},
	}

	got, err := d.Direct(context.Background(), corners, catalog)
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

func TestLLMDirector_Direct_CornerBGMWrapsContent(t *testing.T) {
	mc := &mockClient{
		response: json.RawMessage(`{"insertions":[]}`),
	}
	d := direct.NewLLMDirector(mc, "{{corners}}", 0)

	corners := []model.CornerLines{{
		Title: "C1",
		BGM:   "talk_bgm",
		Lines: []model.Line{
			{SpeakerRole: "host", Text: "話す"},
		},
	}}

	got, err := d.Direct(context.Background(), corners, emptyCatalog())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// [bgm(start), speech, bgm(stop)]
	if len(got.Segments) != 3 {
		t.Fatalf("Segments: got %d, want 3", len(got.Segments))
	}
	if got.Segments[0].Type != model.SegmentTypeBGM || got.Segments[0].AssetName != "talk_bgm" {
		t.Errorf("Segment[0]: got %+v, want bgm(talk_bgm)", got.Segments[0])
	}
	if got.Segments[1].Type != model.SegmentTypeSpeech {
		t.Errorf("Segment[1]: got %+v, want speech", got.Segments[1])
	}
	if got.Segments[2].Type != model.SegmentTypeBGM || got.Segments[2].AssetName != "" {
		t.Errorf("Segment[2]: got %+v, want bgm(stop)", got.Segments[2])
	}
}

func TestLLMDirector_Direct_CornerStartJinglePrependedFirst(t *testing.T) {
	mc := &mockClient{
		response: json.RawMessage(`{"insertions":[]}`),
	}
	d := direct.NewLLMDirector(mc, "{{corners}}", 0)

	corners := []model.CornerLines{{
		Title:       "C1",
		StartJingle: "opening",
		Lines:       []model.Line{{SpeakerRole: "host", Text: "話す"}},
	}}

	got, err := d.Direct(context.Background(), corners, emptyCatalog())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// [jingle(opening), speech]
	if len(got.Segments) != 2 {
		t.Fatalf("Segments: got %d, want 2", len(got.Segments))
	}
	if got.Segments[0].Type != model.SegmentTypeJingle || got.Segments[0].AssetName != "opening" {
		t.Errorf("Segment[0]: got %+v, want jingle(opening)", got.Segments[0])
	}
}

func TestLLMDirector_Direct_CornerEndJingleAppendedLast(t *testing.T) {
	mc := &mockClient{
		response: json.RawMessage(`{"insertions":[]}`),
	}
	d := direct.NewLLMDirector(mc, "{{corners}}", 0)

	corners := []model.CornerLines{{
		Title:     "C1",
		EndJingle: "ending",
		Lines:     []model.Line{{SpeakerRole: "host", Text: "話す"}},
	}}

	got, err := d.Direct(context.Background(), corners, emptyCatalog())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// [speech, jingle(ending)]
	if len(got.Segments) != 2 {
		t.Fatalf("Segments: got %d, want 2", len(got.Segments))
	}
	if got.Segments[1].Type != model.SegmentTypeJingle || got.Segments[1].AssetName != "ending" {
		t.Errorf("Segment[1]: got %+v, want jingle(ending)", got.Segments[1])
	}
}

func TestLLMDirector_Direct_CornerAllAssets_CorrectOrder(t *testing.T) {
	mc := &mockClient{
		response: json.RawMessage(`{"insertions":[]}`),
	}
	d := direct.NewLLMDirector(mc, "{{corners}}", 0)

	corners := []model.CornerLines{{
		Title:       "C1",
		StartJingle: "op",
		BGM:         "bgm1",
		EndJingle:   "ed",
		Lines:       []model.Line{{SpeakerRole: "host", Text: "話す"}},
	}}

	got, err := d.Direct(context.Background(), corners, emptyCatalog())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// [jingle(op), bgm(start), speech, bgm(stop), jingle(ed)]
	if len(got.Segments) != 5 {
		t.Fatalf("Segments: got %d, want 5", len(got.Segments))
	}
	if got.Segments[0].Type != model.SegmentTypeJingle || got.Segments[0].AssetName != "op" {
		t.Errorf("Segment[0]: want jingle(op), got %+v", got.Segments[0])
	}
	if got.Segments[1].Type != model.SegmentTypeBGM || got.Segments[1].AssetName != "bgm1" {
		t.Errorf("Segment[1]: want bgm(bgm1), got %+v", got.Segments[1])
	}
	if got.Segments[2].Type != model.SegmentTypeSpeech {
		t.Errorf("Segment[2]: want speech, got %+v", got.Segments[2])
	}
	if got.Segments[3].Type != model.SegmentTypeBGM || got.Segments[3].AssetName != "" {
		t.Errorf("Segment[3]: want bgm(stop), got %+v", got.Segments[3])
	}
	if got.Segments[4].Type != model.SegmentTypeJingle || got.Segments[4].AssetName != "ed" {
		t.Errorf("Segment[4]: want jingle(ed), got %+v", got.Segments[4])
	}
}

func TestLLMDirector_Direct_NoAssets_OnlySpeech(t *testing.T) {
	mc := &mockClient{
		response: json.RawMessage(`{"insertions":[]}`),
	}
	d := direct.NewLLMDirector(mc, "{{corners}}", 0)

	corners := []model.CornerLines{{
		Title: "C1",
		Lines: []model.Line{{SpeakerRole: "host", Text: "話す"}},
	}}

	got, err := d.Direct(context.Background(), corners, emptyCatalog())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// [speech]
	if len(got.Segments) != 1 {
		t.Fatalf("Segments: got %d, want 1 (no asset = no extra segments)", len(got.Segments))
	}
}

func TestLLMDirector_Direct_CornerAssetFields_NotInLLMPayload(t *testing.T) {
	var capturedPrompt string
	mc := &capturingClient{
		response:       json.RawMessage(`{"insertions":[]}`),
		capturedPrompt: &capturedPrompt,
	}
	d := direct.NewLLMDirector(mc, "{{corners}}", 0)

	corners := []model.CornerLines{{
		Title:       "C1",
		StartJingle: "opening",
		EndJingle:   "ending",
		BGM:         "talk_bgm",
		Lines:       []model.Line{{SpeakerRole: "host", Text: "hello"}},
	}}

	_, err := d.Direct(context.Background(), corners, emptyCatalog())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	for _, key := range []string{"start_jingle", "end_jingle", "bgm"} {
		if strings.Contains(capturedPrompt, `"`+key+`"`) {
			t.Errorf("field %q should not appear in LLM payload, got: %s", key, capturedPrompt)
		}
	}
}

func TestLLMDirector_Direct_BGMDoesNotLeakToNextCorner(t *testing.T) {
	mc := &mockClient{
		response: json.RawMessage(`{"insertions":[]}`),
	}
	d := direct.NewLLMDirector(mc, "{{corners}}", 0)

	corners := []model.CornerLines{
		{
			Title: "C1",
			BGM:   "talk_bgm",
			Lines: []model.Line{{SpeakerRole: "host", Text: "コーナー1"}},
		},
		{
			Title: "C2",
			Lines: []model.Line{{SpeakerRole: "host", Text: "コーナー2"}},
		},
	}

	got, err := d.Direct(context.Background(), corners, emptyCatalog())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// C1: [bgm(start), speech, bgm(stop)], C2: [speech]
	if len(got.Segments) != 4 {
		t.Fatalf("Segments: got %d, want 4", len(got.Segments))
	}
	// After C1 (bgm stop at index 2), C2 should just be speech
	if got.Segments[3].Type != model.SegmentTypeSpeech {
		t.Errorf("Segment[3]: want speech (C2), got %+v", got.Segments[3])
	}
}

func TestLLMDirector_Direct_InsertionAfterLastLine(t *testing.T) {
	mc := &mockClient{
		response: json.RawMessage(`{"insertions":[{"corner_index":0,"after_line_index":1,"type":"se","asset_name":"transition"}]}`),
	}
	d := direct.NewLLMDirector(mc, "{{corners}}", 0)

	corners := oneCorner("C1",
		model.Line{SpeakerRole: "host", Text: "A"},
		model.Line{SpeakerRole: "guest", Text: "B"},
	)
	catalog := model.AssetCatalog{
		SE: []model.AssetCatalogEntry{{Name: "transition"}},
	}

	got, err := d.Direct(context.Background(), corners, catalog)
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
	d := direct.NewLLMDirector(mc, "{{corners}}", 0)

	corners := oneCorner("C1",
		model.Line{SpeakerRole: "zundamon", Style: "なみだめ", Text: "ぐすん"},
		model.Line{SpeakerRole: "metan", Style: "", Text: "大丈夫？"},
	)

	got, err := d.Direct(context.Background(), corners, emptyCatalog())
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
	d := direct.NewLLMDirector(mc, "{{corners}}", 0)

	_, err := d.Direct(context.Background(), nil, emptyCatalog())
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestLLMDirector_Direct_SpeechSegmentFields(t *testing.T) {
	mc := &mockClient{
		response: json.RawMessage(`{"insertions":[]}`),
	}
	d := direct.NewLLMDirector(mc, "{{corners}}", 0)

	corners := oneCorner("C1", model.Line{SpeakerRole: "host", Text: "テストテキスト"})

	got, err := d.Direct(context.Background(), corners, emptyCatalog())
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
		SE: []model.AssetCatalogEntry{{Name: "chime", Description: "コーナー開始時のチャイム音"}},
	}

	_, err := d.Direct(context.Background(), []model.CornerLines{}, catalog)
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
		SE: []model.AssetCatalogEntry{{Name: "chime", Description: "テスト"}},
	}

	_, err := d.Direct(context.Background(), []model.CornerLines{}, catalog)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	for _, field := range []string{`"file"`, `"volume"`, `"duck_ratio"`, `"loop"`, `"fade_in"`, `"fade_out"`} {
		if strings.Contains(capturedPrompt, field) {
			t.Errorf("internal config field %s should not appear in prompt, got: %s", field, capturedPrompt)
		}
	}
}

func TestBuildScript_CopiesPresetFields(t *testing.T) {
	mc := &mockClient{
		response: json.RawMessage(`{"insertions":[]}`),
	}
	d := direct.NewLLMDirector(mc, "{{corners}}", 0)

	corners := oneCorner("C1",
		model.Line{SpeakerRole: "host", Text: "テスト", Intonation: "表現豊か", Pitch: "高め", Speed: "早口"},
		model.Line{SpeakerRole: "guest", Text: "応答"},
	)

	got, err := d.Direct(context.Background(), corners, emptyCatalog())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(got.Segments) != 2 {
		t.Fatalf("Segments: got %d, want 2", len(got.Segments))
	}

	seg0 := got.Segments[0]
	if seg0.Intonation != "表現豊か" {
		t.Errorf("Segment[0].Intonation: got %q, want 表現豊か", seg0.Intonation)
	}
	if seg0.Pitch != "高め" {
		t.Errorf("Segment[0].Pitch: got %q, want 高め", seg0.Pitch)
	}
	if seg0.Speed != "早口" {
		t.Errorf("Segment[0].Speed: got %q, want 早口", seg0.Speed)
	}

	seg1 := got.Segments[1]
	if seg1.Intonation != "" {
		t.Errorf("Segment[1].Intonation: got %q, want empty", seg1.Intonation)
	}
	if seg1.Pitch != "" {
		t.Errorf("Segment[1].Pitch: got %q, want empty", seg1.Pitch)
	}
	if seg1.Speed != "" {
		t.Errorf("Segment[1].Speed: got %q, want empty", seg1.Speed)
	}
}

func TestLLMDirector_Direct_WithPauseInsertion(t *testing.T) {
	mc := &mockClient{
		response: json.RawMessage(`{"insertions":[],"pause_insertions":[{"corner_index":0,"after_line_index":0,"duration_sec":1.2,"reason":"オチの前の溜め"}]}`),
	}
	d := direct.NewLLMDirector(mc, "{{corners}}", 0)

	corners := oneCorner("C1",
		model.Line{SpeakerRole: "host", Text: "オチの前"},
		model.Line{SpeakerRole: "guest", Text: "オチ"},
	)

	got, err := d.Direct(context.Background(), corners, emptyCatalog())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// [speech(host), pause(1.2s), speech(guest)]
	if len(got.Segments) != 3 {
		t.Fatalf("Segments: got %d, want 3", len(got.Segments))
	}
	if got.Segments[0].Type != model.SegmentTypeSpeech {
		t.Errorf("Segment[0].Type: got %q, want speech", got.Segments[0].Type)
	}
	if got.Segments[1].Type != model.SegmentTypePause {
		t.Errorf("Segment[1].Type: got %q, want pause", got.Segments[1].Type)
	}
	if got.Segments[1].DurationSec != 1.2 {
		t.Errorf("Segment[1].DurationSec: got %v, want 1.2", got.Segments[1].DurationSec)
	}
	if got.Segments[2].Type != model.SegmentTypeSpeech {
		t.Errorf("Segment[2].Type: got %q, want speech", got.Segments[2].Type)
	}
}

func TestLLMDirector_Direct_SEAndPauseAtSameIndex_SEFirst(t *testing.T) {
	mc := &mockClient{
		response: json.RawMessage(`{"insertions":[{"corner_index":0,"after_line_index":0,"type":"se","asset_name":"chime"}],"pause_insertions":[{"corner_index":0,"after_line_index":0,"duration_sec":1.0}]}`),
	}
	d := direct.NewLLMDirector(mc, "{{corners}}", 0)

	corners := oneCorner("C1",
		model.Line{SpeakerRole: "host", Text: "A"},
		model.Line{SpeakerRole: "guest", Text: "B"},
	)

	got, err := d.Direct(context.Background(), corners, emptyCatalog())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// [speech, se, pause, speech] — SE before pause at same index
	if len(got.Segments) != 4 {
		t.Fatalf("Segments: got %d, want 4", len(got.Segments))
	}
	if got.Segments[1].Type != model.SegmentTypeSE {
		t.Errorf("Segment[1].Type: got %q, want se (SE should come before pause)", got.Segments[1].Type)
	}
	if got.Segments[2].Type != model.SegmentTypePause {
		t.Errorf("Segment[2].Type: got %q, want pause", got.Segments[2].Type)
	}
}

func TestLLMDirector_Direct_PauseZeroDurationIgnored(t *testing.T) {
	mc := &mockClient{
		response: json.RawMessage(`{"insertions":[],"pause_insertions":[{"corner_index":0,"after_line_index":0,"duration_sec":0}]}`),
	}
	d := direct.NewLLMDirector(mc, "{{corners}}", 0)

	corners := oneCorner("C1",
		model.Line{SpeakerRole: "host", Text: "A"},
		model.Line{SpeakerRole: "guest", Text: "B"},
	)

	got, err := d.Direct(context.Background(), corners, emptyCatalog())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got.Segments) != 2 {
		t.Fatalf("Segments: got %d, want 2 (zero-duration pause should be ignored)", len(got.Segments))
	}
}

func TestLLMDirector_Direct_PauseNegativeDurationIgnored(t *testing.T) {
	mc := &mockClient{
		response: json.RawMessage(`{"insertions":[],"pause_insertions":[{"corner_index":0,"after_line_index":0,"duration_sec":-1.0}]}`),
	}
	d := direct.NewLLMDirector(mc, "{{corners}}", 0)

	corners := oneCorner("C1",
		model.Line{SpeakerRole: "host", Text: "A"},
		model.Line{SpeakerRole: "guest", Text: "B"},
	)

	got, err := d.Direct(context.Background(), corners, emptyCatalog())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got.Segments) != 2 {
		t.Fatalf("Segments: got %d, want 2 (negative-duration pause should be ignored)", len(got.Segments))
	}
}

// TestLLMDirector_Direct_MultiCorner verifies that corner_index routes insertions to the right corner.
func TestLLMDirector_Direct_MultiCorner(t *testing.T) {
	mc := &mockClient{
		// Insert SE after line 0 of corner 1 (second corner)
		response: json.RawMessage(`{"insertions":[{"corner_index":1,"after_line_index":0,"type":"se","asset_name":"chime"}]}`),
	}
	d := direct.NewLLMDirector(mc, "{{corners}}", 0)

	corners := []model.CornerLines{
		{Title: "C1", Lines: []model.Line{
			{SpeakerRole: "host", Text: "コーナー1セリフ"},
		}},
		{Title: "C2", Lines: []model.Line{
			{SpeakerRole: "guest", Text: "コーナー2セリフ"},
		}},
	}

	got, err := d.Direct(context.Background(), corners, model.AssetCatalog{
		SE: []model.AssetCatalogEntry{{Name: "chime"}},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// [speech(C1), speech(C2), se(chime)]
	if len(got.Segments) != 3 {
		t.Fatalf("Segments: got %d, want 3", len(got.Segments))
	}
	if got.Segments[0].Text != "コーナー1セリフ" {
		t.Errorf("Segment[0] should be C1 speech, got %+v", got.Segments[0])
	}
	if got.Segments[1].Text != "コーナー2セリフ" {
		t.Errorf("Segment[1] should be C2 speech, got %+v", got.Segments[1])
	}
	if got.Segments[2].Type != model.SegmentTypeSE || got.Segments[2].AssetName != "chime" {
		t.Errorf("Segment[2] should be SE chime, got %+v", got.Segments[2])
	}
}

// TestLLMDirector_Direct_DirectionInPrompt verifies corner direction appears in the prompt.
func TestLLMDirector_Direct_DirectionInPrompt(t *testing.T) {
	var capturedPrompt string
	mc := &capturingClient{
		response:       json.RawMessage(`{"insertions":[]}`),
		capturedPrompt: &capturedPrompt,
	}
	d := direct.NewLLMDirector(mc, "{{corners}}", 0)

	corners := []model.CornerLines{
		{
			Title:     "オープニング",
			Direction: "冒頭でオープニングジングルを流す。",
			Lines:     []model.Line{{SpeakerRole: "host", Text: "こんにちは"}},
		},
	}

	_, err := d.Direct(context.Background(), corners, emptyCatalog())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(capturedPrompt, "冒頭でオープニングジングルを流す。") {
		t.Errorf("direction value should appear in direct prompt, got: %s", capturedPrompt)
	}
}
