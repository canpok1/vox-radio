package manifest_test

import (
	"testing"
	"time"

	"github.com/canpok1/vox-radio/internal/config"
	"github.com/canpok1/vox-radio/internal/manifest"
	"github.com/canpok1/vox-radio/internal/model"
)

var fixedTime = time.Date(2026, 5, 31, 12, 34, 56, 0, time.UTC)

func TestBuild(t *testing.T) {
	program := config.ProgramConfig{
		Title:       "今日のテックニュース",
		Description: "毎日5分のニュースラジオ",
	}

	corners := []config.CornerConfig{
		{Title: "オープニング"},
		{Title: "今日のテックニュース"},
	}

	articles := model.Articles{
		Corners: []model.CornerArticles{
			{
				CornerTitle: "今日のテックニュース",
				Articles: []model.Article{
					{Title: "記事タイトル", URL: "https://example.com/articles/123"},
				},
			},
		},
	}

	t.Run("title and description from program", func(t *testing.T) {
		got := manifest.Build(program, corners, articles, "episode.mp3", fixedTime)
		if got.Title != program.Title {
			t.Errorf("Title = %q, want %q", got.Title, program.Title)
		}
		if got.Description != program.Description {
			t.Errorf("Description = %q, want %q", got.Description, program.Description)
		}
	})

	t.Run("audio_file is set", func(t *testing.T) {
		got := manifest.Build(program, corners, articles, "episode.mp3", fixedTime)
		if got.AudioFile != "episode.mp3" {
			t.Errorf("AudioFile = %q, want %q", got.AudioFile, "episode.mp3")
		}
	})

	t.Run("datetime is RFC3339 UTC", func(t *testing.T) {
		got := manifest.Build(program, corners, articles, "episode.mp3", fixedTime)
		want := "2026-05-31T12:34:56Z"
		if got.Datetime != want {
			t.Errorf("Datetime = %q, want %q", got.Datetime, want)
		}
	})

	t.Run("corners in profile order", func(t *testing.T) {
		got := manifest.Build(program, corners, articles, "episode.mp3", fixedTime)
		if len(got.Corners) != len(corners) {
			t.Fatalf("len(Corners) = %d, want %d", len(got.Corners), len(corners))
		}
		for i, c := range corners {
			if got.Corners[i].Title != c.Title {
				t.Errorf("Corners[%d].Title = %q, want %q", i, got.Corners[i].Title, c.Title)
			}
		}
	})

	t.Run("corner without articles has empty array not null", func(t *testing.T) {
		got := manifest.Build(program, corners, articles, "episode.mp3", fixedTime)
		opening := got.Corners[0]
		if opening.Articles == nil {
			t.Error("Articles for corner without articles must be [] not nil")
		}
		if len(opening.Articles) != 0 {
			t.Errorf("Articles for corner without articles = %v, want []", opening.Articles)
		}
	})

	t.Run("articles attributed to correct corner", func(t *testing.T) {
		got := manifest.Build(program, corners, articles, "episode.mp3", fixedTime)
		techCorner := got.Corners[1]
		if len(techCorner.Articles) != 1 {
			t.Fatalf("len(Articles) = %d, want 1", len(techCorner.Articles))
		}
		if techCorner.Articles[0].Title != "記事タイトル" {
			t.Errorf("Articles[0].Title = %q, want %q", techCorner.Articles[0].Title, "記事タイトル")
		}
		if techCorner.Articles[0].URL != "https://example.com/articles/123" {
			t.Errorf("Articles[0].URL = %q, want %q", techCorner.Articles[0].URL, "https://example.com/articles/123")
		}
	})

	t.Run("no articles arg produces empty articles arrays", func(t *testing.T) {
		emptyArticles := model.Articles{}
		got := manifest.Build(program, corners, emptyArticles, "episode.mp3", fixedTime)
		for i, c := range got.Corners {
			if c.Articles == nil {
				t.Errorf("Corners[%d].Articles must be [] not nil", i)
			}
		}
	})

	t.Run("corners slice is not nil", func(t *testing.T) {
		got := manifest.Build(program, []config.CornerConfig{}, articles, "episode.mp3", fixedTime)
		if got.Corners == nil {
			t.Error("Corners must be [] not nil")
		}
	})
}
