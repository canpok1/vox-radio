package model_test

import (
	"encoding/json"
	"os"
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
	if len(v.Articles) == 0 {
		t.Error("expected at least one article")
	}
	a := v.Articles[0]
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

func TestSummaries_RoundTrip(t *testing.T) {
	roundTrip[model.Summaries](t, loadFixture(t, "summaries.json"))
}

func TestSummaries_Fields(t *testing.T) {
	v := unmarshalFixture[model.Summaries](t, "summaries.json")
	if len(v.Summaries) == 0 {
		t.Error("expected at least one summary")
	}
	s := v.Summaries[0]
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
			if seg.SEName == "" {
				t.Errorf("segment[%d]: SEName must not be empty for se", i)
			}
		}
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

func TestEpisodes_RoundTrip(t *testing.T) {
	roundTrip[model.Episodes](t, loadFixture(t, "episodes.json"))
}

func TestEpisodes_Fields(t *testing.T) {
	v := unmarshalFixture[model.Episodes](t, "episodes.json")
	if len(v.Episodes) == 0 {
		t.Error("expected at least one episode")
	}
	e := v.Episodes[0]
	if e.GUID == "" {
		t.Error("GUID must not be empty")
	}
	if e.Title == "" {
		t.Error("Title must not be empty")
	}
	if e.AudioURL == "" {
		t.Error("AudioURL must not be empty")
	}
	if e.Bytes <= 0 {
		t.Error("Bytes must be positive")
	}
	if e.Duration == "" {
		t.Error("Duration must not be empty")
	}
}
