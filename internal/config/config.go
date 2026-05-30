package config

import (
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"

	"github.com/canpok1/vox-radio/internal/model"
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

type PodcastConfig struct {
	Title         string `yaml:"title"`
	Description   string `yaml:"description"`
	Language      string `yaml:"language"`
	Author        string `yaml:"author"`
	Category      string `yaml:"category"`
	Explicit      bool   `yaml:"explicit"`
	CoverImageURL string `yaml:"cover_image_url"`
	SiteURL       string `yaml:"site_url"`
	MaxItems      int    `yaml:"max_items"`
}

// Config holds genre-independent common settings (LLM only).
// It is loaded from vox-radio.yaml at the repository root.
type Config struct {
	LLM LLMConfig `yaml:"llm"`
}

// Profile holds genre-specific settings (feeds, show, assets, podcast).
// It is loaded from profiles/<genre>/profile.yaml.
type Profile struct {
	Podcast  PodcastConfig    `yaml:"podcast"`
	Show     model.ShowConfig `yaml:"show"`
	Feeds    []FeedEntry      `yaml:"feeds"`
	Articles []string         `yaml:"articles"`
	Assets   AssetsConfig     `yaml:"assets"`
}

// LoadConfig loads common settings from the given YAML file path.
func LoadConfig(path string) (*Config, error) {
	cfg := &Config{}
	if err := loadYAML(path, cfg); err != nil {
		return nil, err
	}
	return cfg, nil
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
