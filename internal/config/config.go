package config

import (
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"slices"

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
	TrimSilence *bool   `yaml:"trim_silence,omitempty"`
	Description string  `yaml:"description,omitempty"`
}

// EffectiveTrimSilence returns true when TrimSilence is nil (default on) or explicitly true.
func (e JingleEntry) EffectiveTrimSilence() bool { return effectiveTrimSilence(e.TrimSilence) }

type SEEntry struct {
	File        string  `yaml:"file"`
	Volume      float64 `yaml:"volume"`
	TrimSilence *bool   `yaml:"trim_silence,omitempty"`
	Overlay     *bool   `yaml:"overlay,omitempty"` // nil=false (sequential); true=overlay on speech track
	Description string  `yaml:"description,omitempty"`
}

// EffectiveTrimSilence returns true when TrimSilence is nil (default on) or explicitly true.
func (e SEEntry) EffectiveTrimSilence() bool { return effectiveTrimSilence(e.TrimSilence) }

// EffectiveOverlay returns true only when Overlay is explicitly true.
// Default (nil) is false: the SE plays to completion before the next dialogue.
func (e SEEntry) EffectiveOverlay() bool { return e.Overlay != nil && *e.Overlay }

// effectiveTrimSilence returns true when v is nil (default) or points to true.
func effectiveTrimSilence(v *bool) bool {
	if v == nil {
		return true
	}
	return *v
}

// DefaultBGMFadeSec is the default fade-in/out duration (seconds) for BGM when not explicitly specified.
const DefaultBGMFadeSec = 1.0

type BGMEntry struct {
	File        string   `yaml:"file"`
	Volume      float64  `yaml:"volume"`
	DuckRatio   float64  `yaml:"duck_ratio"`
	Loop        bool     `yaml:"loop"`
	FadeIn      *float64 `yaml:"fade_in,omitempty"`
	FadeOut     *float64 `yaml:"fade_out,omitempty"`
	Description string   `yaml:"description,omitempty"`
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

// EffectiveFadeIn returns the fade-in duration in seconds.
// nil (unspecified) → DefaultBGMFadeSec; negative values are clamped to 0.
func (e BGMEntry) EffectiveFadeIn() float64 { return effectiveFadeSec(e.FadeIn) }

// EffectiveFadeOut returns the fade-out duration in seconds.
// nil (unspecified) → DefaultBGMFadeSec; negative values are clamped to 0.
func (e BGMEntry) EffectiveFadeOut() float64 { return effectiveFadeSec(e.FadeOut) }

type AssetsConfig struct {
	Jingle map[string]JingleEntry `yaml:"jingle"`
	SE     map[string]SEEntry     `yaml:"se"`
	BGM    map[string]BGMEntry    `yaml:"bgm"`
}

const (
	// DefaultProvider is the default LLM provider when none is specified.
	DefaultProvider = "openai"
	// ProviderDifyChat selects the Dify chat-messages endpoint.
	ProviderDifyChat = "dify-chat"
	// DefaultDifyUser is the default user identifier sent to Dify when not configured.
	DefaultDifyUser = "vox-radio"
	// DifyTemperaturePlaceholder is the placeholder in inputs values replaced with the effective temperature.
	DifyTemperaturePlaceholder = "${temperature}"
)

type LLMStepConfig struct {
	Temperature *float64 `yaml:"temperature,omitempty"`
}

// OpenAIConfig holds connection settings for the OpenAI-compatible provider.
type OpenAIConfig struct {
	BaseURL   string `yaml:"base_url"`
	APIKeyEnv string `yaml:"api_key_env"`
	Model     string `yaml:"model"`
}

// DifyChatConfig holds connection settings for the Dify chat-messages provider.
type DifyChatConfig struct {
	BaseURL   string            `yaml:"base_url"`
	APIKeyEnv string            `yaml:"api_key_env"`
	User      string            `yaml:"user,omitempty"`
	Inputs    map[string]string `yaml:"inputs,omitempty"`
}

type LLMConfig struct {
	Provider             string                   `yaml:"provider"`
	Temperature          float64                  `yaml:"temperature"`
	MaxRetries           int                      `yaml:"max_retries"`
	MinRequestIntervalMS *int                     `yaml:"min_request_interval_ms,omitempty"`
	Steps                map[string]LLMStepConfig `yaml:"steps"`
	OpenAI               *OpenAIConfig            `yaml:"openai,omitempty"`
	DifyChat             *DifyChatConfig          `yaml:"dify-chat,omitempty"`
}

// EffectiveProvider returns the resolved provider name.
// Empty string defaults to DefaultProvider ("openai").
func (c LLMConfig) EffectiveProvider() string {
	if c.Provider == "" {
		return DefaultProvider
	}
	return c.Provider
}

// EffectiveMinRequestIntervalMS returns the resolved minimum request interval in milliseconds.
// Nil (unspecified in YAML) returns DefaultMinRequestIntervalMS; explicit 0 disables throttling.
func (c LLMConfig) EffectiveMinRequestIntervalMS() int {
	if c.MinRequestIntervalMS == nil {
		return DefaultMinRequestIntervalMS
	}
	return *c.MinRequestIntervalMS
}

// DefaultProgramSummaryLength is the default summary length (chars) for program-wide summaries.
const DefaultProgramSummaryLength = 200

// DefaultCornerSummaryLength is the default summary length (chars) for per-corner summaries.
const DefaultCornerSummaryLength = 100

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
	ID            string `yaml:"id,omitempty"`
	Title         string `yaml:"title"`
	Description   string `yaml:"description"`
	SummaryLength int    `yaml:"summary_length,omitempty"`
}

// EffectiveSummaryLength returns the configured SummaryLength, falling back to DefaultProgramSummaryLength.
func (p ProgramConfig) EffectiveSummaryLength() int {
	if p.SummaryLength <= 0 {
		return DefaultProgramSummaryLength
	}
	return p.SummaryLength
}

// SourceConfig defines the data sources for a corner (feeds and individual article URLs).
type SourceConfig struct {
	Feeds    []FeedEntry `yaml:"feeds"`
	Articles []string    `yaml:"articles"`
}

// CornerConfig defines a fixed corner in the program structure.
type CornerConfig struct {
	Title         string            `yaml:"title"`
	Content       string            `yaml:"content"`
	Direction     string            `yaml:"direction,omitempty"`
	Cast          map[string]string `yaml:"cast"`
	LengthSec     int               `yaml:"length_sec"`
	SummaryLength int               `yaml:"summary_length,omitempty"`
	Source        *SourceConfig     `yaml:"source,omitempty"`
	StartJingle   string            `yaml:"start_jingle,omitempty"`
	EndJingle     string            `yaml:"end_jingle,omitempty"`
	BGM           string            `yaml:"bgm,omitempty"`
	StartPauseSec float64           `yaml:"start_pause_sec,omitempty"`
	EndPauseSec   float64           `yaml:"end_pause_sec,omitempty"`
}

// EffectiveSummaryLength returns the configured SummaryLength, falling back to DefaultCornerSummaryLength.
func (c CornerConfig) EffectiveSummaryLength() int {
	if c.SummaryLength <= 0 {
		return DefaultCornerSummaryLength
	}
	return c.SummaryLength
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

// GuestCondition は出現条件。将来 probability 等を後方互換で追加可能。
type GuestCondition struct {
	Episodes []int `yaml:"episodes,omitempty"` // この回番号で出演（明示リスト）
	Every    int   `yaml:"every,omitempty"`    // N の倍数回で出演（0 で無効）
}

// Matches は episodeNumber が条件に合致するか判定する（論理和）。
func (c GuestCondition) Matches(episodeNumber int) bool {
	if episodeNumber <= 0 {
		return false
	}
	if slices.Contains(c.Episodes, episodeNumber) {
		return true
	}
	if c.Every > 0 && episodeNumber%c.Every == 0 {
		return true
	}
	return false
}

// GuestConfig はゲスト1人分の設定（キャラIDは map のキーで持つため持たない）。
type GuestConfig struct {
	Role      string         `yaml:"role"`
	Condition GuestCondition `yaml:"condition"`
}

// EpisodeSpec holds episode-specific settings (program, corners, assets).
// It is loaded from episode-spec.yaml.
// Data sources (feeds, articles) are defined per-corner in corners[].source.
// Assets are loaded from the files listed in AssetsFiles and merged into Assets.
type EpisodeSpec struct {
	Program     ProgramConfig          `yaml:"program"`
	Corners     []CornerConfig         `yaml:"corners"`
	Guests      map[string]GuestConfig `yaml:"guests,omitempty"`
	AssetsFiles []string               `yaml:"assets_files"`
	Assets      AssetsConfig           `yaml:"-"`
}

// ValidateEpisodeSpecGuests checks that every guest character ID exists in chars,
// and that each guest's condition has at least one of episodes or every set.
func ValidateEpisodeSpecGuests(p *EpisodeSpec, chars map[string]CharacterConfig) error {
	for charID, g := range p.Guests {
		if _, ok := chars[charID]; !ok {
			return fmt.Errorf("guests[%q]: character not found in characters catalog", charID)
		}
		cond := g.Condition
		for _, e := range cond.Episodes {
			if e < 1 {
				return fmt.Errorf("guests[%q].condition.episodes: value %d must be >= 1", charID, e)
			}
		}
		if cond.Every < 0 {
			return fmt.Errorf("guests[%q].condition.every: value %d must be >= 1", charID, cond.Every)
		}
		if len(cond.Episodes) == 0 && cond.Every == 0 {
			return fmt.Errorf("guests[%q].condition: at least one of episodes or every must be set", charID)
		}
	}
	return nil
}

// CornerSummaryLength returns the effective summary length (chars) for the corner matching title.
// Falls back to DefaultCornerSummaryLength when the corner is not found or summary_length is unset.
func (p *EpisodeSpec) CornerSummaryLength(title string) int {
	for _, c := range p.Corners {
		if c.Title == title {
			return c.EffectiveSummaryLength()
		}
	}
	return DefaultCornerSummaryLength
}

// LoadConfig loads common settings from the given YAML file path.
func LoadConfig(path string) (*Config, error) {
	return loadConfigWith(path, false)
}

func loadConfigWith(path string, strict bool) (*Config, error) {
	cfg := &Config{}
	if err := decodeYAML(path, cfg, strict); err != nil {
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
	if err := validateLLMConfig(&cfg.LLM); err != nil {
		return err
	}
	return nil
}

func validateLLMConfig(cfg *LLMConfig) error {
	switch cfg.EffectiveProvider() {
	case DefaultProvider:
		if cfg.OpenAI == nil {
			return fmt.Errorf("llm.openai block is required when provider is %q", DefaultProvider)
		}
		if cfg.OpenAI.BaseURL == "" {
			return fmt.Errorf("llm.openai.base_url is required")
		}
		if cfg.OpenAI.APIKeyEnv == "" {
			return fmt.Errorf("llm.openai.api_key_env is required")
		}
	case ProviderDifyChat:
		if cfg.DifyChat == nil {
			return fmt.Errorf("llm.dify-chat block is required when provider is %q", ProviderDifyChat)
		}
		if cfg.DifyChat.BaseURL == "" {
			return fmt.Errorf("llm.dify-chat.base_url is required")
		}
		if cfg.DifyChat.APIKeyEnv == "" {
			return fmt.Errorf("llm.dify-chat.api_key_env is required")
		}
	default:
		return fmt.Errorf("unknown llm.provider %q", cfg.Provider)
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

// LoadEpisodeSpec loads episode-specific settings from the given YAML file path.
// Relative asset file paths are resolved relative to the spec file's directory.
func LoadEpisodeSpec(path string) (*EpisodeSpec, error) {
	return loadEpisodeSpecWith(path, false)
}

func loadEpisodeSpecWith(path string, strict bool) (*EpisodeSpec, error) {
	p := &EpisodeSpec{}
	if err := decodeYAML(path, p, strict); err != nil {
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
	return p, nil
}

func loadAssetsFile(path string, strict bool) (AssetsConfig, error) {
	var assets AssetsConfig
	if err := decodeYAML(path, &assets, strict); err != nil {
		return AssetsConfig{}, fmt.Errorf("loading asset file %q: %w", path, err)
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

// ValidateAssetsConfig checks that all referenced files exist and that field values are in valid ranges.
func ValidateAssetsConfig(assets *AssetsConfig) error {
	for name, entry := range assets.Jingle {
		if err := validateFileField("jingle", name, entry.File); err != nil {
			return err
		}
		if entry.FadeIn < 0 {
			return fmt.Errorf("jingle[%q].fade_in: must be >= 0, got %v", name, entry.FadeIn)
		}
		if entry.FadeOut < 0 {
			return fmt.Errorf("jingle[%q].fade_out: must be >= 0, got %v", name, entry.FadeOut)
		}
	}
	for name, entry := range assets.SE {
		if err := validateFileField("se", name, entry.File); err != nil {
			return err
		}
		if entry.Volume < 0 {
			return fmt.Errorf("se[%q].volume: must be >= 0, got %v", name, entry.Volume)
		}
	}
	for name, entry := range assets.BGM {
		if err := validateFileField("bgm", name, entry.File); err != nil {
			return err
		}
		if entry.Volume < 0 {
			return fmt.Errorf("bgm[%q].volume: must be >= 0, got %v", name, entry.Volume)
		}
		if entry.DuckRatio < 1 {
			return fmt.Errorf("bgm[%q].duck_ratio: must be >= 1, got %v", name, entry.DuckRatio)
		}
		if entry.FadeIn != nil && *entry.FadeIn < 0 {
			return fmt.Errorf("bgm[%q].fade_in: must be >= 0, got %v", name, *entry.FadeIn)
		}
		if entry.FadeOut != nil && *entry.FadeOut < 0 {
			return fmt.Errorf("bgm[%q].fade_out: must be >= 0, got %v", name, *entry.FadeOut)
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

// ValidateEpisodeSpecCast checks that every character ID in corners[].cast exists in chars.
func ValidateEpisodeSpecCast(p *EpisodeSpec, chars map[string]CharacterConfig) error {
	for _, corner := range p.Corners {
		for charID := range corner.Cast {
			if _, ok := chars[charID]; !ok {
				return fmt.Errorf("corners[%q].cast: unknown character %q", corner.Title, charID)
			}
		}
	}
	return nil
}

// ValidateEpisodeSpecAssets checks that corner-level jingle/bgm keys reference existing assets.
func ValidateEpisodeSpecAssets(p *EpisodeSpec) error {
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
	return loadConfigWith(path, true)
}

// LoadEpisodeSpecStrict loads episode-specific settings from the given YAML file path with strict parsing.
// Unknown keys in the YAML will cause an error (detects typos).
// Relative asset file paths are resolved relative to the spec file's directory.
func LoadEpisodeSpecStrict(path string) (*EpisodeSpec, error) {
	return loadEpisodeSpecWith(path, true)
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
