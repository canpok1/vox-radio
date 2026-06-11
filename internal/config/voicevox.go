package config

import (
	"fmt"
	"os"
)

const (
	// DefaultVoicevoxURL は voicevox.url 未指定時のデフォルト URL。
	DefaultVoicevoxURL = "http://localhost:50021"
	// VoicevoxURLEnv は VOICEVOX URL を上書きする環境変数名。
	VoicevoxURLEnv = "VOX_RADIO_VOICEVOX_URL"
)

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

// EffectiveURL は環境変数 > 設定値 > デフォルトの優先順で URL を返す。
func (c VoicevoxConfig) EffectiveURL() string {
	if v := os.Getenv(VoicevoxURLEnv); v != "" {
		return v
	}
	if c.URL != "" {
		return c.URL
	}
	return DefaultVoicevoxURL
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
