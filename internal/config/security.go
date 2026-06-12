package config

import "fmt"

const (
	// OnDetectExclude excludes the entire article when an injection pattern is detected.
	OnDetectExclude = "exclude"
	// OnDetectError stops the pipeline and returns an error when injection is detected.
	OnDetectError = "error"

	// DefaultMaxArticleBodyChars is the default maximum rune count for article bodies.
	// Bodies longer than this are truncated before being sent to the LLM.
	DefaultMaxArticleBodyChars = 3000
)

// PromptInjectionConfig holds settings for prompt-injection mitigation at the collect boundary.
type PromptInjectionConfig struct {
	// OnDetect controls behavior when an injection pattern is detected in a field.
	// Valid values: "" (defaults to "exclude"), "exclude", "error".
	// "sanitize" is accepted as a deprecated alias for "exclude".
	OnDetect string `yaml:"on_detect,omitempty"`
	// MaxBodyChars is the maximum number of runes allowed in an article body.
	// 0 means use DefaultMaxArticleBodyChars.
	MaxBodyChars int `yaml:"max_body_chars,omitempty"`
}

// EffectiveOnDetect returns the resolved on_detect policy.
// Empty string and the deprecated value "sanitize" both default to OnDetectExclude.
func (c PromptInjectionConfig) EffectiveOnDetect() string {
	switch c.OnDetect {
	case "", "sanitize": // "sanitize" is a deprecated alias for "exclude"
		return OnDetectExclude
	default:
		return c.OnDetect
	}
}

// EffectiveMaxBodyChars returns the resolved maximum body character count.
// 0 defaults to DefaultMaxArticleBodyChars.
func (c PromptInjectionConfig) EffectiveMaxBodyChars() int {
	if c.MaxBodyChars <= 0 {
		return DefaultMaxArticleBodyChars
	}
	return c.MaxBodyChars
}

// SecurityConfig aggregates all security-related settings.
type SecurityConfig struct {
	PromptInjection PromptInjectionConfig `yaml:"prompt_injection,omitempty"`
}

func validateSecurityConfig(cfg *SecurityConfig) error {
	switch cfg.PromptInjection.OnDetect {
	case "", OnDetectExclude, "sanitize", OnDetectError: // "sanitize" is a deprecated alias for "exclude"
		return nil
	default:
		return fmt.Errorf("security.prompt_injection.on_detect: invalid value %q (must be %q or %q)", cfg.PromptInjection.OnDetect, OnDetectExclude, OnDetectError)
	}
}
