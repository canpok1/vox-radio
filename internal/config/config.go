package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// CharsPerSec is the approximate number of characters spoken per second,
// used to convert length_sec to a target character count.
const CharsPerSec = 7

// DefaultMinRequestIntervalMS is the default minimum interval (ms) between LLM API requests.
// Based on gemini-3.1-flash-lite free tier (15 RPM) with ~10% safety margin: 60000/15 * 1.1 ≈ 4500.
const DefaultMinRequestIntervalMS = 4500

// DurationSecToTargetChars converts a duration in seconds to an approximate target character count.
func DurationSecToTargetChars(sec int) int {
	return sec * CharsPerSec
}

type FeedEntry struct {
	URL      string `yaml:"url"`
	MaxItems int    `yaml:"max_items"`
}

type FeedsConfig struct {
	Feeds    []FeedEntry `yaml:"feeds"`
	Articles []string    `yaml:"articles"`
}

type JingleEntry struct {
	File        string  `yaml:"file"`
	FadeIn      float64 `yaml:"fade_in"`
	FadeOut     float64 `yaml:"fade_out"`
	Description string  `yaml:"description,omitempty"`
}

type SEEntry struct {
	File        string  `yaml:"file"`
	Volume      float64 `yaml:"volume"`
	Description string  `yaml:"description,omitempty"`
}

type BGMEntry struct {
	File        string  `yaml:"file"`
	Volume      float64 `yaml:"volume"`
	DuckRatio   float64 `yaml:"duck_ratio"`
	Loop        bool    `yaml:"loop"`
	Description string  `yaml:"description,omitempty"`
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
	BaseURL              string                   `yaml:"base_url"`
	APIKeyEnv            string                   `yaml:"api_key_env"`
	Model                string                   `yaml:"model"`
	Temperature          float64                  `yaml:"temperature"`
	MaxRetries           int                      `yaml:"max_retries"`
	MinRequestIntervalMS *int                     `yaml:"min_request_interval_ms,omitempty"`
	Steps                map[string]LLMStepConfig `yaml:"steps"`
}

// EffectiveMinRequestIntervalMS returns the resolved minimum request interval in milliseconds.
// Nil (unspecified in YAML) returns DefaultMinRequestIntervalMS; explicit 0 disables throttling.
func (c LLMConfig) EffectiveMinRequestIntervalMS() int {
	if c.MinRequestIntervalMS == nil {
		return DefaultMinRequestIntervalMS
	}
	return *c.MinRequestIntervalMS
}

// DefaultCacheMaxEntries is the default maximum number of episodes to keep in the cache.
const DefaultCacheMaxEntries = 100

// DefaultCacheRetentionDays is the default number of days to retain cache entries.
const DefaultCacheRetentionDays = 90

// DefaultCacheLLMContextEntries is the default number of recent episodes to pass to the LLM.
const DefaultCacheLLMContextEntries = 10

// CacheConfig controls the episode history cache behavior.
type CacheConfig struct {
	Enabled           bool `yaml:"enabled"`
	MaxEntries        int  `yaml:"max_entries"`
	RetentionDays     int  `yaml:"retention_days"`
	LLMContextEntries int  `yaml:"llm_context_entries"`
}

// EffectiveMaxEntries returns the configured MaxEntries, falling back to DefaultCacheMaxEntries.
func (c CacheConfig) EffectiveMaxEntries() int {
	if c.MaxEntries <= 0 {
		return DefaultCacheMaxEntries
	}
	return c.MaxEntries
}

// EffectiveRetentionDays returns the configured RetentionDays, falling back to DefaultCacheRetentionDays.
func (c CacheConfig) EffectiveRetentionDays() int {
	if c.RetentionDays <= 0 {
		return DefaultCacheRetentionDays
	}
	return c.RetentionDays
}

// EffectiveLLMContextEntries returns the configured LLMContextEntries, falling back to DefaultCacheLLMContextEntries.
func (c CacheConfig) EffectiveLLMContextEntries() int {
	if c.LLMContextEntries <= 0 {
		return DefaultCacheLLMContextEntries
	}
	return c.LLMContextEntries
}

// ProgramConfig holds program-wide settings for content generation.
type ProgramConfig struct {
	ID              string  `yaml:"id,omitempty"`
	Title           string  `yaml:"title"`
	Description     string  `yaml:"description"`
	SegmentPauseSec float64 `yaml:"segment_pause_sec"`
	LengthSec       int     `yaml:"length_sec"`
}

// SourceConfig defines the data sources for a corner (feeds and individual article URLs).
type SourceConfig struct {
	Feeds    []FeedEntry `yaml:"feeds"`
	Articles []string    `yaml:"articles"`
}

// CornerConfig defines a fixed corner in the program structure.
type CornerConfig struct {
	Title       string            `yaml:"title"`
	Content     string            `yaml:"content"`
	Direction   string            `yaml:"direction,omitempty"`
	Cast        map[string]string `yaml:"cast"`
	LengthSec   int               `yaml:"length_sec"`
	Source      *SourceConfig     `yaml:"source,omitempty"`
	StartJingle string            `yaml:"start_jingle,omitempty"`
	EndJingle   string            `yaml:"end_jingle,omitempty"`
	BGM         string            `yaml:"bgm,omitempty"`
}

// VoicevoxPresets maps preset names to float64 scale values for each axis.
type VoicevoxPresets struct {
	Intonation map[string]float64 `yaml:"intonation"`
	Pitch      map[string]float64 `yaml:"pitch"`
	Speed      map[string]float64 `yaml:"speed"`
}

func resolvePreset(m map[string]float64, name string) (float64, bool) {
	if name == "" {
		return 0, false
	}
	v, ok := m[name]
	return v, ok
}

// ResolveIntonation returns (value, true) if name is non-empty and found in Intonation.
func (p VoicevoxPresets) ResolveIntonation(name string) (float64, bool) {
	return resolvePreset(p.Intonation, name)
}

// ResolvePitch returns (value, true) if name is non-empty and found in Pitch.
func (p VoicevoxPresets) ResolvePitch(name string) (float64, bool) {
	return resolvePreset(p.Pitch, name)
}

// ResolveSpeed returns (value, true) if name is non-empty and found in Speed.
func (p VoicevoxPresets) ResolveSpeed(name string) (float64, bool) {
	return resolvePreset(p.Speed, name)
}

var defaultIntonationPresets = map[string]float64{
	"棒読み":    0.0,
	"かなり控えめ": 0.3,
	"控えめ":    0.6,
	"標準":     1.0,
	"やや豊か":   1.2,
	"表現豊か":   1.5,
	"とても豊か":  1.8,
}

var defaultPitchPresets = map[string]float64{
	"低め":     -0.05,
	"やや低め":   -0.033,
	"わずかに低め": -0.017,
	"標準":     0.0,
	"わずかに高め": 0.017,
	"やや高め":   0.033,
	"高め":     0.05,
}

var defaultSpeedPresets = map[string]float64{
	"とてもゆっくり": 0.6,
	"ゆっくり":    0.8,
	"ややゆっくり":  0.9,
	"標準":      1.0,
	"やや早口":    1.1,
	"早口":      1.2,
	"とても早口":   1.4,
}

type VoicevoxConfig struct {
	URL     string           `yaml:"url"`
	Presets *VoicevoxPresets `yaml:"presets,omitempty"`
}

// EffectivePresets returns the configured presets, falling back per-axis to defaults when nil.
func (c VoicevoxConfig) EffectivePresets() VoicevoxPresets {
	if c.Presets == nil {
		return VoicevoxPresets{
			Intonation: defaultIntonationPresets,
			Pitch:      defaultPitchPresets,
			Speed:      defaultSpeedPresets,
		}
	}
	result := *c.Presets
	if result.Intonation == nil {
		result.Intonation = defaultIntonationPresets
	}
	if result.Pitch == nil {
		result.Pitch = defaultPitchPresets
	}
	if result.Speed == nil {
		result.Speed = defaultSpeedPresets
	}
	return result
}

type CharacterConfig struct {
	Name         string         `yaml:"name"`
	Pronoun      string         `yaml:"pronoun"`
	SpeechSuffix []string       `yaml:"speech_suffix"`
	Personality  []string       `yaml:"personality"`
	DefaultStyle string         `yaml:"default_style"`
	Styles       map[string]int `yaml:"styles"`
}

// DefaultSpeakerID returns the VOICEVOX speaker ID for the character's default style.
func (c CharacterConfig) DefaultSpeakerID() (int, bool) {
	if c.DefaultStyle == "" {
		return 0, false
	}
	id, ok := c.Styles[c.DefaultStyle]
	return id, ok
}

// SpeakerID returns the VOICEVOX speaker ID for the given style name.
// Falls back to the default style if style is empty or not found in Styles.
func (c CharacterConfig) SpeakerID(style string) (int, bool) {
	if style != "" {
		if id, ok := c.Styles[style]; ok {
			return id, true
		}
	}
	return c.DefaultSpeakerID()
}

// Config holds genre-independent common settings.
// It is loaded from vox-radio.yaml at the repository root.
type Config struct {
	LLM        LLMConfig                  `yaml:"llm"`
	Voicevox   VoicevoxConfig             `yaml:"voicevox"`
	Characters map[string]CharacterConfig `yaml:"characters"`
	Cache      CacheConfig                `yaml:"cache"`
}

// Profile holds genre-specific settings (program, corners, assets).
// It is loaded from sample-profiles/<genre>_profile.yaml.
// Data sources (feeds, articles) are defined per-corner in corners[].source.
type Profile struct {
	Program ProgramConfig  `yaml:"program"`
	Corners []CornerConfig `yaml:"corners"`
	Assets  AssetsConfig   `yaml:"assets"`
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
	if err := validateVoicevoxPresets(cfg.Voicevox.Presets); err != nil {
		return err
	}
	return nil
}

func validateVoicevoxPresets(p *VoicevoxPresets) error {
	if p == nil {
		return nil
	}
	axes := []struct {
		name string
		m    map[string]float64
		lo   float64
		hi   float64
	}{
		{"intonation", p.Intonation, 0.0, 2.0},
		{"pitch", p.Pitch, -0.15, 0.15},
		{"speed", p.Speed, 0.5, 2.0},
	}
	for _, ax := range axes {
		for name, v := range ax.m {
			if v < ax.lo || v > ax.hi {
				return fmt.Errorf("voicevox.presets.%s[%q]: value %g is out of range [%g, %g]", ax.name, name, v, ax.lo, ax.hi)
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

// ValidateProfileAssets checks that corner-level jingle/bgm keys reference existing assets.
func ValidateProfileAssets(p *Profile) error {
	for _, corner := range p.Corners {
		if corner.StartJingle != "" {
			if _, ok := p.Assets.Jingle[corner.StartJingle]; !ok {
				return fmt.Errorf("corners[%q].start_jingle: unknown jingle key %q", corner.Title, corner.StartJingle)
			}
		}
		if corner.EndJingle != "" {
			if _, ok := p.Assets.Jingle[corner.EndJingle]; !ok {
				return fmt.Errorf("corners[%q].end_jingle: unknown jingle key %q", corner.Title, corner.EndJingle)
			}
		}
		if corner.BGM != "" {
			if _, ok := p.Assets.BGM[corner.BGM]; !ok {
				return fmt.Errorf("corners[%q].bgm: unknown bgm key %q", corner.Title, corner.BGM)
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

// LoadConfigStrict loads common settings from the given YAML file path with strict parsing.
// Unknown keys in the YAML will cause an error (detects typos).
func LoadConfigStrict(path string) (*Config, error) {
	cfg := &Config{}
	if err := loadYAMLStrict(path, cfg); err != nil {
		return nil, err
	}
	if err := validateConfig(cfg); err != nil {
		return nil, err
	}
	return cfg, nil
}

// LoadProfileStrict loads genre-specific settings from the given YAML file path with strict parsing.
// Unknown keys in the YAML will cause an error (detects typos).
// Relative asset file paths are resolved relative to the profile file's directory.
func LoadProfileStrict(path string) (*Profile, error) {
	p := &Profile{}
	if err := loadYAMLStrict(path, p); err != nil {
		return nil, err
	}
	resolveAssetPaths(filepath.Dir(path), &p.Assets)
	return p, nil
}

func loadYAML(path string, dest any) error {
	return decodeYAML(path, dest, false)
}

func loadYAMLStrict(path string, dest any) error {
	return decodeYAML(path, dest, true)
}

func decodeYAML(path string, dest any, strict bool) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer func() { _ = f.Close() }()
	dec := yaml.NewDecoder(f)
	if strict {
		dec.KnownFields(true)
	}
	return dec.Decode(dest)
}
