package config_test

import (
	"path/filepath"
	"testing"

	"github.com/canpok1/vox-radio/internal/config"
)

func TestDurationSecToTargetChars(t *testing.T) {
	tests := []struct {
		sec  int
		want int
	}{
		{sec: 0, want: 0},
		{sec: 1, want: 7},
		{sec: 14, want: 98},
		{sec: 30, want: 210},
	}
	for _, tt := range tests {
		got := config.DurationSecToTargetChars(tt.sec)
		if got != tt.want {
			t.Errorf("DurationSecToTargetChars(%d) = %d, want %d", tt.sec, got, tt.want)
		}
	}
}

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

	t.Run("Program", func(t *testing.T) {
		if profile.Program.Title == "" {
			t.Error("Program.Title must not be empty")
		}
		if profile.Program.Language == "" {
			t.Error("Program.Language must not be empty")
		}
		if profile.Program.MaxItems <= 0 {
			t.Error("Program.MaxItems must be positive")
		}
		if profile.Program.SegmentPauseSec <= 0 {
			t.Error("Program.SegmentPauseSec must be positive")
		}
	})

	t.Run("Corners", func(t *testing.T) {
		if len(profile.Corners) == 0 {
			t.Error("Corners must not be empty")
		}
		c := profile.Corners[0]
		if c.Title == "" {
			t.Error("Corners[0].Title must not be empty")
		}
		if c.Content == "" {
			t.Error("Corners[0].Content must not be empty")
		}
		if len(c.Cast) == 0 {
			t.Error("Corners[0].Cast must not be empty")
		}
		if c.TargetDurationSec <= 0 {
			t.Error("Corners[0].TargetDurationSec must be positive")
		}
	})

	t.Run("ProgramTargetDurationSec", func(t *testing.T) {
		if profile.Program.TargetDurationSec <= 0 {
			t.Error("Program.TargetDurationSec must be positive")
		}
	})

	t.Run("CornerSource", func(t *testing.T) {
		var sourceCorner *config.CornerConfig
		for i := range profile.Corners {
			if profile.Corners[i].Source != nil {
				sourceCorner = &profile.Corners[i]
				break
			}
		}
		if sourceCorner == nil {
			t.Fatal("no corner with source found")
		}
		if len(sourceCorner.Source.Feeds) == 0 {
			t.Error("Source.Feeds must not be empty")
		}
		if sourceCorner.Source.Feeds[0].URL == "" {
			t.Error("Source.Feeds[0].URL must not be empty")
		}
		if len(sourceCorner.Source.Articles) == 0 {
			t.Error("Source.Articles must not be empty")
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

func TestValidateProfileCast_Valid(t *testing.T) {
	p := &config.Profile{
		Corners: []config.CornerConfig{
			{Title: "opening", Cast: map[string]string{"zundamon": "司会"}},
		},
	}
	chars := map[string]config.CharacterConfig{
		"zundamon": {Name: "ずんだもん"},
	}
	if err := config.ValidateProfileCast(p, chars); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestValidateProfileCast_UnknownCharacter(t *testing.T) {
	p := &config.Profile{
		Corners: []config.CornerConfig{
			{Title: "opening", Cast: map[string]string{"unknown_char": "司会"}},
		},
	}
	chars := map[string]config.CharacterConfig{
		"zundamon": {Name: "ずんだもん"},
	}
	if err := config.ValidateProfileCast(p, chars); err == nil {
		t.Error("expected error for unknown character in cast")
	}
}
