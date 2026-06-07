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

func TestScriptLines_RoundTrip(t *testing.T) {
	original := model.ScriptLines{
		Corners: []model.CornerLines{
			{
				Title:     "オープニング",
				Direction: "冒頭でジングルを流す。",
				Lines: []model.Line{
					{SpeakerRole: "zundamon", Text: "こんにちは"},
				},
			},
			{
				Title: "エンディング",
				Lines: []model.Line{
					{SpeakerRole: "metan", Text: "さようなら"},
				},
			},
		},
	}

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}

	var got model.ScriptLines
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	if len(got.Corners) != 2 {
		t.Fatalf("Corners: got %d, want 2", len(got.Corners))
	}
	if got.Corners[0].Title != "オープニング" {
		t.Errorf("Corners[0].Title: got %q, want オープニング", got.Corners[0].Title)
	}
	if got.Corners[0].Direction != "冒頭でジングルを流す。" {
		t.Errorf("Corners[0].Direction: got %q, want 冒頭でジングルを流す。", got.Corners[0].Direction)
	}
	if len(got.Corners[0].Lines) != 1 || got.Corners[0].Lines[0].Text != "こんにちは" {
		t.Errorf("Corners[0].Lines: unexpected %+v", got.Corners[0].Lines)
	}
	if got.Corners[1].Direction != "" {
		t.Errorf("Corners[1].Direction: got %q, want empty (omitempty)", got.Corners[1].Direction)
	}
}

func TestScriptLines_DirectionOmittedWhenEmpty(t *testing.T) {
	sl := model.ScriptLines{
		Corners: []model.CornerLines{
			{Title: "テスト", Lines: make([]model.Line, 0)},
		},
	}
	data, err := json.Marshal(sl)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}
	if strings.Contains(string(data), `"direction"`) {
		t.Errorf("direction field should be omitted when empty, got: %s", string(data))
	}
}

func TestScriptLines_ProgramDirection_RoundTrip(t *testing.T) {
	original := model.ScriptLines{
		Direction: "番組全体の演出指示",
		Corners:   []model.CornerLines{{Title: "C1", Lines: make([]model.Line, 0)}},
	}
	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}
	if !strings.Contains(string(data), `"direction":"番組全体の演出指示"`) {
		t.Errorf("direction field should be present, got: %s", string(data))
	}
	var got model.ScriptLines
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if got.Direction != "番組全体の演出指示" {
		t.Errorf("Direction: got %q, want 番組全体の演出指示", got.Direction)
	}
}

func TestScriptLines_ProgramDirection_OmittedWhenEmpty(t *testing.T) {
	sl := model.ScriptLines{
		Corners: []model.CornerLines{{Title: "C1", Lines: make([]model.Line, 0)}},
	}
	data, err := json.Marshal(sl)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}
	if strings.Contains(string(data), `"direction"`) {
		t.Errorf("direction field should be omitted when empty, got: %s", string(data))
	}
}

func TestScriptLines_TotalLines(t *testing.T) {
	tests := []struct {
		name string
		sl   model.ScriptLines
		want int
	}{
		{
			name: "empty corners",
			sl:   model.ScriptLines{},
			want: 0,
		},
		{
			name: "single corner with lines",
			sl: model.ScriptLines{
				Corners: []model.CornerLines{
					{Title: "C1", Lines: []model.Line{{SpeakerRole: "a", Text: "x"}, {SpeakerRole: "b", Text: "y"}}},
				},
			},
			want: 2,
		},
		{
			name: "multiple corners",
			sl: model.ScriptLines{
				Corners: []model.CornerLines{
					{Title: "C1", Lines: []model.Line{{SpeakerRole: "a", Text: "x"}}},
					{Title: "C2", Lines: []model.Line{{SpeakerRole: "b", Text: "y"}, {SpeakerRole: "c", Text: "z"}}},
				},
			},
			want: 3,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.sl.TotalLines()
			if got != tt.want {
				t.Errorf("TotalLines() = %d, want %d", got, tt.want)
			}
		})
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
	if c.Flow == "" {
		t.Error("Flow must not be empty")
	}
	if len(c.Articles) == 0 {
		t.Error("Articles must not be empty")
	}
	a := c.Articles[0]
	if a.URL == "" {
		t.Error("Article.URL must not be empty")
	}
	if a.Title == "" {
		t.Error("Article.Title must not be empty")
	}
	if a.Summary == "" {
		t.Error("Article.Summary must not be empty")
	}
	if len(a.Points) == 0 {
		t.Error("Article.Points must not be empty")
	}
}

func TestRundown_CornerMap(t *testing.T) {
	art1 := model.RundownArticle{URL: "https://example.com/1", Title: "T1", Summary: "S1", Points: []string{"p1"}}
	art2 := model.RundownArticle{URL: "https://example.com/2", Title: "T2", Summary: "S2", Points: []string{"p2"}}
	rd := model.Rundown{
		Corners: []model.RundownCorner{
			{Title: "ニュース", Flow: "最新ニュースを紹介", Articles: []model.RundownArticle{art1}},
			{Title: "エンディング", Flow: "締めの言葉", Articles: []model.RundownArticle{art2}},
		},
	}

	m := rd.CornerMap()

	if len(m) != 2 {
		t.Fatalf("map length: got %d, want 2", len(m))
	}
	news, ok := m["ニュース"]
	if !ok {
		t.Fatal("key ニュース not found")
	}
	if len(news.Articles) != 1 || news.Articles[0].URL != art1.URL {
		t.Errorf("CornerMap[\"ニュース\"].Articles: got %v, want [%v]", news.Articles, art1)
	}
	if news.Flow != "最新ニュースを紹介" {
		t.Errorf("CornerMap[\"ニュース\"].Flow: got %q, want %q", news.Flow, "最新ニュースを紹介")
	}
	if _, ok := m["存在しないコーナー"]; ok {
		t.Error("missing key should not exist in map")
	}
}

func TestRundown_CornerMap_Empty(t *testing.T) {
	m := model.Rundown{}.CornerMap()
	if len(m) != 0 {
		t.Errorf("empty Rundown.CornerMap() should return empty map, got %d entries", len(m))
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

func TestConversationNote_JSONRoundTrip(t *testing.T) {
	note := model.ConversationNote{
		Category:     "近況",
		CharacterIDs: []string{"zundamon", "metan"},
		Note:         "ずんだもんがカフェにハマっている",
	}
	b, err := json.Marshal(note)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}
	var got model.ConversationNote
	if err := json.Unmarshal(b, &got); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if got.Category != "近況" {
		t.Errorf("Category: got %q, want %q", got.Category, "近況")
	}
	if len(got.CharacterIDs) != 2 || got.CharacterIDs[0] != "zundamon" {
		t.Errorf("CharacterIDs: got %v, want [zundamon metan]", got.CharacterIDs)
	}
	if got.Note != "ずんだもんがカフェにハマっている" {
		t.Errorf("Note: got %q, want %q", got.Note, "ずんだもんがカフェにハマっている")
	}
}

func TestConversationNote_CharacterIDsEmptyArrayNotNull(t *testing.T) {
	note := model.ConversationNote{
		Category:     "ハプニング",
		CharacterIDs: make([]string, 0),
		Note:         "誰かが噛んだ",
	}
	b, err := json.Marshal(note)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}
	s := string(b)
	if !strings.Contains(s, `"character_ids":[]`) {
		t.Errorf("character_ids should be [] not null: %s", s)
	}
}

func TestProgramSummary_JSONRoundTrip(t *testing.T) {
	ps := model.ProgramSummary{
		Summary:      "番組全体の要約",
		EpisodeTitle: "今週の面白技術",
		ConversationNotes: []model.ConversationNote{
			{Category: "掛け合い", CharacterIDs: []string{"zundamon"}, Note: "ずんだもんが食レポした"},
		},
	}
	b, err := json.Marshal(ps)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}
	var got model.ProgramSummary
	if err := json.Unmarshal(b, &got); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if got.Summary != "番組全体の要約" {
		t.Errorf("Summary: got %q, want %q", got.Summary, "番組全体の要約")
	}
	if got.EpisodeTitle != "今週の面白技術" {
		t.Errorf("EpisodeTitle: got %q, want %q", got.EpisodeTitle, "今週の面白技術")
	}
	if len(got.ConversationNotes) != 1 {
		t.Fatalf("ConversationNotes: got %d, want 1", len(got.ConversationNotes))
	}
	if got.ConversationNotes[0].Category != "掛け合い" {
		t.Errorf("ConversationNotes[0].Category: got %q, want %q", got.ConversationNotes[0].Category, "掛け合い")
	}
}

func TestProgramSummary_ConversationNotesEmptyArrayNotNull(t *testing.T) {
	ps := model.ProgramSummary{
		Summary:           "要約",
		ConversationNotes: make([]model.ConversationNote, 0),
	}
	b, err := json.Marshal(ps)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}
	s := string(b)
	if !strings.Contains(s, `"conversation_notes":[]`) {
		t.Errorf("conversation_notes should be [] not null: %s", s)
	}
}

func TestManifest_ConversationNotesEmptyArrayNotNull(t *testing.T) {
	m := model.Manifest{
		ConversationNotes: make([]model.ConversationNote, 0),
	}
	b, err := json.Marshal(m)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}
	s := string(b)
	if !strings.Contains(s, `"conversation_notes":[]`) {
		t.Errorf("conversation_notes should be [] not null: %s", s)
	}
}

func TestCornerLines_AudioFields_MarshaledAndUnmarshaled(t *testing.T) {
	cl := model.CornerLines{
		Title:      "C1",
		StartAudio: &model.CornerAudio{Type: model.SegmentTypeJingle, AssetName: "opening"},
		EndAudio:   &model.CornerAudio{Type: model.SegmentTypeSE, AssetName: "chime"},
		BGM:        "talk_bgm",
		Lines:      []model.Line{{SpeakerRole: "host", Text: "hello"}},
	}
	data, err := json.Marshal(cl)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}
	var got model.CornerLines
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}
	if got.StartAudio == nil {
		t.Fatal("StartAudio: got nil, want non-nil")
	}
	if got.StartAudio.Type != model.SegmentTypeJingle {
		t.Errorf("StartAudio.Type: got %q, want jingle", got.StartAudio.Type)
	}
	if got.StartAudio.AssetName != "opening" {
		t.Errorf("StartAudio.AssetName: got %q, want opening", got.StartAudio.AssetName)
	}
	if got.EndAudio == nil {
		t.Fatal("EndAudio: got nil, want non-nil")
	}
	if got.EndAudio.Type != model.SegmentTypeSE {
		t.Errorf("EndAudio.Type: got %q, want se", got.EndAudio.Type)
	}
	if got.EndAudio.AssetName != "chime" {
		t.Errorf("EndAudio.AssetName: got %q, want chime", got.EndAudio.AssetName)
	}
	if got.BGM != "talk_bgm" {
		t.Errorf("BGM: got %q, want talk_bgm", got.BGM)
	}
}

func TestRundown_Casts_EmptySliceNotNull(t *testing.T) {
	rd := model.Rundown{
		Corners: []model.RundownCorner{{Title: "op", Flow: "test", Articles: []model.RundownArticle{}}},
		Casts:   make([]model.RundownCast, 0),
	}
	data, err := json.Marshal(rd)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}
	if !strings.Contains(string(data), `"casts":[]`) {
		t.Errorf("expected casts to be [] (not null), got: %s", string(data))
	}
}

func TestRundown_Casts_RoundTrip(t *testing.T) {
	rd := model.Rundown{
		Corners: []model.RundownCorner{},
		Casts: []model.RundownCast{
			{CharacterID: "metan", Role: "解説", Type: "guest"},
			{CharacterID: "zundamon", Role: "MC", Type: "regular"},
		},
	}
	data, err := json.Marshal(rd)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}
	var rd2 model.Rundown
	if err := json.Unmarshal(data, &rd2); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}
	if len(rd2.Casts) != 2 {
		t.Fatalf("expected 2 casts, got %d", len(rd2.Casts))
	}
	if rd2.Casts[0].CharacterID != "metan" || rd2.Casts[1].CharacterID != "zundamon" {
		t.Errorf("unexpected casts order: %+v", rd2.Casts)
	}
	if rd2.Casts[0].Type != "guest" || rd2.Casts[1].Type != "regular" {
		t.Errorf("unexpected casts types: %+v", rd2.Casts)
	}
}

func TestRundownCast_PastAppearanceCount(t *testing.T) {
	tests := []struct {
		name            string
		appearanceCount int
		want            int
	}{
		{"初登場（1）は0を返す", 1, 0},
		{"4回目（4）は3を返す", 4, 3},
		{"0のとき0クランプ", 0, 0},
		{"負値のとき0クランプ", -1, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := model.RundownCast{AppearanceCount: tt.appearanceCount}
			got := c.PastAppearanceCount()
			if got != tt.want {
				t.Errorf("PastAppearanceCount() = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestCastsForLLM_ConvertsAppearanceCount(t *testing.T) {
	original := []model.RundownCast{
		{CharacterID: "zundamon", Role: "MC", Type: "regular", AppearanceCount: 5},
		{CharacterID: "guest1", Role: "ゲスト", Type: "guest", AppearanceCount: 1},
	}

	got := model.CastsForLLM(original)

	if len(got) != 2 {
		t.Fatalf("CastsForLLM: got %d items, want 2", len(got))
	}
	if got[0].AppearanceCount != 4 {
		t.Errorf("got[0].AppearanceCount = %d, want 4 (5-1)", got[0].AppearanceCount)
	}
	if got[1].AppearanceCount != 0 {
		t.Errorf("got[1].AppearanceCount = %d, want 0 (1-1)", got[1].AppearanceCount)
	}
}

func TestCastsForLLM_DoesNotModifyOriginal(t *testing.T) {
	original := []model.RundownCast{
		{CharacterID: "zundamon", Role: "MC", Type: "regular", AppearanceCount: 5},
	}

	_ = model.CastsForLLM(original)

	if original[0].AppearanceCount != 5 {
		t.Errorf("original[0].AppearanceCount modified: got %d, want 5", original[0].AppearanceCount)
	}
}

func TestCastsForLLM_Empty(t *testing.T) {
	got := model.CastsForLLM([]model.RundownCast{})
	if len(got) != 0 {
		t.Errorf("CastsForLLM(empty) should return empty slice, got %d items", len(got))
	}
}

func TestCornerLines_EmptyAssetFields_OmittedFromJSON(t *testing.T) {
	cl := model.CornerLines{
		Title: "C1",
		Lines: []model.Line{{SpeakerRole: "host", Text: "hello"}},
	}
	data, err := json.Marshal(cl)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}
	jsonStr := string(data)
	for _, field := range []string{`"start_audio"`, `"end_audio"`, `"bgm"`} {
		if strings.Contains(jsonStr, field) {
			t.Errorf("field %q should be omitted when empty, got: %s", field, jsonStr)
		}
	}
}

func TestProofreadResult_RoundTrip(t *testing.T) {
	pr := model.ProofreadResult{
		Corrections: []model.ProofreadCorrection{
			{CornerIndex: 0, LineIndex: 1, Before: "まえ", After: "あと", Reason: "理由"},
		},
	}
	data, err := json.Marshal(pr)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}
	var got model.ProofreadResult
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}
	if len(got.Corrections) != 1 {
		t.Fatalf("Corrections: got %d, want 1", len(got.Corrections))
	}
	c := got.Corrections[0]
	if c.CornerIndex != 0 {
		t.Errorf("CornerIndex: got %d, want 0", c.CornerIndex)
	}
	if c.LineIndex != 1 {
		t.Errorf("LineIndex: got %d, want 1", c.LineIndex)
	}
	if c.Before != "まえ" {
		t.Errorf("Before: got %q, want まえ", c.Before)
	}
	if c.After != "あと" {
		t.Errorf("After: got %q, want あと", c.After)
	}
	if c.Reason != "理由" {
		t.Errorf("Reason: got %q, want 理由", c.Reason)
	}
}

func TestProofreadResult_EmptyCorrections_NotNull(t *testing.T) {
	pr := model.ProofreadResult{Corrections: make([]model.ProofreadCorrection, 0)}
	data, err := json.Marshal(pr)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}
	if !strings.Contains(string(data), `"corrections":[]`) {
		t.Errorf("corrections should be [] not null: %s", string(data))
	}
}

func TestProofreadCorrection_ReasonOmittedWhenEmpty(t *testing.T) {
	c := model.ProofreadCorrection{CornerIndex: 0, LineIndex: 0, Before: "a", After: "b"}
	data, err := json.Marshal(c)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}
	if strings.Contains(string(data), `"reason"`) {
		t.Errorf("reason should be omitted when empty: %s", string(data))
	}
}
