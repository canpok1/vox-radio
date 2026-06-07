package direct_test

import (
	"context"
	"encoding/json"
	"fmt"
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

// helper: director that returns no insertions (common setup for structural tests)
func noInsertionDirector() *direct.LLMDirector {
	mc := &mockClient{response: json.RawMessage(`{"insertions":[]}`)}
	return direct.NewLLMDirector(mc, "{{corners}}", 0)
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

	got, err := d.Direct(context.Background(), corners, catalog, "")
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

	got, err := d.Direct(context.Background(), corners, catalog, "")
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

	got, err := d.Direct(context.Background(), corners, emptyCatalog(), "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// [bgm(start), speech] — single/last corner: no trailing BGM stop
	if len(got.Segments) != 2 {
		t.Fatalf("Segments: got %d, want 2", len(got.Segments))
	}
	if got.Segments[0].Type != model.SegmentTypeBGM || got.Segments[0].AssetName != "talk_bgm" {
		t.Errorf("Segment[0]: got %+v, want bgm(talk_bgm)", got.Segments[0])
	}
	if got.Segments[1].Type != model.SegmentTypeSpeech {
		t.Errorf("Segment[1]: got %+v, want speech", got.Segments[1])
	}
}

func TestLLMDirector_Direct_StartAudio_Jingle_PrependedFirst(t *testing.T) {
	mc := &mockClient{
		response: json.RawMessage(`{"insertions":[]}`),
	}
	d := direct.NewLLMDirector(mc, "{{corners}}", 0)

	corners := []model.CornerLines{{
		Title:      "C1",
		StartAudio: &model.CornerAudio{Type: model.SegmentTypeJingle, AssetName: "opening"},
		Lines:      []model.Line{{SpeakerRole: "host", Text: "話す"}},
	}}

	got, err := d.Direct(context.Background(), corners, emptyCatalog(), "")
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

func TestLLMDirector_Direct_EndAudio_Jingle_AppendedLast(t *testing.T) {
	mc := &mockClient{
		response: json.RawMessage(`{"insertions":[]}`),
	}
	d := direct.NewLLMDirector(mc, "{{corners}}", 0)

	corners := []model.CornerLines{{
		Title:    "C1",
		EndAudio: &model.CornerAudio{Type: model.SegmentTypeJingle, AssetName: "ending"},
		Lines:    []model.Line{{SpeakerRole: "host", Text: "話す"}},
	}}

	got, err := d.Direct(context.Background(), corners, emptyCatalog(), "")
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

func TestLLMDirector_Direct_CornerAllAssets_Jingle_CorrectOrder(t *testing.T) {
	mc := &mockClient{
		response: json.RawMessage(`{"insertions":[]}`),
	}
	d := direct.NewLLMDirector(mc, "{{corners}}", 0)

	corners := []model.CornerLines{{
		Title:      "C1",
		StartAudio: &model.CornerAudio{Type: model.SegmentTypeJingle, AssetName: "op"},
		BGM:        "bgm1",
		EndAudio:   &model.CornerAudio{Type: model.SegmentTypeJingle, AssetName: "ed"},
		Lines:      []model.Line{{SpeakerRole: "host", Text: "話す"}},
	}}

	got, err := d.Direct(context.Background(), corners, emptyCatalog(), "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// [jingle(op), bgm(start), speech, jingle(ed)] — no BGM stop before EndJingle (filter handles it)
	if len(got.Segments) != 4 {
		t.Fatalf("Segments: got %d, want 4", len(got.Segments))
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
	if got.Segments[3].Type != model.SegmentTypeJingle || got.Segments[3].AssetName != "ed" {
		t.Errorf("Segment[3]: want jingle(ed), got %+v", got.Segments[3])
	}
}

func TestLLMDirector_Direct_StartAudio_SE_AfterBGM_BGMContinues(t *testing.T) {
	mc := &mockClient{
		response: json.RawMessage(`{"insertions":[]}`),
	}
	d := direct.NewLLMDirector(mc, "{{corners}}", 0)

	// type:se → BGM先行 → SE（activeBGM維持）
	corners := []model.CornerLines{
		{
			Title:      "C1",
			StartAudio: &model.CornerAudio{Type: model.SegmentTypeSE, AssetName: "chime"},
			BGM:        "talk_bgm",
			Lines:      []model.Line{{SpeakerRole: "host", Text: "話す"}},
		},
		{
			Title: "C2",
			BGM:   "talk_bgm",
			Lines: []model.Line{{SpeakerRole: "host", Text: "続き"}},
		},
	}

	got, err := d.Direct(context.Background(), corners, emptyCatalog(), "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// C1: [bgm(talk_bgm), se(chime), speech]
	// C2: BGMが維持されているので bgm再開なし → [speech]
	// 計 [bgm(talk_bgm), se(chime), speech, speech]
	if len(got.Segments) != 4 {
		t.Fatalf("Segments: got %d, want 4\nsegments: %+v", len(got.Segments), got.Segments)
	}
	if got.Segments[0].Type != model.SegmentTypeBGM || got.Segments[0].AssetName != "talk_bgm" {
		t.Errorf("Segment[0]: want bgm(talk_bgm), got %+v", got.Segments[0])
	}
	if got.Segments[1].Type != model.SegmentTypeSE || got.Segments[1].AssetName != "chime" {
		t.Errorf("Segment[1]: want se(chime), got %+v", got.Segments[1])
	}
	if got.Segments[2].Type != model.SegmentTypeSpeech {
		t.Errorf("Segment[2]: want speech, got %+v", got.Segments[2])
	}
	if got.Segments[3].Type != model.SegmentTypeSpeech {
		t.Errorf("Segment[3]: want speech, got %+v", got.Segments[3])
	}
}

func TestLLMDirector_Direct_EndAudio_SE_BGMContinues(t *testing.T) {
	mc := &mockClient{
		response: json.RawMessage(`{"insertions":[]}`),
	}
	d := direct.NewLLMDirector(mc, "{{corners}}", 0)

	// type:se の end_audio → SE後もactiveBGM維持
	corners := []model.CornerLines{
		{
			Title:    "C1",
			BGM:      "talk_bgm",
			EndAudio: &model.CornerAudio{Type: model.SegmentTypeSE, AssetName: "chime"},
			Lines:    []model.Line{{SpeakerRole: "host", Text: "話す"}},
		},
		{
			Title: "C2",
			BGM:   "talk_bgm",
			Lines: []model.Line{{SpeakerRole: "host", Text: "続き"}},
		},
	}

	got, err := d.Direct(context.Background(), corners, emptyCatalog(), "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// C1: [bgm(talk_bgm), speech, se(chime)]
	// C2: BGMが維持されているので bgm再開なし → [speech]
	// 計 [bgm(talk_bgm), speech, se(chime), speech]
	if len(got.Segments) != 4 {
		t.Fatalf("Segments: got %d, want 4\nsegments: %+v", len(got.Segments), got.Segments)
	}
	if got.Segments[0].Type != model.SegmentTypeBGM || got.Segments[0].AssetName != "talk_bgm" {
		t.Errorf("Segment[0]: want bgm(talk_bgm), got %+v", got.Segments[0])
	}
	if got.Segments[1].Type != model.SegmentTypeSpeech {
		t.Errorf("Segment[1]: want speech, got %+v", got.Segments[1])
	}
	if got.Segments[2].Type != model.SegmentTypeSE || got.Segments[2].AssetName != "chime" {
		t.Errorf("Segment[2]: want se(chime), got %+v", got.Segments[2])
	}
	if got.Segments[3].Type != model.SegmentTypeSpeech {
		t.Errorf("Segment[3]: want speech, got %+v", got.Segments[3])
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

	got, err := d.Direct(context.Background(), corners, emptyCatalog(), "")
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
		Title:      "C1",
		StartAudio: &model.CornerAudio{Type: model.SegmentTypeJingle, AssetName: "opening"},
		EndAudio:   &model.CornerAudio{Type: model.SegmentTypeJingle, AssetName: "ending"},
		BGM:        "talk_bgm",
		Lines:      []model.Line{{SpeakerRole: "host", Text: "hello"}},
	}}

	_, err := d.Direct(context.Background(), corners, emptyCatalog(), "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	for _, key := range []string{"start_audio", "end_audio", "bgm"} {
		if strings.Contains(capturedPrompt, `"`+key+`"`) {
			t.Errorf("field %q should not appear in LLM payload, got: %s", key, capturedPrompt)
		}
	}
}

func TestLLMDirector_Direct_BGMDoesNotLeakToNextCorner(t *testing.T) {
	d := noInsertionDirector()

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

	got, err := d.Direct(context.Background(), corners, emptyCatalog(), "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// [bgm(talk_bgm), speech_c1, bgm("" stop), speech_c2] — stop emitted at C2 entry
	if len(got.Segments) != 4 {
		t.Fatalf("Segments: got %d, want 4", len(got.Segments))
	}
	if got.Segments[2].Type != model.SegmentTypeBGM || got.Segments[2].AssetName != "" {
		t.Errorf("Segment[2]: want bgm(stop), got %+v", got.Segments[2])
	}
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

	got, err := d.Direct(context.Background(), corners, catalog, "")
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

	got, err := d.Direct(context.Background(), corners, emptyCatalog(), "")
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

	_, err := d.Direct(context.Background(), nil, emptyCatalog(), "")
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

	got, err := d.Direct(context.Background(), corners, emptyCatalog(), "")
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

func TestBuildScript_StartPauseSec_InjectedBeforeStartAudio(t *testing.T) {
	d := noInsertionDirector()

	corners := []model.CornerLines{{
		Title:         "C1",
		StartAudio:    &model.CornerAudio{Type: model.SegmentTypeJingle, AssetName: "opening"},
		StartPauseSec: 1.0,
		Lines:         []model.Line{{SpeakerRole: "host", Text: "話す"}},
	}}

	got, err := d.Direct(context.Background(), corners, emptyCatalog(), "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// [pause(1.0), jingle(opening), speech]
	if len(got.Segments) != 3 {
		t.Fatalf("Segments: got %d, want 3", len(got.Segments))
	}
	if got.Segments[0].Type != model.SegmentTypePause {
		t.Errorf("Segment[0].Type: got %q, want pause", got.Segments[0].Type)
	}
	if got.Segments[0].DurationSec != 1.0 {
		t.Errorf("Segment[0].DurationSec: got %f, want 1.0", got.Segments[0].DurationSec)
	}
	if got.Segments[1].Type != model.SegmentTypeJingle {
		t.Errorf("Segment[1].Type: got %q, want jingle", got.Segments[1].Type)
	}
}

func TestBuildScript_EndPauseSec_InjectedAfterEndAudio(t *testing.T) {
	d := noInsertionDirector()

	corners := []model.CornerLines{{
		Title:       "C1",
		EndAudio:    &model.CornerAudio{Type: model.SegmentTypeJingle, AssetName: "ending"},
		EndPauseSec: 2.0,
		Lines:       []model.Line{{SpeakerRole: "host", Text: "話す"}},
	}}

	got, err := d.Direct(context.Background(), corners, emptyCatalog(), "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// [speech, jingle(ending), pause(2.0)]
	if len(got.Segments) != 3 {
		t.Fatalf("Segments: got %d, want 3", len(got.Segments))
	}
	if got.Segments[1].Type != model.SegmentTypeJingle {
		t.Errorf("Segment[1].Type: got %q, want jingle", got.Segments[1].Type)
	}
	if got.Segments[2].Type != model.SegmentTypePause {
		t.Errorf("Segment[2].Type: got %q, want pause", got.Segments[2].Type)
	}
	if got.Segments[2].DurationSec != 2.0 {
		t.Errorf("Segment[2].DurationSec: got %f, want 2.0", got.Segments[2].DurationSec)
	}
}

func TestBuildScript_ZeroStartPauseSec_NoInjection(t *testing.T) {
	d := noInsertionDirector()

	corners := []model.CornerLines{{
		Title:         "C1",
		StartPauseSec: 0,
		Lines:         []model.Line{{SpeakerRole: "host", Text: "話す"}},
	}}

	got, err := d.Direct(context.Background(), corners, emptyCatalog(), "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// [speech] — no pause injected when StartPauseSec == 0
	if len(got.Segments) != 1 {
		t.Fatalf("Segments: got %d, want 1", len(got.Segments))
	}
	if got.Segments[0].Type != model.SegmentTypeSpeech {
		t.Errorf("Segment[0].Type: got %q, want speech", got.Segments[0].Type)
	}
}

func TestBuildScript_ZeroEndPauseSec_NoInjection(t *testing.T) {
	d := noInsertionDirector()

	corners := []model.CornerLines{{
		Title:       "C1",
		EndPauseSec: 0,
		Lines:       []model.Line{{SpeakerRole: "host", Text: "話す"}},
	}}

	got, err := d.Direct(context.Background(), corners, emptyCatalog(), "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// [speech] — no pause injected when EndPauseSec == 0
	if len(got.Segments) != 1 {
		t.Fatalf("Segments: got %d, want 1", len(got.Segments))
	}
	if got.Segments[0].Type != model.SegmentTypeSpeech {
		t.Errorf("Segment[0].Type: got %q, want speech", got.Segments[0].Type)
	}
}

func TestBuildScript_AdjacentCorners_BothPausesInjected(t *testing.T) {
	d := noInsertionDirector()

	corners := []model.CornerLines{
		{
			Title:       "C1",
			EndPauseSec: 1.0,
			Lines:       []model.Line{{SpeakerRole: "host", Text: "コーナー1"}},
		},
		{
			Title:         "C2",
			StartPauseSec: 0.5,
			Lines:         []model.Line{{SpeakerRole: "host", Text: "コーナー2"}},
		},
	}

	got, err := d.Direct(context.Background(), corners, emptyCatalog(), "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// [speech_c1, pause(1.0), pause(0.5), speech_c2]
	if len(got.Segments) != 4 {
		t.Fatalf("Segments: got %d, want 4", len(got.Segments))
	}
	if got.Segments[1].Type != model.SegmentTypePause || got.Segments[1].DurationSec != 1.0 {
		t.Errorf("Segment[1]: want pause(1.0), got %+v", got.Segments[1])
	}
	if got.Segments[2].Type != model.SegmentTypePause || got.Segments[2].DurationSec != 0.5 {
		t.Errorf("Segment[2]: want pause(0.5), got %+v", got.Segments[2])
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

	_, err := d.Direct(context.Background(), []model.CornerLines{}, catalog, "")
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

	_, err := d.Direct(context.Background(), []model.CornerLines{}, catalog, "")
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

	got, err := d.Direct(context.Background(), corners, emptyCatalog(), "")
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

	got, err := d.Direct(context.Background(), corners, emptyCatalog(), "")
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

	got, err := d.Direct(context.Background(), corners, emptyCatalog(), "")
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

	got, err := d.Direct(context.Background(), corners, emptyCatalog(), "")
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

	got, err := d.Direct(context.Background(), corners, emptyCatalog(), "")
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
	}, "")
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

// TestBuildScript_SameBGM_ContinuousPlay verifies that consecutive corners with the same BGM
// do not insert stop/start segments between them (continuous playback).
func TestBuildScript_SameBGM_ContinuousPlay(t *testing.T) {
	d := noInsertionDirector()

	corners := []model.CornerLines{
		{Title: "C1", BGM: "talk_bgm", Lines: []model.Line{{SpeakerRole: "host", Text: "コーナー1"}}},
		{Title: "C2", BGM: "talk_bgm", Lines: []model.Line{{SpeakerRole: "host", Text: "コーナー2"}}},
	}

	got, err := d.Direct(context.Background(), corners, emptyCatalog(), "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// [bgm(talk_bgm), speech_c1, speech_c2] — no stop/start between corners
	if len(got.Segments) != 3 {
		t.Fatalf("Segments: got %d, want 3 (no BGM stop/start between same-BGM corners)\nsegments: %+v", len(got.Segments), got.Segments)
	}
	if got.Segments[0].Type != model.SegmentTypeBGM || got.Segments[0].AssetName != "talk_bgm" {
		t.Errorf("Segment[0]: want bgm(talk_bgm), got %+v", got.Segments[0])
	}
	if got.Segments[1].Type != model.SegmentTypeSpeech {
		t.Errorf("Segment[1]: want speech, got %+v", got.Segments[1])
	}
	if got.Segments[2].Type != model.SegmentTypeSpeech {
		t.Errorf("Segment[2]: want speech, got %+v", got.Segments[2])
	}
}

// TestBuildScript_DifferentBGM_SeamlessSwitch verifies that adjacent corners with different BGMs
// switch without a stop segment (no silent gap).
func TestBuildScript_DifferentBGM_SeamlessSwitch(t *testing.T) {
	d := noInsertionDirector()

	corners := []model.CornerLines{
		{Title: "C1", BGM: "bgm_a", Lines: []model.Line{{SpeakerRole: "host", Text: "コーナー1"}}},
		{Title: "C2", BGM: "bgm_b", Lines: []model.Line{{SpeakerRole: "host", Text: "コーナー2"}}},
	}

	got, err := d.Direct(context.Background(), corners, emptyCatalog(), "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// [bgm(a), speech_c1, bgm(b), speech_c2] — no stop, seamless switch
	if len(got.Segments) != 4 {
		t.Fatalf("Segments: got %d, want 4 (BGM(a), sp1, BGM(b), sp2)\nsegments: %+v", len(got.Segments), got.Segments)
	}
	if got.Segments[0].Type != model.SegmentTypeBGM || got.Segments[0].AssetName != "bgm_a" {
		t.Errorf("Segment[0]: want bgm(bgm_a), got %+v", got.Segments[0])
	}
	if got.Segments[1].Type != model.SegmentTypeSpeech {
		t.Errorf("Segment[1]: want speech, got %+v", got.Segments[1])
	}
	if got.Segments[2].Type != model.SegmentTypeBGM || got.Segments[2].AssetName != "bgm_b" {
		t.Errorf("Segment[2]: want bgm(bgm_b), got %+v", got.Segments[2])
	}
	if got.Segments[3].Type != model.SegmentTypeSpeech {
		t.Errorf("Segment[3]: want speech, got %+v", got.Segments[3])
	}
}

// TestBuildScript_BGMToNoBGM_StopAtBoundary verifies that when a BGM corner is followed by
// a no-BGM corner, the stop segment is inserted at the beginning of the second corner.
func TestBuildScript_BGMToNoBGM_StopAtBoundary(t *testing.T) {
	d := noInsertionDirector()

	corners := []model.CornerLines{
		{Title: "C1", BGM: "talk_bgm", Lines: []model.Line{{SpeakerRole: "host", Text: "コーナー1"}}},
		{Title: "C2", Lines: []model.Line{{SpeakerRole: "host", Text: "コーナー2"}}},
	}

	got, err := d.Direct(context.Background(), corners, emptyCatalog(), "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// [bgm(start), speech_c1, bgm(""), speech_c2] — stop at C2 entry
	if len(got.Segments) != 4 {
		t.Fatalf("Segments: got %d, want 4\nsegments: %+v", len(got.Segments), got.Segments)
	}
	if got.Segments[0].Type != model.SegmentTypeBGM || got.Segments[0].AssetName != "talk_bgm" {
		t.Errorf("Segment[0]: want bgm(talk_bgm), got %+v", got.Segments[0])
	}
	if got.Segments[1].Type != model.SegmentTypeSpeech {
		t.Errorf("Segment[1]: want speech(C1), got %+v", got.Segments[1])
	}
	if got.Segments[2].Type != model.SegmentTypeBGM || got.Segments[2].AssetName != "" {
		t.Errorf("Segment[2]: want bgm(stop), got %+v", got.Segments[2])
	}
	if got.Segments[3].Type != model.SegmentTypeSpeech {
		t.Errorf("Segment[3]: want speech(C2), got %+v", got.Segments[3])
	}
}

// TestBuildScript_LastCorner_NoBGMStop verifies that the last corner with BGM
// does not get a trailing BGM stop segment.
func TestBuildScript_LastCorner_NoBGMStop(t *testing.T) {
	d := noInsertionDirector()

	corners := []model.CornerLines{
		{Title: "C1", BGM: "talk_bgm", Lines: []model.Line{{SpeakerRole: "host", Text: "コーナー1"}}},
	}

	got, err := d.Direct(context.Background(), corners, emptyCatalog(), "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// [bgm(talk_bgm), speech] — no trailing BGM stop
	if len(got.Segments) != 2 {
		t.Fatalf("Segments: got %d, want 2 (no trailing BGM stop)\nsegments: %+v", len(got.Segments), got.Segments)
	}
	if got.Segments[0].Type != model.SegmentTypeBGM || got.Segments[0].AssetName != "talk_bgm" {
		t.Errorf("Segment[0]: want bgm(talk_bgm), got %+v", got.Segments[0])
	}
	if got.Segments[1].Type != model.SegmentTypeSpeech {
		t.Errorf("Segment[1]: want speech, got %+v", got.Segments[1])
	}
}

// TestBuildScript_EndAudio_Jingle_ResetsBGM verifies that an EndAudio with type:jingle resets activeBGM,
// causing the same BGM to restart after the jingle.
func TestBuildScript_EndAudio_Jingle_ResetsBGM(t *testing.T) {
	d := noInsertionDirector()

	corners := []model.CornerLines{
		{Title: "C1", BGM: "talk_bgm", EndAudio: &model.CornerAudio{Type: model.SegmentTypeJingle, AssetName: "jingle"}, Lines: []model.Line{{SpeakerRole: "host", Text: "コーナー1"}}},
		{Title: "C2", BGM: "talk_bgm", Lines: []model.Line{{SpeakerRole: "host", Text: "コーナー2"}}},
	}

	got, err := d.Direct(context.Background(), corners, emptyCatalog(), "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// [bgm(talk_bgm), speech_c1, jingle, bgm(talk_bgm), speech_c2]
	// EndJingle resets activeBGM → C2 must re-emit BGM start
	if len(got.Segments) != 5 {
		t.Fatalf("Segments: got %d, want 5\nsegments: %+v", len(got.Segments), got.Segments)
	}
	if got.Segments[0].Type != model.SegmentTypeBGM || got.Segments[0].AssetName != "talk_bgm" {
		t.Errorf("Segment[0]: want bgm(talk_bgm), got %+v", got.Segments[0])
	}
	if got.Segments[1].Type != model.SegmentTypeSpeech {
		t.Errorf("Segment[1]: want speech(C1), got %+v", got.Segments[1])
	}
	if got.Segments[2].Type != model.SegmentTypeJingle || got.Segments[2].AssetName != "jingle" {
		t.Errorf("Segment[2]: want jingle(jingle), got %+v", got.Segments[2])
	}
	if got.Segments[3].Type != model.SegmentTypeBGM || got.Segments[3].AssetName != "talk_bgm" {
		t.Errorf("Segment[3]: want bgm(talk_bgm) restart after jingle, got %+v", got.Segments[3])
	}
	if got.Segments[4].Type != model.SegmentTypeSpeech {
		t.Errorf("Segment[4]: want speech(C2), got %+v", got.Segments[4])
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

	_, err := d.Direct(context.Background(), corners, emptyCatalog(), "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(capturedPrompt, "冒頭でオープニングジングルを流す。") {
		t.Errorf("direction value should appear in direct prompt, got: %s", capturedPrompt)
	}
}

func TestLLMDirector_Direct_LineConversionApplied(t *testing.T) {
	mc := &mockClient{
		response: json.RawMessage(`{"insertions":[],"line_conversions":[{"corner_index":0,"line_index":0,"text":"えーあい"},{"corner_index":0,"line_index":1,"text":"りどみーどっとえむでぃー"}]}`),
	}
	d := direct.NewLLMDirector(mc, "{{corners}}", 0)

	corners := oneCorner("C1",
		model.Line{SpeakerRole: "host", Text: "AI"},
		model.Line{SpeakerRole: "guest", Text: "README.md"},
	)

	got, err := d.Direct(context.Background(), corners, emptyCatalog(), "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got.Segments) != 2 {
		t.Fatalf("Segments: got %d, want 2", len(got.Segments))
	}
	if got.Segments[0].Text != "えーあい" {
		t.Errorf("Segment[0].Text: got %q, want えーあい (converted)", got.Segments[0].Text)
	}
	if got.Segments[1].Text != "りどみーどっとえむでぃー" {
		t.Errorf("Segment[1].Text: got %q, want りどみーどっとえむでぃー (converted)", got.Segments[1].Text)
	}
}

func TestLLMDirector_Direct_LineConversionFallback_MissingEntry(t *testing.T) {
	mc := &mockClient{
		// line_conversions has only line 0; line 1 is missing → fallback to original
		response: json.RawMessage(`{"insertions":[],"line_conversions":[{"corner_index":0,"line_index":0,"text":"えーあい"}]}`),
	}
	d := direct.NewLLMDirector(mc, "{{corners}}", 0)

	corners := oneCorner("C1",
		model.Line{SpeakerRole: "host", Text: "AI"},
		model.Line{SpeakerRole: "guest", Text: "README.md"},
	)

	got, err := d.Direct(context.Background(), corners, emptyCatalog(), "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got.Segments) != 2 {
		t.Fatalf("Segments: got %d, want 2", len(got.Segments))
	}
	if got.Segments[0].Text != "えーあい" {
		t.Errorf("Segment[0].Text: got %q, want えーあい (converted)", got.Segments[0].Text)
	}
	if got.Segments[1].Text != "README.md" {
		t.Errorf("Segment[1].Text: got %q, want README.md (original fallback)", got.Segments[1].Text)
	}
}

func TestLLMDirector_Direct_LineConversionFallback_EmptyConversions(t *testing.T) {
	mc := &mockClient{
		response: json.RawMessage(`{"insertions":[],"line_conversions":[]}`),
	}
	d := direct.NewLLMDirector(mc, "{{corners}}", 0)

	corners := oneCorner("C1",
		model.Line{SpeakerRole: "host", Text: "AI"},
	)

	got, err := d.Direct(context.Background(), corners, emptyCatalog(), "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got.Segments) != 1 {
		t.Fatalf("Segments: got %d, want 1", len(got.Segments))
	}
	if got.Segments[0].Text != "AI" {
		t.Errorf("Segment[0].Text: got %q, want AI (original fallback)", got.Segments[0].Text)
	}
}

func TestLLMDirector_Direct_LineConversionFallback_EmptyConvertedText(t *testing.T) {
	mc := &mockClient{
		// converted text is empty string → fallback to original
		response: json.RawMessage(`{"insertions":[],"line_conversions":[{"corner_index":0,"line_index":0,"text":""}]}`),
	}
	d := direct.NewLLMDirector(mc, "{{corners}}", 0)

	corners := oneCorner("C1",
		model.Line{SpeakerRole: "host", Text: "AI"},
	)

	got, err := d.Direct(context.Background(), corners, emptyCatalog(), "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Segments[0].Text != "AI" {
		t.Errorf("Segment[0].Text: got %q, want AI (empty converted text → original fallback)", got.Segments[0].Text)
	}
}

// sequentialClient returns different responses for each successive Complete call.
type sequentialClient struct {
	responses []json.RawMessage
	errs      []error
	callIdx   int
}

func (c *sequentialClient) Complete(_ context.Context, _ llm.CompletionRequest) (json.RawMessage, error) {
	i := c.callIdx
	c.callIdx++
	if i >= len(c.responses) {
		return nil, fmt.Errorf("unexpected call index %d", i)
	}
	var e error
	if i < len(c.errs) {
		e = c.errs[i]
	}
	return c.responses[i], e
}

// TestLLMDirector_Direct_WithProofread_CorrectsMisreading verifies that the proofreading pass
// applies corrections to the conversionMap, overriding the direct step's conversion.
// Reproduces the "頭突き" misread case: direct converts to "あたまつきするのだ",
// proofread corrects it to "ずつきするのだ".
func TestLLMDirector_Direct_WithProofread_CorrectsMisreading(t *testing.T) {
	mc := &sequentialClient{
		responses: []json.RawMessage{
			// Call 1: main direct LLM — returns wrong kana for 頭突き
			json.RawMessage(`{"insertions":[],"line_conversions":[{"corner_index":0,"line_index":0,"text":"あたまつきするのだ"}]}`),
			// Call 2: proofread LLM — corrects to the right reading
			json.RawMessage(`{"corrections":[{"corner_index":0,"line_index":0,"text":"ずつきするのだ","reason":"頭突き→ずつき（連濁の取りこぼし）"}]}`),
		},
		errs: []error{nil, nil},
	}
	d := direct.NewLLMDirector(mc, "{{corners}}", 0,
		direct.WithProofread("{{lines}}", 0),
	)

	corners := oneCorner("C1",
		model.Line{SpeakerRole: "host", Text: "頭突きするのだ"},
	)

	got, err := d.Direct(context.Background(), corners, emptyCatalog(), "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got.Segments) != 1 {
		t.Fatalf("Segments: got %d, want 1", len(got.Segments))
	}
	if got.Segments[0].Text != "ずつきするのだ" {
		t.Errorf("Segment[0].Text: got %q, want ずつきするのだ (proofread correction applied)", got.Segments[0].Text)
	}
}

// TestLLMDirector_Direct_WithProofread_FallbackOnLLMError verifies that when the proofread
// LLM call fails, the pipeline does not stop and falls back to the direct conversion.
func TestLLMDirector_Direct_WithProofread_FallbackOnLLMError(t *testing.T) {
	mc := &sequentialClient{
		responses: []json.RawMessage{
			// Call 1: main direct LLM — returns kana conversion
			json.RawMessage(`{"insertions":[],"line_conversions":[{"corner_index":0,"line_index":0,"text":"あたまつきするのだ"}]}`),
			// Call 2: proofread LLM — fails
			nil,
		},
		errs: []error{nil, fmt.Errorf("proofread llm error")},
	}
	d := direct.NewLLMDirector(mc, "{{corners}}", 0,
		direct.WithProofread("{{lines}}", 0),
	)

	corners := oneCorner("C1",
		model.Line{SpeakerRole: "host", Text: "頭突きするのだ"},
	)

	got, err := d.Direct(context.Background(), corners, emptyCatalog(), "")
	if err != nil {
		t.Fatalf("pipeline must not stop on proofread error: %v", err)
	}
	if len(got.Segments) != 1 {
		t.Fatalf("Segments: got %d, want 1", len(got.Segments))
	}
	// Falls back to direct's conversion (not original text)
	if got.Segments[0].Text != "あたまつきするのだ" {
		t.Errorf("Segment[0].Text: got %q, want あたまつきするのだ (fallback to direct conversion)", got.Segments[0].Text)
	}
}

// TestLLMDirector_Direct_WithProofread_SkipWhenNotSet verifies that when WithProofread is not
// used, only one LLM call is made and the direct conversion result is used unchanged.
func TestLLMDirector_Direct_WithProofread_SkipWhenNotSet(t *testing.T) {
	mc := &sequentialClient{
		responses: []json.RawMessage{
			// Only one call expected
			json.RawMessage(`{"insertions":[],"line_conversions":[{"corner_index":0,"line_index":0,"text":"ずつきするのだ"}]}`),
		},
		errs: []error{nil},
	}
	// No WithProofread option
	d := direct.NewLLMDirector(mc, "{{corners}}", 0)

	corners := oneCorner("C1",
		model.Line{SpeakerRole: "host", Text: "頭突きするのだ"},
	)

	got, err := d.Direct(context.Background(), corners, emptyCatalog(), "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got.Segments) != 1 {
		t.Fatalf("Segments: got %d, want 1", len(got.Segments))
	}
	if got.Segments[0].Text != "ずつきするのだ" {
		t.Errorf("Segment[0].Text: got %q, want ずつきするのだ (direct conversion used, no proofreading)", got.Segments[0].Text)
	}
	// Verify only one LLM call was made
	if mc.callIdx != 1 {
		t.Errorf("LLM call count: got %d, want 1 (no proofread call when WithProofread not set)", mc.callIdx)
	}
}

// TestLLMDirector_Direct_WithProofread_EmptyCorrections verifies that when proofread returns
// an empty corrections list, the direct conversion is used unchanged.
func TestLLMDirector_Direct_WithProofread_EmptyCorrections(t *testing.T) {
	mc := &sequentialClient{
		responses: []json.RawMessage{
			json.RawMessage(`{"insertions":[],"line_conversions":[{"corner_index":0,"line_index":0,"text":"てすとてきすと"}]}`),
			json.RawMessage(`{"corrections":[]}`),
		},
		errs: []error{nil, nil},
	}
	d := direct.NewLLMDirector(mc, "{{corners}}", 0,
		direct.WithProofread("{{lines}}", 0),
	)

	corners := oneCorner("C1",
		model.Line{SpeakerRole: "host", Text: "テストテキスト"},
	)

	got, err := d.Direct(context.Background(), corners, emptyCatalog(), "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got.Segments) != 1 {
		t.Fatalf("Segments: got %d, want 1", len(got.Segments))
	}
	if got.Segments[0].Text != "てすとてきすと" {
		t.Errorf("Segment[0].Text: got %q, want てすとてきすと (direct conversion unchanged when no corrections)", got.Segments[0].Text)
	}
}

func TestLLMDirector_Direct_ProgramDirectionInPrompt(t *testing.T) {
	var capturedPrompt string
	mc := &capturingClient{
		response:       json.RawMessage(`{"insertions":[]}`),
		capturedPrompt: &capturedPrompt,
	}
	d := direct.NewLLMDirector(mc, "{{program_direction}}", 0)

	_, err := d.Direct(context.Background(), []model.CornerLines{}, emptyCatalog(), "番組全体の演出方針")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(capturedPrompt, "番組全体の演出方針") {
		t.Errorf("program_direction should appear in prompt, got: %s", capturedPrompt)
	}
}

func TestLLMDirector_Direct_ProgramDirectionEmptyUsesNone(t *testing.T) {
	var capturedPrompt string
	mc := &capturingClient{
		response:       json.RawMessage(`{"insertions":[]}`),
		capturedPrompt: &capturedPrompt,
	}
	d := direct.NewLLMDirector(mc, "{{program_direction}}", 0)

	_, err := d.Direct(context.Background(), []model.CornerLines{}, emptyCatalog(), "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(capturedPrompt, "（なし）") {
		t.Errorf("empty program_direction should be rendered as （なし）, got: %s", capturedPrompt)
	}
}
