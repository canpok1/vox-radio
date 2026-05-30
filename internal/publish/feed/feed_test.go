package feed_test

import (
	"bytes"
	"os"
	"testing"

	"github.com/canpok1/vox-radio/internal/config"
	"github.com/canpok1/vox-radio/internal/model"
	"github.com/canpok1/vox-radio/internal/publish/feed"
)

var testPodcast = config.PodcastConfig{
	Title:         "今日のテックニュース",
	Description:   "毎日5分のニュースラジオ",
	Language:      "ja",
	Author:        "vox-radio",
	Category:      "News",
	Explicit:      false,
	CoverImageURL: "https://example.github.io/vox-radio/cover.jpg",
	SiteURL:       "https://example.github.io/vox-radio/",
	MaxItems:      7,
}

func TestGenerate_GoldenTest(t *testing.T) {
	episodes := model.Episodes{
		Episodes: []model.Episode{
			{
				GUID:        "episode-2026-05-30",
				Title:       "2026-05-30 今日のテックニュース",
				Description: "本日の話題は新型AIチップとオープンソースの動向です",
				PubDate:     "2026-05-30T21:00:00Z",
				AudioURL:    "https://example.github.io/vox-radio/audio/episode_2026-05-30.mp3",
				Bytes:       5242880,
				Duration:    "00:05:12",
			},
		},
	}

	got, err := feed.Generate(testPodcast, episodes)
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}

	goldenPath := "testdata/golden/feed.xml"
	if os.Getenv("UPDATE_GOLDEN") == "1" {
		if err := os.MkdirAll("testdata/golden", 0o755); err != nil {
			t.Fatalf("mkdir: %v", err)
		}
		if err := os.WriteFile(goldenPath, got, 0o644); err != nil {
			t.Fatalf("write golden: %v", err)
		}
		return
	}

	want, err := os.ReadFile(goldenPath)
	if err != nil {
		t.Fatalf("read golden %s: %v", goldenPath, err)
	}
	if !bytes.Equal(got, want) {
		t.Errorf("feed.xml mismatch:\ngot:\n%s\nwant:\n%s", got, want)
	}
}

func TestGenerate_EmptyEpisodes(t *testing.T) {
	episodes := model.Episodes{Episodes: make([]model.Episode, 0)}

	got, err := feed.Generate(testPodcast, episodes)
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	if !bytes.Contains(got, []byte(`<?xml version="1.0" encoding="UTF-8"?>`)) {
		t.Error("missing XML declaration")
	}
	if !bytes.Contains(got, []byte(`<title>今日のテックニュース</title>`)) {
		t.Error("missing channel title")
	}
	if bytes.Contains(got, []byte(`<item>`)) {
		t.Error("unexpected <item> for empty episodes")
	}
}

func TestGenerate_XMLEscaping(t *testing.T) {
	podcast := config.PodcastConfig{
		Title:       "Test & Show",
		Description: "A show with <special> chars",
		Language:    "en",
		SiteURL:     "https://example.com/",
	}
	episodes := model.Episodes{Episodes: make([]model.Episode, 0)}

	got, err := feed.Generate(podcast, episodes)
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	if !bytes.Contains(got, []byte("Test &amp; Show")) {
		t.Errorf("& should be escaped to &amp;, got:\n%s", got)
	}
	if bytes.Contains(got, []byte("<special>")) {
		t.Error("< > should be XML-escaped")
	}
	if !bytes.Contains(got, []byte("&lt;special&gt;")) {
		t.Errorf("< > should be escaped to &lt;&gt;, got:\n%s", got)
	}
}

func TestGenerate_InvalidPubDate(t *testing.T) {
	episodes := model.Episodes{
		Episodes: []model.Episode{
			{
				GUID:    "episode-bad",
				PubDate: "not-a-date",
			},
		},
	}

	_, err := feed.Generate(testPodcast, episodes)
	if err == nil {
		t.Error("expected error for invalid pub_date")
	}
}

func TestGenerate_ExplicitYes(t *testing.T) {
	podcast := config.PodcastConfig{
		Title:    "Explicit Show",
		Language: "en",
		SiteURL:  "https://example.com/",
		Explicit: true,
	}
	episodes := model.Episodes{Episodes: make([]model.Episode, 0)}

	got, err := feed.Generate(podcast, episodes)
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	if !bytes.Contains(got, []byte("<itunes:explicit>yes</itunes:explicit>")) {
		t.Errorf("expected explicit=yes, got:\n%s", got)
	}
}
