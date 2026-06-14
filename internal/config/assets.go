package config

import (
	"errors"
	"fmt"
	"io"
	"maps"
	"os"
	"path/filepath"

	"github.com/canpok1/vox-radio/internal/fileio"
)

const (
	// DefaultSilenceTrimThresholdDB is the default amplitude threshold (dB) below which
	// audio is treated as silence when trimming leading/trailing silence from jingle/SE.
	DefaultSilenceTrimThresholdDB = -50.0
	// DefaultBGMFadeSec is the default fade-in/out duration (seconds) for BGM when not explicitly specified.
	DefaultBGMFadeSec = 1.0
)

type JingleEntry struct {
	File                 string   `yaml:"file"`
	FadeIn               float64  `yaml:"fade_in"`
	FadeOut              float64  `yaml:"fade_out"`
	TrimSilence          *bool    `yaml:"trim_silence,omitempty"`
	TrimSilenceThreshold *float64 `yaml:"trim_silence_threshold,omitempty"` // dB; nil → DefaultSilenceTrimThresholdDB
	Description          string   `yaml:"description,omitempty"`
	Credit               string   `yaml:"credit,omitempty"`
}

// EffectiveTrimSilence returns true when TrimSilence is nil (default on) or explicitly true.
func (e JingleEntry) EffectiveTrimSilence() bool { return effectiveTrimSilence(e.TrimSilence) }

// EffectiveTrimSilenceThresholdDB returns the threshold in dB; nil → DefaultSilenceTrimThresholdDB.
func (e JingleEntry) EffectiveTrimSilenceThresholdDB() float64 {
	return effectiveTrimSilenceThresholdDB(e.TrimSilenceThreshold)
}

// Validate checks that field values are in valid ranges.
func (e JingleEntry) Validate() error {
	if err := validateNonNegative("fade_in", e.FadeIn); err != nil {
		return err
	}
	if err := validateNonNegative("fade_out", e.FadeOut); err != nil {
		return err
	}
	return validateOptionalNegative("trim_silence_threshold", e.TrimSilenceThreshold)
}

type SEEntry struct {
	File                 string   `yaml:"file"`
	Volume               float64  `yaml:"volume"`
	TrimSilence          *bool    `yaml:"trim_silence,omitempty"`
	TrimSilenceThreshold *float64 `yaml:"trim_silence_threshold,omitempty"` // dB; nil → DefaultSilenceTrimThresholdDB
	Overlay              *bool    `yaml:"overlay,omitempty"`                // nil=false (sequential); true=overlay on speech track
	Description          string   `yaml:"description,omitempty"`
	Credit               string   `yaml:"credit,omitempty"`
}

// EffectiveTrimSilence returns true when TrimSilence is nil (default on) or explicitly true.
func (e SEEntry) EffectiveTrimSilence() bool { return effectiveTrimSilence(e.TrimSilence) }

// EffectiveTrimSilenceThresholdDB returns the threshold in dB; nil → DefaultSilenceTrimThresholdDB.
func (e SEEntry) EffectiveTrimSilenceThresholdDB() float64 {
	return effectiveTrimSilenceThresholdDB(e.TrimSilenceThreshold)
}

// EffectiveOverlay returns true only when Overlay is explicitly true.
// Default (nil) is false: the SE plays to completion before the next dialogue.
func (e SEEntry) EffectiveOverlay() bool { return e.Overlay != nil && *e.Overlay }

// Validate checks that field values are in valid ranges.
func (e SEEntry) Validate() error {
	if err := validateNonNegative("volume", e.Volume); err != nil {
		return err
	}
	return validateOptionalNegative("trim_silence_threshold", e.TrimSilenceThreshold)
}

type BGMEntry struct {
	File        string   `yaml:"file"`
	Volume      float64  `yaml:"volume"`
	DuckRatio   float64  `yaml:"duck_ratio"`
	Loop        bool     `yaml:"loop"`
	FadeIn      *float64 `yaml:"fade_in,omitempty"`
	FadeOut     *float64 `yaml:"fade_out,omitempty"`
	Description string   `yaml:"description,omitempty"`
	Credit      string   `yaml:"credit,omitempty"`
}

// EffectiveFadeIn returns the fade-in duration in seconds.
// nil (unspecified) → DefaultBGMFadeSec; negative values are clamped to 0.
func (e BGMEntry) EffectiveFadeIn() float64 { return effectiveFadeSec(e.FadeIn) }

// EffectiveFadeOut returns the fade-out duration in seconds.
// nil (unspecified) → DefaultBGMFadeSec; negative values are clamped to 0.
func (e BGMEntry) EffectiveFadeOut() float64 { return effectiveFadeSec(e.FadeOut) }

// Validate checks that field values are in valid ranges.
func (e BGMEntry) Validate() error {
	if err := validateNonNegative("volume", e.Volume); err != nil {
		return err
	}
	if e.DuckRatio < 1 {
		return fmt.Errorf("duck_ratio: must be >= 1, got %v", e.DuckRatio)
	}
	if err := validateOptionalNonNegative("fade_in", e.FadeIn); err != nil {
		return err
	}
	return validateOptionalNonNegative("fade_out", e.FadeOut)
}

type AssetsConfig struct {
	Jingle map[string]JingleEntry `yaml:"jingle"`
	SE     map[string]SEEntry     `yaml:"se"`
	BGM    map[string]BGMEntry    `yaml:"bgm"`
}

// effectiveTrimSilence returns true when v is nil (default) or points to true.
func effectiveTrimSilence(v *bool) bool {
	if v == nil {
		return true
	}
	return *v
}

// effectiveTrimSilenceThresholdDB resolves a *float64 threshold: nil → DefaultSilenceTrimThresholdDB.
func effectiveTrimSilenceThresholdDB(v *float64) float64 {
	if v == nil {
		return DefaultSilenceTrimThresholdDB
	}
	return *v
}

// effectiveFadeSec resolves a *float64 fade setting: nil → DefaultBGMFadeSec, negative → 0.
func effectiveFadeSec(v *float64) float64 {
	if v == nil {
		return DefaultBGMFadeSec
	}
	if *v < 0 {
		return 0
	}
	return *v
}

// validateNonNegative returns an error if v < 0.
func validateNonNegative(field string, v float64) error {
	if v < 0 {
		return fmt.Errorf("%s: must be >= 0, got %v", field, v)
	}
	return nil
}

// validateOptionalNonNegative returns an error if v is non-nil and *v < 0.
func validateOptionalNonNegative(field string, v *float64) error {
	if v != nil && *v < 0 {
		return fmt.Errorf("%s: must be >= 0, got %v", field, *v)
	}
	return nil
}

// validateFileField checks that file is non-empty and exists on disk.
func validateFileField(category, name, file string) error {
	if file == "" {
		return fmt.Errorf("%s[%q].file: must not be empty", category, name)
	}
	if _, err := os.Stat(file); err != nil {
		return fmt.Errorf("%s[%q].file: %w", category, name, err)
	}
	return nil
}

// validateOptionalNegative returns an error if v is non-nil and *v >= 0.
func validateOptionalNegative(field string, v *float64) error {
	if v != nil && *v >= 0 {
		return fmt.Errorf("%s: must be < 0 (dB), got %v", field, *v)
	}
	return nil
}

// ValidateAssetsConfig checks that all referenced files exist and that field values are in valid ranges.
func ValidateAssetsConfig(assets *AssetsConfig) error {
	for name, entry := range assets.Jingle {
		if err := validateFileField("jingle", name, entry.File); err != nil {
			return err
		}
		if err := entry.Validate(); err != nil {
			return fmt.Errorf("jingle[%q]: %w", name, err)
		}
	}
	for name, entry := range assets.SE {
		if err := validateFileField("se", name, entry.File); err != nil {
			return err
		}
		if err := entry.Validate(); err != nil {
			return fmt.Errorf("se[%q]: %w", name, err)
		}
	}
	for name, entry := range assets.BGM {
		if err := validateFileField("bgm", name, entry.File); err != nil {
			return err
		}
		if err := entry.Validate(); err != nil {
			return fmt.Errorf("bgm[%q]: %w", name, err)
		}
	}
	return nil
}

func mergeAssets(dst, src *AssetsConfig) {
	if src.Jingle != nil {
		if dst.Jingle == nil {
			dst.Jingle = make(map[string]JingleEntry, len(src.Jingle))
		}
		maps.Copy(dst.Jingle, src.Jingle)
	}
	if src.SE != nil {
		if dst.SE == nil {
			dst.SE = make(map[string]SEEntry, len(src.SE))
		}
		maps.Copy(dst.SE, src.SE)
	}
	if src.BGM != nil {
		if dst.BGM == nil {
			dst.BGM = make(map[string]BGMEntry, len(src.BGM))
		}
		maps.Copy(dst.BGM, src.BGM)
	}
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

func loadAssetsFile(path string, strict bool) (AssetsConfig, error) {
	var assets AssetsConfig
	if err := fileio.DecodeYAML(path, &assets, strict); err != nil {
		if errors.Is(err, io.EOF) {
			return assets, nil
		}
		return AssetsConfig{}, fmt.Errorf("loading asset file: %w", err)
	}
	resolveAssetPaths(filepath.Dir(path), &assets)
	return assets, nil
}

// LoadAssetsFileStrict loads an assets configuration file with strict parsing.
// Unknown keys in the YAML will cause an error (detects typos).
// Relative file paths are resolved relative to the assets file's directory.
func LoadAssetsFileStrict(path string) (AssetsConfig, error) {
	return loadAssetsFile(path, true)
}
