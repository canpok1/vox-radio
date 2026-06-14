package feed_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/canpok1/vox-radio/internal/cache"
	"github.com/canpok1/vox-radio/internal/feed"
)

func TestBuildFeed_GoldenOutput(t *testing.T) {
	cfg := feed.FeedSpec{
		Feed: feed.FeedConfig{
			Language:         "ja",
			Author:           "testauthor",
			Email:            "test@example.com",
			Category:         "Technology",
			Explicit:         false,
			CoverImageURL:    "https://example.com/cover.png",
			SiteURL:          "https://example.com/",
			AudioURLTemplate: "https://example.com/releases/ep-{episode_number}/{audio_file}",
			Credit:           "VOICEVOX:ずんだもん",
		},
		Output: feed.OutputConfig{Public: "public"},
	}
	entries := []cache.Entry{
		{
			ProgramID:     "test-radio",
			Datetime:      "2024-01-01T10:00:00+09:00",
			EpisodeNumber: 1,
			EpisodeTitle:  "AIニュース特集",
			Title:         "テストラジオ",
			Summary:       "エピソード1の概要",
			Description:   "番組説明テキスト",
			AudioFile:     "episode.mp3",
			Bytes:         12345678,
			DurationSec:   1800,
		},
		{
			ProgramID:     "test-radio",
			Datetime:      "2024-01-08T10:00:00+09:00",
			EpisodeNumber: 2,
			EpisodeTitle:  "Go言語特集",
			Title:         "テストラジオ",
			Summary:       "エピソード2の概要",
			Description:   "番組説明テキスト",
			AudioFile:     "episode.mp3",
			Bytes:         23456789,
			DurationSec:   2100,
		},
	}

	got, err := feed.BuildFeed(cfg, entries)
	if err != nil {
		t.Fatalf("BuildFeed: unexpected error: %v", err)
	}

	goldenPath := filepath.Join("testdata", "feed_golden.xml")
	if os.Getenv("UPDATE_GOLDEN") == "1" {
		if err := os.MkdirAll(filepath.Dir(goldenPath), 0o755); err != nil {
			t.Fatalf("mkdir: %v", err)
		}
		if err := os.WriteFile(goldenPath, []byte(got), 0o644); err != nil {
			t.Fatalf("write golden: %v", err)
		}
		t.Logf("golden file updated: %s", goldenPath)
		return
	}

	wantBytes, err := os.ReadFile(goldenPath)
	if err != nil {
		t.Fatalf("read golden file: %v (run with UPDATE_GOLDEN=1 to generate)", err)
	}
	want := string(wantBytes)
	if got != want {
		t.Errorf("BuildFeed output mismatch\ngot:\n%s\nwant:\n%s", got, want)
	}
}

func TestBuildFeed_AudioURLTemplateSubstitution(t *testing.T) {
	cfg := feed.FeedSpec{
		Feed: feed.FeedConfig{
			AudioURLTemplate: "https://host.example/ep-{episode_number}/{audio_file}",
		},
	}
	entries := []cache.Entry{
		{
			ProgramID:     "radio",
			Datetime:      "2024-01-01T00:00:00Z",
			EpisodeNumber: 42,
			Title:         "タイトル",
			Summary:       "要約",
			AudioFile:     "episode.mp3",
			Bytes:         1000,
			DurationSec:   600,
		},
	}

	got, err := feed.BuildFeed(cfg, entries)
	if err != nil {
		t.Fatalf("BuildFeed: %v", err)
	}

	wantURL := "https://host.example/ep-42/episode.mp3"
	if !strings.Contains(got, wantURL) {
		t.Errorf("BuildFeed: expected URL %q in output\ngot:\n%s", wantURL, got)
	}
}

func TestBuildFeed_GUID(t *testing.T) {
	cfg := feed.FeedSpec{
		Feed: feed.FeedConfig{
			AudioURLTemplate: "https://host.example/ep-{episode_number}/{audio_file}",
		},
	}
	entries := []cache.Entry{
		{
			ProgramID:     "radio",
			Datetime:      "2024-01-01T00:00:00Z",
			EpisodeNumber: 5,
			Title:         "タイトル",
			Summary:       "要約",
			AudioFile:     "episode.mp3",
			Bytes:         1000,
			DurationSec:   600,
		},
	}

	got, err := feed.BuildFeed(cfg, entries)
	if err != nil {
		t.Fatalf("BuildFeed: %v", err)
	}

	if !strings.Contains(got, "ep-5") {
		t.Errorf("BuildFeed: expected GUID 'ep-5' in output\ngot:\n%s", got)
	}
}

func TestBuildFeed_ChannelFromLatestEntry(t *testing.T) {
	cfg := feed.FeedSpec{
		Feed: feed.FeedConfig{
			AudioURLTemplate: "https://host.example/{episode_number}/{audio_file}",
		},
	}
	entries := []cache.Entry{
		{
			ProgramID:     "radio",
			Datetime:      "2024-01-01T00:00:00Z",
			EpisodeNumber: 1,
			Title:         "古いタイトル",
			Description:   "古い番組説明",
			Summary:       "古い要約",
			AudioFile:     "episode.mp3",
			Bytes:         1000,
			DurationSec:   600,
		},
		{
			ProgramID:     "radio",
			Datetime:      "2024-01-08T00:00:00Z",
			EpisodeNumber: 2,
			Title:         "最新タイトル",
			Description:   "最新番組説明",
			Summary:       "最新要約",
			AudioFile:     "episode.mp3",
			Bytes:         2000,
			DurationSec:   700,
		},
	}

	got, err := feed.BuildFeed(cfg, entries)
	if err != nil {
		t.Fatalf("BuildFeed: %v", err)
	}

	// channel title and description should come from latest entry (entries[last])
	if !strings.Contains(got, "最新タイトル") {
		t.Errorf("BuildFeed: channel title should be from latest entry '最新タイトル'\ngot:\n%s", got)
	}
	if !strings.Contains(got, "最新番組説明") {
		t.Errorf("BuildFeed: channel description should be from latest entry '最新番組説明'\ngot:\n%s", got)
	}
}

func TestBuildFeed_EmptyEntries(t *testing.T) {
	cfg := feed.FeedSpec{
		Feed: feed.FeedConfig{
			AudioURLTemplate: "https://host.example/{episode_number}/{audio_file}",
		},
	}

	got, err := feed.BuildFeed(cfg, []cache.Entry{})
	if err != nil {
		t.Fatalf("BuildFeed: unexpected error for empty entries: %v", err)
	}

	if !strings.Contains(got, "<rss") {
		t.Errorf("BuildFeed: expected <rss> element in output\ngot:\n%s", got)
	}
	if strings.Contains(got, "<item>") {
		t.Errorf("BuildFeed: expected no <item> elements for empty entries\ngot:\n%s", got)
	}
}

func TestBuildFeed_CreditsAppendedToDescription(t *testing.T) {
	cfg := feed.FeedSpec{
		Feed: feed.FeedConfig{
			AudioURLTemplate: "https://host.example/{episode_number}/{audio_file}",
		},
	}
	entries := []cache.Entry{
		{
			ProgramID:     "radio",
			Datetime:      "2024-01-01T00:00:00Z",
			EpisodeNumber: 1,
			Title:         "タイトル",
			Summary:       "番組の要約テキスト",
			AudioFile:     "episode.mp3",
			Bytes:         1000,
			DurationSec:   600,
			Credits:       []string{"OtoLogic / CC BY 4.0", "VOICEVOX:ずんだもん"},
		},
	}

	got, err := feed.BuildFeed(cfg, entries)
	if err != nil {
		t.Fatalf("BuildFeed: %v", err)
	}

	// description にクレジット節が含まれること
	if !strings.Contains(got, "番組の要約テキスト") {
		t.Errorf("BuildFeed: expected summary in description\ngot:\n%s", got)
	}
	if !strings.Contains(got, feed.DefaultCreditsHeader) {
		t.Errorf("BuildFeed: expected credit section header in description\ngot:\n%s", got)
	}
	if !strings.Contains(got, "OtoLogic / CC BY 4.0") {
		t.Errorf("BuildFeed: expected credit 'OtoLogic / CC BY 4.0' in description\ngot:\n%s", got)
	}
	if !strings.Contains(got, "VOICEVOX:ずんだもん") {
		t.Errorf("BuildFeed: expected credit 'VOICEVOX:ずんだもん' in description\ngot:\n%s", got)
	}
}

func TestBuildFeed_CustomCreditsHeader(t *testing.T) {
	cfg := feed.FeedSpec{
		Feed: feed.FeedConfig{
			AudioURLTemplate: "https://host.example/{episode_number}/{audio_file}",
			CreditsHeader:    "Credits",
		},
	}
	entries := []cache.Entry{
		{
			ProgramID:     "radio",
			Datetime:      "2024-01-01T00:00:00Z",
			EpisodeNumber: 1,
			Title:         "タイトル",
			Summary:       "番組の要約テキスト",
			AudioFile:     "episode.mp3",
			Bytes:         1000,
			DurationSec:   600,
			Credits:       []string{"OtoLogic / CC BY 4.0"},
		},
	}

	got, err := feed.BuildFeed(cfg, entries)
	if err != nil {
		t.Fatalf("BuildFeed: %v", err)
	}

	if !strings.Contains(got, "Credits") {
		t.Errorf("BuildFeed: expected custom credits header 'Credits' in description\ngot:\n%s", got)
	}
	if strings.Contains(got, feed.DefaultCreditsHeader) {
		t.Errorf("BuildFeed: expected %q to be replaced by custom header\ngot:\n%s", feed.DefaultCreditsHeader, got)
	}
}

func TestBuildFeed_NoCreditsWhenEmpty(t *testing.T) {
	cfg := feed.FeedSpec{
		Feed: feed.FeedConfig{
			AudioURLTemplate: "https://host.example/{episode_number}/{audio_file}",
		},
	}
	entries := []cache.Entry{
		{
			ProgramID:     "radio",
			Datetime:      "2024-01-01T00:00:00Z",
			EpisodeNumber: 1,
			Title:         "タイトル",
			Summary:       "番組の要約テキスト",
			AudioFile:     "episode.mp3",
			Bytes:         1000,
			DurationSec:   600,
		},
	}

	got, err := feed.BuildFeed(cfg, entries)
	if err != nil {
		t.Fatalf("BuildFeed: %v", err)
	}

	// credits が空のとき「クレジット」節が追加されないこと
	if strings.Contains(got, feed.DefaultCreditsHeader) {
		t.Errorf("BuildFeed: should not contain credit section when credits is empty\ngot:\n%s", got)
	}
}
