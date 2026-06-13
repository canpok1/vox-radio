package config

import (
	"path/filepath"

	"github.com/canpok1/vox-radio/internal/fileio"
)

// LoadConfig loads common settings from the given YAML file path.
func LoadConfig(path string) (*Config, error) {
	return loadConfigWith(path, false)
}

func loadConfigWith(path string, strict bool) (*Config, error) {
	cfg := &Config{}
	if err := fileio.DecodeYAML(path, cfg, strict); err != nil {
		return nil, err
	}
	if err := validateConfig(cfg); err != nil {
		return nil, err
	}
	return cfg, nil
}

func validateConfig(cfg *Config) error {
	if err := validateCharacters(cfg.Characters); err != nil {
		return err
	}
	if err := validateVoicevoxPresets(cfg.Voicevox.Presets); err != nil {
		return err
	}
	if err := validateLLMConfig(&cfg.LLM); err != nil {
		return err
	}
	return validateSecurityConfig(&cfg.Security)
}

// LoadConfigStrict loads common settings from the given YAML file path with strict parsing.
// Unknown keys in the YAML will cause an error (detects typos).
func LoadConfigStrict(path string) (*Config, error) {
	return loadConfigWith(path, true)
}

// LoadEpisodeSpec loads episode-specific settings from the given YAML file path.
// Relative asset file paths are resolved relative to the spec file's directory.
func LoadEpisodeSpec(path string) (*EpisodeSpec, error) {
	return loadEpisodeSpecWith(path, false)
}

func loadEpisodeSpecWith(path string, strict bool) (*EpisodeSpec, error) {
	p := &EpisodeSpec{}
	if err := fileio.DecodeYAML(path, p, strict); err != nil {
		return nil, err
	}
	specDir := filepath.Dir(path)
	for _, assetsPath := range p.AssetsFiles {
		absPath := resolveFile(specDir, assetsPath)
		assets, err := loadAssetsFile(absPath, strict)
		if err != nil {
			return nil, err
		}
		mergeAssets(&p.Assets, &assets)
	}
	for i := range p.Corners {
		corner := &p.Corners[i]
		if corner.Source == nil {
			continue
		}
		for j := range corner.Source.Feeds {
			resolved, err := resolveFileURL(specDir, corner.Source.Feeds[j].URL)
			if err != nil {
				return nil, err
			}
			corner.Source.Feeds[j].URL = resolved
		}
		for j, article := range corner.Source.Articles {
			resolved, err := resolveFileURL(specDir, article)
			if err != nil {
				return nil, err
			}
			corner.Source.Articles[j] = resolved
		}
	}
	return p, nil
}

// LoadEpisodeSpecStrict loads episode-specific settings from the given YAML file path with strict parsing.
// Unknown keys in the YAML will cause an error (detects typos).
// Relative asset file paths are resolved relative to the spec file's directory.
func LoadEpisodeSpecStrict(path string) (*EpisodeSpec, error) {
	return loadEpisodeSpecWith(path, true)
}
