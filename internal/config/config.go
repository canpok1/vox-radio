package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

type FeedEntry struct {
	URL      string `yaml:"url"`
	MaxItems int    `yaml:"max_items"`
}

type FeedsConfig struct {
	Feeds    []FeedEntry `yaml:"feeds"`
	Articles []string    `yaml:"articles"`
}

type JingleEntry struct {
	File    string  `yaml:"file"`
	FadeIn  float64 `yaml:"fade_in"`
	FadeOut float64 `yaml:"fade_out"`
}

type SEEntry struct {
	File   string  `yaml:"file"`
	Volume float64 `yaml:"volume"`
}

type BGMEntry struct {
	File      string  `yaml:"file"`
	Volume    float64 `yaml:"volume"`
	DuckRatio float64 `yaml:"duck_ratio"`
	Loop      bool    `yaml:"loop"`
}

type AssetsConfig struct {
	Jingle map[string]JingleEntry `yaml:"jingle"`
	SE     map[string]SEEntry     `yaml:"se"`
	BGM    map[string]BGMEntry    `yaml:"bgm"`
}

type LLMStepConfig struct {
	Temperature *float64 `yaml:"temperature,omitempty"`
}

type LLMConfig struct {
	BaseURL     string                   `yaml:"base_url"`
	APIKeyEnv   string                   `yaml:"api_key_env"`
	Model       string                   `yaml:"model"`
	Temperature float64                  `yaml:"temperature"`
	MaxRetries  int                      `yaml:"max_retries"`
	Steps       map[string]LLMStepConfig `yaml:"steps"`
}

// ProgramConfig holds program-wide settings (formerly PodcastConfig + SegmentPauseSec from ShowConfig).
type ProgramConfig struct {
	Title           string  `yaml:"title"`
	Description     string  `yaml:"description"`
	Language        string  `yaml:"language"`
	Author          string  `yaml:"author"`
	Category        string  `yaml:"category"`
	Explicit        bool    `yaml:"explicit"`
	CoverImageURL   string  `yaml:"cover_image_url"`
	SiteURL         string  `yaml:"site_url"`
	MaxItems        int     `yaml:"max_items"`
	SegmentPauseSec float64 `yaml:"segment_pause_sec"`
}

// CornerConfig defines a fixed corner in the program structure.
// target_chars is provisional; will be replaced by target_duration_sec in #4.
type CornerConfig struct {
	Title       string            `yaml:"title"`
	Content     string            `yaml:"content"`
	Cast        map[string]string `yaml:"cast"`
	TargetChars int               `yaml:"target_chars"`
}

type VoicevoxConfig struct {
	URL string `yaml:"url"`
}

type CharacterConfig struct {
	Name         string         `yaml:"name"`
	Pronoun      string         `yaml:"pronoun"`
	SpeechSuffix []string       `yaml:"speech_suffix"`
	Personality  []string       `yaml:"personality"`
	DefaultStyle string         `yaml:"default_style"`
	Styles       map[string]int `yaml:"styles"`
}

// Config holds genre-independent common settings.
// It is loaded from vox-radio.yaml at the repository root.
type Config struct {
	LLM        LLMConfig                  `yaml:"llm"`
	Voicevox   VoicevoxConfig             `yaml:"voicevox"`
	Characters map[string]CharacterConfig `yaml:"characters"`
}

// Profile holds genre-specific settings (feeds, program, corners, assets).
// It is loaded from profiles/<genre>/profile.yaml.
type Profile struct {
	Program  ProgramConfig  `yaml:"program"`
	Corners  []CornerConfig `yaml:"corners"`
	Feeds    []FeedEntry    `yaml:"feeds"`
	Articles []string       `yaml:"articles"`
	Assets   AssetsConfig   `yaml:"assets"`
}

// LoadConfig loads common settings from the given YAML file path.
func LoadConfig(path string) (*Config, error) {
	cfg := &Config{}
	if err := loadYAML(path, cfg); err != nil {
		return nil, err
	}
	if err := validateConfig(cfg); err != nil {
		return nil, err
	}
	return cfg, nil
}

func validateConfig(cfg *Config) error {
	for id, ch := range cfg.Characters {
		if ch.DefaultStyle != "" {
			if _, ok := ch.Styles[ch.DefaultStyle]; !ok {
				return fmt.Errorf("characters[%q].default_style %q not found in styles", id, ch.DefaultStyle)
			}
		}
	}
	return nil
}

// LoadProfile loads genre-specific settings from the given YAML file path.
// Relative asset file paths are resolved relative to the profile file's directory.
func LoadProfile(path string) (*Profile, error) {
	p := &Profile{}
	if err := loadYAML(path, p); err != nil {
		return nil, err
	}
	resolveAssetPaths(filepath.Dir(path), &p.Assets)
	return p, nil
}

// ValidateProfileCast checks that every character ID in corners[].cast exists in chars.
func ValidateProfileCast(p *Profile, chars map[string]CharacterConfig) error {
	for _, corner := range p.Corners {
		for charID := range corner.Cast {
			if _, ok := chars[charID]; !ok {
				return fmt.Errorf("corners[%q].cast: unknown character %q", corner.Title, charID)
			}
		}
	}
	return nil
}

func resolveAssetPaths(base string, assets *AssetsConfig) {
	for name, entry := range assets.Jingle {
		entry.File = resolveFile(base, entry.File)
		assets.Jingle[name] = entry
	}
	for name, entry := range assets.SE {
		entry.File = resolveFile(base, entry.File)
		assets.SE[name] = entry
	}
	for name, entry := range assets.BGM {
		entry.File = resolveFile(base, entry.File)
		assets.BGM[name] = entry
	}
}

func resolveFile(base, file string) string {
	if file != "" && !filepath.IsAbs(file) {
		return filepath.Join(base, file)
	}
	return file
}

func loadYAML(path string, dest any) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer func() { _ = f.Close() }()
	return yaml.NewDecoder(f).Decode(dest)
}
