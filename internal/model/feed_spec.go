package model

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

const DefaultPublicDir = "public"

// FeedConfig holds RSS feed metadata for feed-spec.yaml.
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

// OutputConfig holds output settings for feed-spec.yaml.
type OutputConfig struct {
	Public string `yaml:"public"`
}

// FeedSpec is the top-level structure for feed-spec.yaml.
type FeedSpec struct {
	ProgramID string       `yaml:"program_id"`
	Feed      FeedConfig   `yaml:"feed"`
	Output    OutputConfig `yaml:"output"`
}

// EffectivePublicDir returns the configured public directory or DefaultPublicDir if not set.
func (c FeedSpec) EffectivePublicDir() string {
	if c.Output.Public == "" {
		return DefaultPublicDir
	}
	return c.Output.Public
}

// LoadFeedSpec reads and parses a feed-spec.yaml file.
func LoadFeedSpec(path string) (FeedSpec, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return FeedSpec{}, fmt.Errorf("read feed spec: %w", err)
	}
	var cfg FeedSpec
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return FeedSpec{}, fmt.Errorf("parse feed spec: %w", err)
	}
	return cfg, nil
}
