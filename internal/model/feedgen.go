package model

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

const DefaultPublicDir = "public"

// FeedConfig holds RSS feed metadata for feedgen.yaml.
type FeedConfig struct {
	Language         string `yaml:"language"`
	Author           string `yaml:"author"`
	Email            string `yaml:"email"`
	Category         string `yaml:"category"`
	Explicit         bool   `yaml:"explicit"`
	CoverImageURL    string `yaml:"cover_image_url"`
	SiteURL          string `yaml:"site_url"`
	AudioURLTemplate string `yaml:"audio_url_template"`
	Credit           string `yaml:"credit"`
}

// OutputConfig holds output settings for feedgen.yaml.
type OutputConfig struct {
	Public string `yaml:"public"`
}

// FeedgenConfig is the top-level structure for feedgen.yaml.
type FeedgenConfig struct {
	ProgramID string       `yaml:"program_id"`
	Feed      FeedConfig   `yaml:"feed"`
	Output    OutputConfig `yaml:"output"`
}

// EffectivePublicDir returns the configured public directory or DefaultPublicDir if not set.
func (c FeedgenConfig) EffectivePublicDir() string {
	if c.Output.Public == "" {
		return DefaultPublicDir
	}
	return c.Output.Public
}

// LoadFeedgen reads and parses a feedgen.yaml file.
func LoadFeedgen(path string) (FeedgenConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return FeedgenConfig{}, fmt.Errorf("read feedgen config: %w", err)
	}
	var cfg FeedgenConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return FeedgenConfig{}, fmt.Errorf("parse feedgen config: %w", err)
	}
	return cfg, nil
}
