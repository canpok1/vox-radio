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
	// VoicevoxURLEnv は VOICEVOX URL を上書きする環境変数名（default サーバー専用）。
	VoicevoxURLEnv = "VOX_RADIO_VOICEVOX_URL"
	// DefaultServerName は voicevox.servers 省略時・characters[].engine 省略時に使うサーバー名。
	DefaultServerName = "default"
	// DefaultStartupTimeoutSeconds は startup_timeout_seconds 未指定時のデフォルト秒数。
	DefaultStartupTimeoutSeconds = 60
)

// serverNamePattern restricts voicevox.servers keys to a form that normalizes
// unambiguously to an environment variable name (see serverURLEnvName).
var serverNamePattern = regexp.MustCompile(`^[a-z0-9_-]+$`)

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

// VoicevoxServerConfig is a single named VOICEVOX-compatible server entry
// under voicevox.servers.
type VoicevoxServerConfig struct {
	URL string `yaml:"url"`
}

type VoicevoxConfig struct {
	URL                   string                          `yaml:"url"`
	Servers               map[string]VoicevoxServerConfig `yaml:"servers,omitempty"`
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

// serverURLEnvName returns the per-server URL override env var name for a
// server name: VOX_RADIO_VOICEVOX_URL_<NAME> with the name upper-cased and
// "-" normalized to "_".
func serverURLEnvName(name string) string {
	norm := strings.ToUpper(strings.ReplaceAll(name, "-", "_"))
	return VoicevoxURLEnv + "_" + norm
}

// resolveServerURL resolves a single named server's URL following the
// priority: ① VOX_RADIO_VOICEVOX_URL_<NAME> ② (default のみ) VOX_RADIO_VOICEVOX_URL
// ③ 設定値 ④ (default のみ) デフォルト定数。Returns "" if none apply (only
// possible for a non-default server with no configured url and no override).
func (c VoicevoxConfig) resolveServerURL(name string, sc VoicevoxServerConfig) string {
	if v := os.Getenv(serverURLEnvName(name)); v != "" {
		return v
	}
	if name == DefaultServerName {
		if v := os.Getenv(VoicevoxURLEnv); v != "" {
			return v
		}
	}
	if sc.URL != "" {
		return sc.URL
	}
	if name == DefaultServerName {
		return DefaultVoicevoxURL
	}
	return ""
}

// EffectiveURLs resolves every configured VOICEVOX-compatible server to its
// final URL, keyed by server name. When voicevox.servers is not set, the
// single voicevox.url (or its env/default fallback) is returned as the
// implicit "default" server, preserving pre-multi-server behavior.
func (c VoicevoxConfig) EffectiveURLs() map[string]string {
	if len(c.Servers) == 0 {
		return map[string]string{DefaultServerName: c.resolveServerURL(DefaultServerName, VoicevoxServerConfig{URL: c.URL})}
	}
	result := make(map[string]string, len(c.Servers))
	for name, sc := range c.Servers {
		result[name] = c.resolveServerURL(name, sc)
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

// validateVoicevoxServers checks voicevox.url / voicevox.servers for mutual
// exclusion, server name format, required urls (env override may substitute),
// and normalized-env-name collisions between server names.
func validateVoicevoxServers(c VoicevoxConfig) error {
	if len(c.Servers) == 0 {
		return nil
	}
	if c.URL != "" {
		return fmt.Errorf("voicevox: url と servers は同時に指定できません（url を voicevox.servers.%s.url へ移行してください）", DefaultServerName)
	}

	envNameToServers := make(map[string][]string, len(c.Servers))
	names := make([]string, 0, len(c.Servers))
	for name := range c.Servers {
		names = append(names, name)
	}
	sort.Strings(names)

	for _, name := range names {
		if !serverNamePattern.MatchString(name) {
			return fmt.Errorf("voicevox.servers[%q]: サーバー名は英小文字・数字・-・_ のみ使用できます", name)
		}
		envName := serverURLEnvName(name)
		envNameToServers[envName] = append(envNameToServers[envName], name)

		if c.resolveServerURL(name, c.Servers[name]) == "" {
			return fmt.Errorf("voicevox.servers[%q].url が空です（環境変数 %s で指定するか url を設定してください）", name, envName)
		}
	}

	for _, envName := range sortedKeys(envNameToServers) {
		conflicting := envNameToServers[envName]
		if len(conflicting) > 1 {
			return fmt.Errorf("voicevox.servers: サーバー名 %s は環境変数名 %s に正規化され衝突します", strings.Join(conflicting, ", "), envName)
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
