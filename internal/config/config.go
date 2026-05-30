package config

import (
	"fmt"
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

type Config struct {
	Feeds   FeedsConfig
	Show    model.ShowConfig
	Assets  AssetsConfig
	LLM     LLMConfig
	Podcast PodcastConfig
}

func Load(dir string) (*Config, error) {
	cfg := &Config{}
	loaders := []struct {
		name string
		dest any
	}{
		{"feeds.yaml", &cfg.Feeds},
		{"show.yaml", &cfg.Show},
		{"assets.yaml", &cfg.Assets},
		{"llm.yaml", &cfg.LLM},
		{"podcast.yaml", &cfg.Podcast},
	}
	for _, l := range loaders {
		if err := loadYAML(filepath.Join(dir, l.name), l.dest); err != nil {
			return nil, fmt.Errorf("loading %s: %w", l.name, err)
		}
	}
	return cfg, nil
}

func loadYAML(path string, dest any) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer func() { _ = f.Close() }()
	return yaml.NewDecoder(f).Decode(dest)
}
