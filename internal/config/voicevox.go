package config

import (
	"fmt"
	"os"
	"regexp"
	"sort"
	"strings"
	"time"
)

const (
	// DefaultVoicevoxURL は voicevox.url 未指定時のデフォルト URL。
	DefaultVoicevoxURL = "http://localhost:50021"
	// VoicevoxURLEnv は VOICEVOX URL を上書きする環境変数名（default エンジン専用）。
	VoicevoxURLEnv = "VOX_RADIO_VOICEVOX_URL"
	// DefaultEngineName は voicevox.engines 省略時・characters[].engine 省略時に使うエンジン名。
	DefaultEngineName = "default"
	// DefaultStartupTimeoutSeconds は startup_timeout_seconds 未指定時のデフォルト秒数。
	DefaultStartupTimeoutSeconds = 60
)

// engineNamePattern restricts voicevox.engines keys to a form that normalizes
// unambiguously to an environment variable name (see engineURLEnvName).
var engineNamePattern = regexp.MustCompile(`^[a-z0-9_-]+$`)

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

func presetNames(m map[string]float64) []string {
	names := make([]string, 0, len(m))
	for name := range m {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// IntonationNames returns the sorted preset names in Intonation.
func (p VoicevoxPresets) IntonationNames() []string {
	return presetNames(p.Intonation)
}

// PitchNames returns the sorted preset names in Pitch.
func (p VoicevoxPresets) PitchNames() []string {
	return presetNames(p.Pitch)
}

// SpeedNames returns the sorted preset names in Speed.
func (p VoicevoxPresets) SpeedNames() []string {
	return presetNames(p.Speed)
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

// VoicevoxEngineConfig is a single named VOICEVOX-compatible engine entry
// under voicevox.engines.
type VoicevoxEngineConfig struct {
	URL string `yaml:"url"`
}

type VoicevoxConfig struct {
	URL                   string                          `yaml:"url"`
	Engines               map[string]VoicevoxEngineConfig `yaml:"engines,omitempty"`
	Presets               *VoicevoxPresets                `yaml:"presets,omitempty"`
	StartupTimeoutSeconds *int                            `yaml:"startup_timeout_seconds,omitempty"`
}

// EffectiveStartupTimeout は起動待機タイムアウトを返す。nil（未指定）はデフォルト 60 秒、*0 は待機無効（0）。
func (c VoicevoxConfig) EffectiveStartupTimeout() time.Duration {
	if c.StartupTimeoutSeconds == nil {
		return DefaultStartupTimeoutSeconds * time.Second
	}
	return time.Duration(*c.StartupTimeoutSeconds) * time.Second
}

// engineURLEnvName returns the per-engine URL override env var name for an
// engine name: VOX_RADIO_VOICEVOX_URL_<NAME> with the name upper-cased and
// "-" normalized to "_".
func engineURLEnvName(name string) string {
	norm := strings.ToUpper(strings.ReplaceAll(name, "-", "_"))
	return VoicevoxURLEnv + "_" + norm
}

// resolveEngineURL resolves a single named engine's URL following the
// priority: ① VOX_RADIO_VOICEVOX_URL_<NAME> ② (default のみ) VOX_RADIO_VOICEVOX_URL
// ③ 設定値 ④ (default のみ) デフォルト定数。Returns "" if none apply (only
// possible for a non-default engine with no configured url and no override).
func (c VoicevoxConfig) resolveEngineURL(name string, ec VoicevoxEngineConfig) string {
	if v := os.Getenv(engineURLEnvName(name)); v != "" {
		return v
	}
	if name == DefaultEngineName {
		if v := os.Getenv(VoicevoxURLEnv); v != "" {
			return v
		}
	}
	if ec.URL != "" {
		return ec.URL
	}
	if name == DefaultEngineName {
		return DefaultVoicevoxURL
	}
	return ""
}

// EffectiveURLs resolves every configured VOICEVOX-compatible engine to its
// final URL, keyed by engine name. When voicevox.engines is not set, the
// single voicevox.url (or its env/default fallback) is returned as the
// implicit "default" engine, preserving pre-multi-engine behavior.
func (c VoicevoxConfig) EffectiveURLs() map[string]string {
	if len(c.Engines) == 0 {
		return map[string]string{DefaultEngineName: c.resolveEngineURL(DefaultEngineName, VoicevoxEngineConfig{URL: c.URL})}
	}
	result := make(map[string]string, len(c.Engines))
	for name, ec := range c.Engines {
		result[name] = c.resolveEngineURL(name, ec)
	}
	return result
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

// validateVoicevoxEngines checks voicevox.url / voicevox.engines for mutual
// exclusion, engine name format, required urls (env override may substitute),
// and normalized-env-name collisions between engine names.
func validateVoicevoxEngines(c VoicevoxConfig) error {
	if len(c.Engines) == 0 {
		return nil
	}
	if c.URL != "" {
		return fmt.Errorf("voicevox: url と engines は同時に指定できません（url を voicevox.engines.%s.url へ移行してください）", DefaultEngineName)
	}

	envNameToEngines := make(map[string][]string, len(c.Engines))
	names := make([]string, 0, len(c.Engines))
	for name := range c.Engines {
		names = append(names, name)
	}
	sort.Strings(names)

	for _, name := range names {
		if !engineNamePattern.MatchString(name) {
			return fmt.Errorf("voicevox.engines[%q]: エンジン名は英小文字・数字・-・_ のみ使用できます", name)
		}
		envName := engineURLEnvName(name)
		envNameToEngines[envName] = append(envNameToEngines[envName], name)

		if c.resolveEngineURL(name, c.Engines[name]) == "" {
			return fmt.Errorf("voicevox.engines[%q].url が空です（環境変数 %s で指定するか url を設定してください）", name, envName)
		}
	}

	for _, envName := range sortedKeys(envNameToEngines) {
		conflicting := envNameToEngines[envName]
		if len(conflicting) > 1 {
			return fmt.Errorf("voicevox.engines: エンジン名 %s は環境変数名 %s に正規化され衝突します", strings.Join(conflicting, ", "), envName)
		}
	}
	return nil
}

func sortedKeys(m map[string][]string) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
