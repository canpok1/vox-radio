package config

import (
	"fmt"
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
// Relative asset file paths and file:// URLs in corners[].source are resolved
// relative to the spec file's directory.
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
		for j := range corner.Source {
			resolved, err := resolveFileURL(specDir, corner.Source[j].URL)
			if err != nil {
				return nil, err
			}
			corner.Source[j].URL = resolved
			if corner.Source[j].Path != "" {
				absPath, err := filepath.Abs(resolveFile(specDir, corner.Source[j].Path))
				if err != nil {
					return nil, fmt.Errorf("resolve path %q: %w", corner.Source[j].Path, err)
				}
				corner.Source[j].Path = absPath
			}
		}
	}
	resolveCornerDefaults(p)
	return p, nil
}

// resolveCornerDefaults applies CornerDefaults to each corner that does not have
// an explicit value set. Must be called after assets are merged.
// After this call, an AudioRef with empty Type is normalized to nil (explicit disable).
func resolveCornerDefaults(p *EpisodeSpec) {
	if p.CornerDefaults == nil {
		return
	}
	d := p.CornerDefaults
	for i := range p.Corners {
		c := &p.Corners[i]
		if c.BGM == nil {
			c.BGM = d.BGM
		}
		if c.StartAudio == nil {
			c.StartAudio = d.StartAudio
		}
		if c.StartAudio != nil && c.StartAudio.Type == "" {
			c.StartAudio = nil
		}
		if c.EndAudio == nil {
			c.EndAudio = d.EndAudio
		}
		if c.EndAudio != nil && c.EndAudio.Type == "" {
			c.EndAudio = nil
		}
		if c.StartPauseSec == nil {
			c.StartPauseSec = d.StartPauseSec
		}
		if c.EndPauseSec == nil {
			c.EndPauseSec = d.EndPauseSec
		}
	}
}

// LoadEpisodeSpecStrict loads episode-specific settings from the given YAML file path with strict parsing.
// Unknown keys in the YAML will cause an error (detects typos).
// Relative asset file paths and file:// URLs in corners[].source are resolved
// relative to the spec file's directory.
func LoadEpisodeSpecStrict(path string) (*EpisodeSpec, error) {
	return loadEpisodeSpecWith(path, true)
}
