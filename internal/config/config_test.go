package config_test

import (
	"testing"

	"github.com/canpok1/vox-radio/internal/config"
)

func TestLoad(t *testing.T) {
	cfg, err := config.Load("testdata")
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	t.Run("Feeds", func(t *testing.T) {
		if len(cfg.Feeds.Feeds) == 0 {
			t.Error("expected at least one feed")
		}
		if cfg.Feeds.Feeds[0].URL == "" {
			t.Error("feed URL must not be empty")
		}
		if cfg.Feeds.Feeds[0].MaxItems <= 0 {
			t.Error("feed MaxItems must be positive")
		}
		if len(cfg.Feeds.Articles) == 0 {
			t.Error("expected at least one article URL")
		}
	})

	t.Run("Show", func(t *testing.T) {
		if cfg.Show.TitleFormat == "" {
			t.Error("TitleFormat must not be empty")
		}
		if cfg.Show.TargetChars <= 0 {
			t.Error("TargetChars must be positive")
		}
		if len(cfg.Show.Speakers) == 0 {
			t.Error("Speakers must not be empty")
		}
		if cfg.Show.SegmentPauseSec <= 0 {
			t.Error("SegmentPauseSec must be positive")
		}
	})

	t.Run("Assets", func(t *testing.T) {
		if len(cfg.Assets.Jingle) == 0 {
			t.Error("expected at least one jingle")
		}
		if len(cfg.Assets.SE) == 0 {
			t.Error("expected at least one SE")
		}
		if len(cfg.Assets.BGM) == 0 {
			t.Error("expected at least one BGM")
		}
	})

	t.Run("LLM", func(t *testing.T) {
		if cfg.LLM.BaseURL == "" {
			t.Error("BaseURL must not be empty")
		}
		if cfg.LLM.APIKeyEnv == "" {
			t.Error("APIKeyEnv must not be empty")
		}
		if cfg.LLM.Model == "" {
			t.Error("Model must not be empty")
		}
		if cfg.LLM.MaxRetries <= 0 {
			t.Error("MaxRetries must be positive")
		}
		if len(cfg.LLM.Steps) == 0 {
			t.Error("expected at least one step config")
		}
	})

	t.Run("Podcast", func(t *testing.T) {
		if cfg.Podcast.Title == "" {
			t.Error("Title must not be empty")
		}
		if cfg.Podcast.Language == "" {
			t.Error("Language must not be empty")
		}
		if cfg.Podcast.MaxItems <= 0 {
			t.Error("MaxItems must be positive")
		}
	})
}

func TestLoad_MissingDir(t *testing.T) {
	_, err := config.Load("testdata/nonexistent")
	if err == nil {
		t.Error("expected error for missing directory")
	}
}
