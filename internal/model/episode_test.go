package model_test

import (
	"testing"
	"time"

	"github.com/canpok1/vox-radio/internal/model"
)

func TestEpisodeDisplayTitle(t *testing.T) {
	tests := []struct {
		name          string
		episodeNumber int
		episodeTitle  string
		fallbackTitle string
		want          string
	}{
		{"number+subtitle", 5, "技術ニュース", "番組タイトル", "第5回 技術ニュース"},
		{"number only", 5, "", "番組タイトル", "第5回"},
		{"no number with subtitle", 0, "技術ニュース", "番組タイトル", "番組タイトル"},
		{"no number no subtitle", 0, "", "番組タイトル", "番組タイトル"},
		{"all empty", 0, "", "", ""},
		{"number=1", 1, "初回", "番組名", "第1回 初回"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := model.EpisodeDisplayTitle(tt.episodeNumber, tt.episodeTitle, tt.fallbackTitle)
			if got != tt.want {
				t.Errorf("EpisodeDisplayTitle(%d, %q, %q) = %q, want %q",
					tt.episodeNumber, tt.episodeTitle, tt.fallbackTitle, got, tt.want)
			}
		})
	}
}

func TestEpisodeMeta_ZeroValue(t *testing.T) {
	var m model.EpisodeMeta
	if m.Number != 0 {
		t.Errorf("zero EpisodeMeta.Number = %d, want 0", m.Number)
	}
	if m.Title != "" {
		t.Errorf("zero EpisodeMeta.Title = %q, want empty", m.Title)
	}
	if !m.GeneratedAt.IsZero() {
		t.Errorf("zero EpisodeMeta.GeneratedAt should be zero time")
	}
}

func TestEpisodeMeta_Fields(t *testing.T) {
	ts := time.Date(2026, 6, 14, 12, 0, 0, 0, time.UTC)
	m := model.EpisodeMeta{
		Number:      10,
		Title:       "テストエピソード",
		GeneratedAt: ts,
	}
	if m.Number != 10 {
		t.Errorf("Number = %d, want 10", m.Number)
	}
	if m.Title != "テストエピソード" {
		t.Errorf("Title = %q, want テストエピソード", m.Title)
	}
	if !m.GeneratedAt.Equal(ts) {
		t.Errorf("GeneratedAt = %v, want %v", m.GeneratedAt, ts)
	}
}
