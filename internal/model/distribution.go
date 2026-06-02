package model

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

const DefaultPublicDir = "public"

// FeedConfig holds RSS feed metadata for distribution.yaml.
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

// OutputConfig holds output settings for distribution.yaml.
type OutputConfig struct {
	Public string `yaml:"public"`
}

// DistributionConfig is the top-level structure for distribution.yaml.
type DistributionConfig struct {
	ProgramID string       `yaml:"program_id"`
	Feed      FeedConfig   `yaml:"feed"`
	Output    OutputConfig `yaml:"output"`
}

// EffectivePublicDir returns the configured public directory or DefaultPublicDir if not set.
func (c DistributionConfig) EffectivePublicDir() string {
	if c.Output.Public == "" {
		return DefaultPublicDir
	}
	return c.Output.Public
}

// LoadDistribution reads and parses a distribution.yaml file.
func LoadDistribution(path string) (DistributionConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return DistributionConfig{}, fmt.Errorf("read distribution config: %w", err)
	}
	var cfg DistributionConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return DistributionConfig{}, fmt.Errorf("parse distribution config: %w", err)
	}
	return cfg, nil
}
