package publish

import (
	"bytes"
	"context"
	"io"
	"os"
	"testing"

	"github.com/canpok1/vox-radio/internal/config"
	"github.com/canpok1/vox-radio/internal/model"
)

type mockHosting struct {
	episodes  model.Episodes
	savedEps  model.Episodes
	savedFeed []byte
	audioName string
}

func (m *mockHosting) PutAudio(_ context.Context, name string, r io.Reader) (string, error) {
	m.audioName = name
	if _, err := io.Copy(io.Discard, r); err != nil {
		return "", err
	}
	return "https://example.com/audio/" + name, nil
}

func (m *mockHosting) PutFeed(_ context.Context, feedXML []byte) (string, error) {
	m.savedFeed = feedXML
	return "https://example.com/feed.xml", nil
}

func (m *mockHosting) LoadEpisodes(_ context.Context) (model.Episodes, error) {
	return m.episodes, nil
}

func (m *mockHosting) SaveEpisodes(_ context.Context, e model.Episodes) error {
	m.savedEps = e
	return nil
}

func (m *mockHosting) DeleteAudio(_ context.Context, _ string) error {
	return nil
}

func newTestPublisher(h *mockHosting) *Publisher {
	podcast := config.PodcastConfig{
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
	return &Publisher{
		Hosting:     h,
		Podcast:     podcast,
		getDuration: func(string) (float64, error) { return 312.0, nil },
		getFileSize: func(string) (int64, error) { return 5242880, nil },
	}
}

func newTempMP3(t *testing.T) string {
	t.Helper()
	f, err := os.CreateTemp(t.TempDir(), "*.mp3")
	if err != nil {
		t.Fatalf("create temp mp3: %v", err)
	}
	f.Close()
	return f.Name()
}

func TestPublisher_Run_CreatesEpisode(t *testing.T) {
	h := &mockHosting{episodes: model.Episodes{Episodes: make([]model.Episode, 0)}}
	p := newTestPublisher(h)

	if err := p.Run(context.Background(), newTempMP3(t), Options{Date: "2026-05-30"}); err != nil {
		t.Fatalf("Run: %v", err)
	}

	if len(h.savedEps.Episodes) != 1 {
		t.Fatalf("expected 1 episode, got %d", len(h.savedEps.Episodes))
	}
	ep := h.savedEps.Episodes[0]
	if ep.GUID != "episode-2026-05-30" {
		t.Errorf("guid = %q, want episode-2026-05-30", ep.GUID)
	}
	if ep.Duration != "00:05:12" {
		t.Errorf("duration = %q, want 00:05:12", ep.Duration)
	}
	if ep.Bytes != 5242880 {
		t.Errorf("bytes = %d, want 5242880", ep.Bytes)
	}
	if ep.PubDate != "2026-05-30T00:00:00Z" {
		t.Errorf("pub_date = %q, want 2026-05-30T00:00:00Z", ep.PubDate)
	}
}

func TestPublisher_Run_AudioName(t *testing.T) {
	h := &mockHosting{episodes: model.Episodes{Episodes: make([]model.Episode, 0)}}
	p := newTestPublisher(h)

	if err := p.Run(context.Background(), newTempMP3(t), Options{Date: "2026-05-30"}); err != nil {
		t.Fatalf("Run: %v", err)
	}

	if h.audioName != "episode_2026-05-30.mp3" {
		t.Errorf("audioName = %q, want episode_2026-05-30.mp3", h.audioName)
	}
}

func TestPublisher_Run_DefaultTitle(t *testing.T) {
	h := &mockHosting{episodes: model.Episodes{Episodes: make([]model.Episode, 0)}}
	p := newTestPublisher(h)

	if err := p.Run(context.Background(), newTempMP3(t), Options{Date: "2026-05-30"}); err != nil {
		t.Fatalf("Run: %v", err)
	}

	ep := h.savedEps.Episodes[0]
	want := "2026-05-30 今日のテックニュース"
	if ep.Title != want {
		t.Errorf("title = %q, want %q", ep.Title, want)
	}
}

func TestPublisher_Run_CustomTitle(t *testing.T) {
	h := &mockHosting{episodes: model.Episodes{Episodes: make([]model.Episode, 0)}}
	p := newTestPublisher(h)

	opts := Options{Date: "2026-05-30", Title: "Custom Title", Description: "Custom Desc"}
	if err := p.Run(context.Background(), newTempMP3(t), opts); err != nil {
		t.Fatalf("Run: %v", err)
	}

	ep := h.savedEps.Episodes[0]
	if ep.Title != "Custom Title" {
		t.Errorf("title = %q, want Custom Title", ep.Title)
	}
	if ep.Description != "Custom Desc" {
		t.Errorf("description = %q, want Custom Desc", ep.Description)
	}
}

func TestPublisher_Run_PrependNewest(t *testing.T) {
	existing := model.Episodes{
		Episodes: []model.Episode{
			{GUID: "episode-2026-05-29", Title: "Old", PubDate: "2026-05-29T00:00:00Z"},
		},
	}
	h := &mockHosting{episodes: existing}
	p := newTestPublisher(h)

	if err := p.Run(context.Background(), newTempMP3(t), Options{Date: "2026-05-30"}); err != nil {
		t.Fatalf("Run: %v", err)
	}

	if len(h.savedEps.Episodes) != 2 {
		t.Fatalf("expected 2 episodes, got %d", len(h.savedEps.Episodes))
	}
	if h.savedEps.Episodes[0].GUID != "episode-2026-05-30" {
		t.Errorf("newest episode should be first, got %q", h.savedEps.Episodes[0].GUID)
	}
}

func TestPublisher_Run_TrimsToMaxItems(t *testing.T) {
	existing := model.Episodes{
		Episodes: make([]model.Episode, 7),
	}
	for i := range existing.Episodes {
		existing.Episodes[i] = model.Episode{
			GUID:    "old",
			PubDate: "2026-05-01T00:00:00Z",
		}
	}
	h := &mockHosting{episodes: existing}
	p := newTestPublisher(h) // MaxItems=7

	if err := p.Run(context.Background(), newTempMP3(t), Options{Date: "2026-05-30"}); err != nil {
		t.Fatalf("Run: %v", err)
	}

	if len(h.savedEps.Episodes) != 7 {
		t.Errorf("expected 7 episodes after trim, got %d", len(h.savedEps.Episodes))
	}
	if h.savedEps.Episodes[0].GUID != "episode-2026-05-30" {
		t.Errorf("newest episode should be first")
	}
}

func TestPublisher_Run_MaxItemsZero_NoTrim(t *testing.T) {
	existing := model.Episodes{
		Episodes: []model.Episode{
			{GUID: "ep1", PubDate: "2026-05-01T00:00:00Z"},
			{GUID: "ep2", PubDate: "2026-05-02T00:00:00Z"},
		},
	}
	h := &mockHosting{episodes: existing}
	podcast := config.PodcastConfig{
		Title:    "Test",
		MaxItems: 0, // 0 = unlimited
	}
	p := &Publisher{
		Hosting:     h,
		Podcast:     podcast,
		getDuration: func(string) (float64, error) { return 60.0, nil },
		getFileSize: func(string) (int64, error) { return 1024, nil },
	}

	if err := p.Run(context.Background(), newTempMP3(t), Options{Date: "2026-05-30"}); err != nil {
		t.Fatalf("Run: %v", err)
	}

	if len(h.savedEps.Episodes) != 3 {
		t.Errorf("MaxItems=0 should not trim, expected 3 episodes, got %d", len(h.savedEps.Episodes))
	}
}

func TestPublisher_Run_GeneratesFeed(t *testing.T) {
	h := &mockHosting{episodes: model.Episodes{Episodes: make([]model.Episode, 0)}}
	p := newTestPublisher(h)

	if err := p.Run(context.Background(), newTempMP3(t), Options{Date: "2026-05-30"}); err != nil {
		t.Fatalf("Run: %v", err)
	}

	if len(h.savedFeed) == 0 {
		t.Error("feed should be generated")
	}
	if !bytes.Contains(h.savedFeed, []byte(`<?xml version="1.0" encoding="UTF-8"?>`)) {
		t.Errorf("feed missing XML declaration, got:\n%s", h.savedFeed)
	}
}

func TestPublisher_Run_InvalidDate(t *testing.T) {
	h := &mockHosting{episodes: model.Episodes{Episodes: make([]model.Episode, 0)}}
	p := newTestPublisher(h)

	err := p.Run(context.Background(), newTempMP3(t), Options{Date: "not-a-date"})
	if err == nil {
		t.Error("expected error for invalid date")
	}
}
