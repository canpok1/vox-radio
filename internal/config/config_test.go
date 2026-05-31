package config_test

import (
	"path/filepath"
	"testing"

	"github.com/canpok1/vox-radio/internal/config"
)

func TestLoadConfig(t *testing.T) {
	cfg, err := config.LoadConfig("testdata/config.yaml")
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}

	t.Run("LLM", func(t *testing.T) {
		if cfg.LLM.BaseURL == "" {
			t.Error("LLM.BaseURL must not be empty")
		}
		if cfg.LLM.APIKeyEnv == "" {
			t.Error("LLM.APIKeyEnv must not be empty")
		}
		if cfg.LLM.Model == "" {
			t.Error("LLM.Model must not be empty")
		}
		if cfg.LLM.MaxRetries <= 0 {
			t.Error("LLM.MaxRetries must be positive")
		}
		if len(cfg.LLM.Steps) == 0 {
			t.Error("LLM.Steps must not be empty")
		}
	})
}

func TestLoadConfig_MissingFile(t *testing.T) {
	_, err := config.LoadConfig("testdata/nonexistent.yaml")
	if err == nil {
		t.Error("expected error for missing file")
	}
}

func TestLoadProfile(t *testing.T) {
	profile, err := config.LoadProfile("testdata/profile.yaml")
	if err != nil {
		t.Fatalf("LoadProfile failed: %v", err)
	}

	t.Run("Podcast", func(t *testing.T) {
		if profile.Podcast.Title == "" {
			t.Error("Podcast.Title must not be empty")
		}
		if profile.Podcast.Language == "" {
			t.Error("Podcast.Language must not be empty")
		}
		if profile.Podcast.MaxItems <= 0 {
			t.Error("Podcast.MaxItems must be positive")
		}
	})

	t.Run("Show", func(t *testing.T) {
		if profile.Show.TitleFormat == "" {
			t.Error("Show.TitleFormat must not be empty")
		}
		if profile.Show.TargetChars <= 0 {
			t.Error("Show.TargetChars must be positive")
		}
		if profile.Show.SegmentPauseSec <= 0 {
			t.Error("Show.SegmentPauseSec must be positive")
		}
	})

	t.Run("Feeds", func(t *testing.T) {
		if len(profile.Feeds) == 0 {
			t.Error("Feeds must not be empty")
		}
		if profile.Feeds[0].URL == "" {
			t.Error("Feeds[0].URL must not be empty")
		}
		if len(profile.Articles) == 0 {
			t.Error("Articles must not be empty")
		}
	})

	t.Run("Assets", func(t *testing.T) {
		if len(profile.Assets.Jingle) == 0 {
			t.Error("Assets.Jingle must not be empty")
		}
		if len(profile.Assets.SE) == 0 {
			t.Error("Assets.SE must not be empty")
		}
		if len(profile.Assets.BGM) == 0 {
			t.Error("Assets.BGM must not be empty")
		}
	})

	t.Run("Assets_PathResolution", func(t *testing.T) {
		base := "testdata"

		jingle, ok := profile.Assets.Jingle["opening"]
		if !ok {
			t.Fatal("Assets.Jingle[\"opening\"] not found")
		}
		if want := filepath.Join(base, "assets/jingle/opening.mp3"); jingle.File != want {
			t.Errorf("Jingle[\"opening\"].File: expected %q, got %q", want, jingle.File)
		}

		se, ok := profile.Assets.SE["chime"]
		if !ok {
			t.Fatal("Assets.SE[\"chime\"] not found")
		}
		if want := filepath.Join(base, "assets/se/chime.wav"); se.File != want {
			t.Errorf("SE[\"chime\"].File: expected %q, got %q", want, se.File)
		}

		bgm, ok := profile.Assets.BGM["talk_bgm"]
		if !ok {
			t.Fatal("Assets.BGM[\"talk_bgm\"] not found")
		}
		if want := filepath.Join(base, "assets/bgm/talk.mp3"); bgm.File != want {
			t.Errorf("BGM[\"talk_bgm\"].File: expected %q, got %q", want, bgm.File)
		}
	})
}

func TestLoadProfile_AbsolutePaths(t *testing.T) {
	profile, err := config.LoadProfile("testdata/profile_abs.yaml")
	if err != nil {
		t.Fatalf("LoadProfile failed: %v", err)
	}

	for name, entry := range profile.Assets.Jingle {
		if !filepath.IsAbs(entry.File) {
			t.Errorf("Jingle[%q].File should remain absolute, got %q", name, entry.File)
		}
	}
	for name, entry := range profile.Assets.SE {
		if !filepath.IsAbs(entry.File) {
			t.Errorf("SE[%q].File should remain absolute, got %q", name, entry.File)
		}
	}
	for name, entry := range profile.Assets.BGM {
		if !filepath.IsAbs(entry.File) {
			t.Errorf("BGM[%q].File should remain absolute, got %q", name, entry.File)
		}
	}
}

func TestLoadProfile_MissingFile(t *testing.T) {
	_, err := config.LoadProfile("testdata/nonexistent.yaml")
	if err == nil {
		t.Error("expected error for missing file")
	}
}

func TestLoadConfig_Voicevox(t *testing.T) {
	cfg, err := config.LoadConfig("testdata/config.yaml")
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}
	if cfg.Voicevox.URL == "" {
		t.Error("Voicevox.URL must not be empty")
	}
}

func TestLoadConfig_Characters(t *testing.T) {
	cfg, err := config.LoadConfig("testdata/config.yaml")
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}
	if len(cfg.Characters) == 0 {
		t.Error("Characters must not be empty")
	}

	ch, ok := cfg.Characters["zundamon"]
	if !ok {
		t.Fatal("Characters[\"zundamon\"] not found")
	}
	if ch.Name == "" {
		t.Error("CharacterConfig.Name must not be empty")
	}
	if ch.Pronoun == "" {
		t.Error("CharacterConfig.Pronoun must not be empty")
	}
	if len(ch.SpeechSuffix) == 0 {
		t.Error("CharacterConfig.SpeechSuffix must not be empty")
	}
	if len(ch.Personality) == 0 {
		t.Error("CharacterConfig.Personality must not be empty")
	}
	if ch.DefaultStyle == "" {
		t.Error("CharacterConfig.DefaultStyle must not be empty")
	}
	if len(ch.Styles) == 0 {
		t.Error("CharacterConfig.Styles must not be empty")
	}
	if _, ok := ch.Styles[ch.DefaultStyle]; !ok {
		t.Errorf("DefaultStyle %q not found in Styles", ch.DefaultStyle)
	}
}

func TestLoadConfig_ValidationError_DefaultStyleNotInStyles(t *testing.T) {
	_, err := config.LoadConfig("testdata/config_invalid_default_style.yaml")
	if err == nil {
		t.Error("expected error when default_style is not in styles")
	}
}
