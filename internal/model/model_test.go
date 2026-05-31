package model_test

import (
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/canpok1/vox-radio/internal/model"
)

func loadFixture(t *testing.T, name string) []byte {
	t.Helper()
	b, err := os.ReadFile("testdata/" + name)
	if err != nil {
		t.Fatalf("failed to read fixture %s: %v", name, err)
	}
	return b
}

func unmarshalFixture[T any](t *testing.T, name string) T {
	t.Helper()
	var v T
	if err := json.Unmarshal(loadFixture(t, name), &v); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	return v
}

func roundTrip[T any](t *testing.T, data []byte) {
	t.Helper()
	var v T
	if err := json.Unmarshal(data, &v); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	out, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}
	var v2 T
	if err := json.Unmarshal(out, &v2); err != nil {
		t.Fatalf("second unmarshal failed: %v", err)
	}
}

func TestArticles_RoundTrip(t *testing.T) {
	roundTrip[model.Articles](t, loadFixture(t, "articles.json"))
}

func TestArticles_Fields(t *testing.T) {
	v := unmarshalFixture[model.Articles](t, "articles.json")
	if len(v.Corners) == 0 {
		t.Error("expected at least one corner")
	}
	ca := v.Corners[0]
	if ca.CornerTitle == "" {
		t.Error("CornerTitle must not be empty")
	}
	if len(ca.Articles) == 0 {
		t.Error("expected at least one article in corner")
	}
	a := ca.Articles[0]
	if a.URL == "" {
		t.Error("URL must not be empty")
	}
	if a.Title == "" {
		t.Error("Title must not be empty")
	}
	if a.Body == "" {
		t.Error("Body must not be empty")
	}
}

func TestArticles_CornerMap(t *testing.T) {
	art1 := model.Article{URL: "https://example.com/1", Title: "T1", Body: "B1"}
	art2 := model.Article{URL: "https://example.com/2", Title: "T2", Body: "B2"}
	a := model.Articles{
		Corners: []model.CornerArticles{
			{CornerTitle: "ニュース", Articles: []model.Article{art1}},
			{CornerTitle: "エンディング", Articles: []model.Article{art2}},
		},
	}

	m := a.CornerMap()

	if len(m) != 2 {
		t.Fatalf("map length: got %d, want 2", len(m))
	}
	if arts := m["ニュース"]; len(arts) != 1 || arts[0].URL != art1.URL {
		t.Errorf("CornerMap[\"ニュース\"]: got %v, want [%v]", arts, art1)
	}
	if arts := m["エンディング"]; len(arts) != 1 || arts[0].URL != art2.URL {
		t.Errorf("CornerMap[\"エンディング\"]: got %v, want [%v]", arts, art2)
	}
	if _, ok := m["存在しないコーナー"]; ok {
		t.Error("missing key should not exist in map")
	}
}

func TestArticles_CornerMap_Empty(t *testing.T) {
	m := model.Articles{}.CornerMap()
	if len(m) != 0 {
		t.Errorf("empty Articles.CornerMap() should return empty map, got %d entries", len(m))
	}
}

func TestSummaries_RoundTrip(t *testing.T) {
	roundTrip[model.Summaries](t, loadFixture(t, "summaries.json"))
}

func TestSummaries_Fields(t *testing.T) {
	v := unmarshalFixture[model.Summaries](t, "summaries.json")
	if len(v.Corners) == 0 {
		t.Error("expected at least one corner")
	}
	cs := v.Corners[0]
	if cs.CornerTitle == "" {
		t.Error("CornerTitle must not be empty")
	}
	if len(cs.Summaries) == 0 {
		t.Error("expected at least one summary in corner")
	}
	s := cs.Summaries[0]
	if s.URL == "" {
		t.Error("URL must not be empty")
	}
	if s.Summary == "" {
		t.Error("Summary must not be empty")
	}
	if len(s.Points) == 0 {
		t.Error("Points must not be empty")
	}
}

func TestSummaries_CornerMap(t *testing.T) {
	sum1 := model.Summary{URL: "https://example.com/1", Summary: "S1", Points: []string{"p1"}}
	sum2 := model.Summary{URL: "https://example.com/2", Summary: "S2", Points: []string{"p2"}}
	s := model.Summaries{
		Corners: []model.CornerSummaries{
			{CornerTitle: "ニュース", Summaries: []model.Summary{sum1}},
			{CornerTitle: "エンディング", Summaries: []model.Summary{sum2}},
		},
	}

	m := s.CornerMap()

	if len(m) != 2 {
		t.Fatalf("map length: got %d, want 2", len(m))
	}
	if sums := m["ニュース"]; len(sums) != 1 || sums[0].URL != sum1.URL {
		t.Errorf("CornerMap[\"ニュース\"]: got %v, want [%v]", sums, sum1)
	}
	if sums := m["エンディング"]; len(sums) != 1 || sums[0].URL != sum2.URL {
		t.Errorf("CornerMap[\"エンディング\"]: got %v, want [%v]", sums, sum2)
	}
}

func TestSummaries_CornerMap_Empty(t *testing.T) {
	m := model.Summaries{}.CornerMap()
	if len(m) != 0 {
		t.Errorf("empty Summaries.CornerMap() should return empty map, got %d entries", len(m))
	}
}

func TestRundown_RoundTrip(t *testing.T) {
	roundTrip[model.Rundown](t, loadFixture(t, "rundown.json"))
}

func TestRundown_Fields(t *testing.T) {
	v := unmarshalFixture[model.Rundown](t, "rundown.json")
	if len(v.Corners) == 0 {
		t.Error("expected at least one corner")
	}
	c := v.Corners[0]
	if c.Title == "" {
		t.Error("Title must not be empty")
	}
	if c.Topic == "" {
		t.Error("Topic must not be empty")
	}
	if c.TargetChars <= 0 {
		t.Error("TargetChars must be positive")
	}
}

func TestLines_RoundTrip(t *testing.T) {
	roundTrip[model.Lines](t, loadFixture(t, "lines.json"))
}

func TestLines_Fields(t *testing.T) {
	v := unmarshalFixture[model.Lines](t, "lines.json")
	if len(v.Lines) == 0 {
		t.Error("expected at least one line")
	}
	l := v.Lines[0]
	if l.SpeakerRole == "" {
		t.Error("SpeakerRole must not be empty")
	}
	if l.Text == "" {
		t.Error("Text must not be empty")
	}
}

func TestScript_RoundTrip(t *testing.T) {
	roundTrip[model.Script](t, loadFixture(t, "script.json"))
}

func TestScript_Fields(t *testing.T) {
	v := unmarshalFixture[model.Script](t, "script.json")
	if len(v.Segments) == 0 {
		t.Error("expected at least one segment")
	}
	for i, seg := range v.Segments {
		if seg.Type == "" {
			t.Errorf("segment[%d]: Type must not be empty", i)
		}
		switch seg.Type {
		case model.SegmentTypeSpeech:
			if seg.SpeakerRole == "" {
				t.Errorf("segment[%d]: SpeakerRole must not be empty for speech", i)
			}
			if seg.Text == "" {
				t.Errorf("segment[%d]: Text must not be empty for speech", i)
			}
		case model.SegmentTypeSE:
			if seg.AssetName == "" {
				t.Errorf("segment[%d]: AssetName must not be empty for se", i)
			}
		case model.SegmentTypeBGM:
			// asset_name empty = stop; non-empty = start/switch
		case model.SegmentTypeJingle:
			if seg.AssetName == "" {
				t.Errorf("segment[%d]: AssetName must not be empty for jingle", i)
			}
		}
	}
}

func TestLine_PresetFields_OmitEmpty(t *testing.T) {
	line := model.Line{SpeakerRole: "zundamon", Text: "こんにちは"}
	b, err := json.Marshal(line)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}
	s := string(b)
	if strings.Contains(s, "intonation") {
		t.Errorf("should omit intonation when empty, got: %s", s)
	}
	if strings.Contains(s, "pitch") {
		t.Errorf("should omit pitch when empty, got: %s", s)
	}
	if strings.Contains(s, "speed") {
		t.Errorf("should omit speed when empty, got: %s", s)
	}
}

func TestLine_PresetFields_Present(t *testing.T) {
	line := model.Line{
		SpeakerRole: "zundamon",
		Text:        "こんにちは",
		Intonation:  "表現豊か",
		Pitch:       "高め",
		Speed:       "早口",
	}
	b, err := json.Marshal(line)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}
	s := string(b)
	if !strings.Contains(s, "表現豊か") {
		t.Errorf("should contain intonation value, got: %s", s)
	}
	if !strings.Contains(s, "高め") {
		t.Errorf("should contain pitch value, got: %s", s)
	}
	if !strings.Contains(s, "早口") {
		t.Errorf("should contain speed value, got: %s", s)
	}

	var got model.Line
	if err := json.Unmarshal(b, &got); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if got.Intonation != "表現豊か" {
		t.Errorf("Intonation: got %q, want 表現豊か", got.Intonation)
	}
	if got.Pitch != "高め" {
		t.Errorf("Pitch: got %q, want 高め", got.Pitch)
	}
	if got.Speed != "早口" {
		t.Errorf("Speed: got %q, want 早口", got.Speed)
	}
}

func TestScriptSegment_PresetFields_OmitEmpty(t *testing.T) {
	seg := model.ScriptSegment{Type: model.SegmentTypeSpeech, SpeakerRole: "zundamon", Text: "テスト"}
	b, err := json.Marshal(seg)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}
	s := string(b)
	if strings.Contains(s, "intonation") {
		t.Errorf("should omit intonation when empty, got: %s", s)
	}
	if strings.Contains(s, "pitch") {
		t.Errorf("should omit pitch when empty, got: %s", s)
	}
	if strings.Contains(s, "speed") {
		t.Errorf("should omit speed when empty, got: %s", s)
	}
}

func TestScriptSegment_PresetFields_Present(t *testing.T) {
	seg := model.ScriptSegment{
		Type:        model.SegmentTypeSpeech,
		SpeakerRole: "zundamon",
		Text:        "テスト",
		Intonation:  "棒読み",
		Pitch:       "低め",
		Speed:       "ゆっくり",
	}
	b, err := json.Marshal(seg)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}
	var got model.ScriptSegment
	if err := json.Unmarshal(b, &got); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if got.Intonation != "棒読み" {
		t.Errorf("Intonation: got %q, want 棒読み", got.Intonation)
	}
	if got.Pitch != "低め" {
		t.Errorf("Pitch: got %q, want 低め", got.Pitch)
	}
	if got.Speed != "ゆっくり" {
		t.Errorf("Speed: got %q, want ゆっくり", got.Speed)
	}
}

func TestClipsMeta_RoundTrip(t *testing.T) {
	roundTrip[model.ClipsMeta](t, loadFixture(t, "clips.json"))
}

func TestClipsMeta_Fields(t *testing.T) {
	v := unmarshalFixture[model.ClipsMeta](t, "clips.json")
	if len(v.Clips) == 0 {
		t.Error("expected at least one clip")
	}
	c := v.Clips[0]
	if c.File == "" {
		t.Error("File must not be empty")
	}
	if c.DurationSec <= 0 {
		t.Error("DurationSec must be positive")
	}
	if c.SpeakerRole == "" {
		t.Error("SpeakerRole must not be empty")
	}
	if c.Text == "" {
		t.Error("Text must not be empty")
	}
}
