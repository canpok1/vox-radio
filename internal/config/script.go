package config

const (
	// DefaultScriptRegenThreshold is the default per-corner deviation ratio (|actual-target|/target)
	// above which a corner's script is regenerated once during regenIfNeeded.
	DefaultScriptRegenThreshold = 0.20
	// DefaultScriptRegenMaxRetries is the default number of regeneration attempts made per corner
	// that still exceeds ScriptConfig.RegenThreshold.
	DefaultScriptRegenMaxRetries = 1
)

// ScriptConfig controls the script generation step's regenIfNeeded pass, which regenerates any
// corner whose written character count deviates too far from its target_chars.
type ScriptConfig struct {
	RegenThreshold  float64 `yaml:"regen_threshold"`
	RegenMaxRetries int     `yaml:"regen_max_retries"`
}

// EffectiveRegenThreshold returns the configured RegenThreshold, falling back to
// DefaultScriptRegenThreshold.
func (c ScriptConfig) EffectiveRegenThreshold() float64 {
	if c.RegenThreshold <= 0 {
		return DefaultScriptRegenThreshold
	}
	return c.RegenThreshold
}

// EffectiveRegenMaxRetries returns the configured RegenMaxRetries, falling back to
// DefaultScriptRegenMaxRetries.
func (c ScriptConfig) EffectiveRegenMaxRetries() int {
	if c.RegenMaxRetries <= 0 {
		return DefaultScriptRegenMaxRetries
	}
	return c.RegenMaxRetries
}
