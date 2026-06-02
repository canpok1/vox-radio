package model_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/canpok1/vox-radio/internal/model"
)

func TestLoadDistribution_ValidYAML(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "distribution.yaml")
	content := `
program_id: zundamon-tech-radio
feed:
  language: ja
  author: testauthor
  email: test@example.com
  category: Technology
  explicit: false
  cover_image_url: https://example.com/cover.png
  site_url: https://example.com/
  audio_url_template: "https://github.com/owner/repo/releases/download/ep-{episode_number}/{audio_file}"
  credit: "VOICEVOX:ずんだもん"
output:
  public: public
`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write yaml: %v", err)
	}

	cfg, err := model.LoadDistribution(path)
	if err != nil {
		t.Fatalf("LoadDistribution: unexpected error: %v", err)
	}

	if cfg.ProgramID != "zundamon-tech-radio" {
		t.Errorf("ProgramID: got %q, want %q", cfg.ProgramID, "zundamon-tech-radio")
	}
	if cfg.Feed.Language != "ja" {
		t.Errorf("Feed.Language: got %q, want %q", cfg.Feed.Language, "ja")
	}
	if cfg.Feed.Author != "testauthor" {
		t.Errorf("Feed.Author: got %q, want %q", cfg.Feed.Author, "testauthor")
	}
	if cfg.Feed.Email != "test@example.com" {
		t.Errorf("Feed.Email: got %q, want %q", cfg.Feed.Email, "test@example.com")
	}
	if cfg.Feed.Category != "Technology" {
		t.Errorf("Feed.Category: got %q, want %q", cfg.Feed.Category, "Technology")
	}
	if cfg.Feed.Explicit {
		t.Errorf("Feed.Explicit: got true, want false")
	}
	if cfg.Feed.CoverImageURL != "https://example.com/cover.png" {
		t.Errorf("Feed.CoverImageURL: got %q, want %q", cfg.Feed.CoverImageURL, "https://example.com/cover.png")
	}
	if cfg.Feed.SiteURL != "https://example.com/" {
		t.Errorf("Feed.SiteURL: got %q, want %q", cfg.Feed.SiteURL, "https://example.com/")
	}
	wantTemplate := "https://github.com/owner/repo/releases/download/ep-{episode_number}/{audio_file}"
	if cfg.Feed.AudioURLTemplate != wantTemplate {
		t.Errorf("Feed.AudioURLTemplate: got %q, want %q", cfg.Feed.AudioURLTemplate, wantTemplate)
	}
	if cfg.Feed.Credit != "VOICEVOX:ずんだもん" {
		t.Errorf("Feed.Credit: got %q, want %q", cfg.Feed.Credit, "VOICEVOX:ずんだもん")
	}
	if cfg.Output.Public != "public" {
		t.Errorf("Output.Public: got %q, want %q", cfg.Output.Public, "public")
	}
}

func TestLoadDistribution_FileNotExist(t *testing.T) {
	_, err := model.LoadDistribution("/nonexistent/path/distribution.yaml")
	if err == nil {
		t.Error("LoadDistribution: expected error for nonexistent file, got nil")
	}
}

func TestDistributionConfig_EffectivePublicDir_Default(t *testing.T) {
	cfg := model.DistributionConfig{}
	got := cfg.EffectivePublicDir()
	if got != model.DefaultPublicDir {
		t.Errorf("EffectivePublicDir(): got %q, want %q", got, model.DefaultPublicDir)
	}
}

func TestDistributionConfig_EffectivePublicDir_Custom(t *testing.T) {
	cfg := model.DistributionConfig{
		Output: model.OutputConfig{Public: "dist/public"},
	}
	got := cfg.EffectivePublicDir()
	if got != "dist/public" {
		t.Errorf("EffectivePublicDir(): got %q, want %q", got, "dist/public")
	}
}
