package config

import "fmt"

const (
	// OnDetectSanitize drops the flagged field and continues processing.
	OnDetectSanitize = "sanitize"
	// OnDetectError stops the pipeline and returns an error when injection is detected.
	OnDetectError = "error"

	// DefaultMaxArticleBodyChars is the default maximum rune count for article bodies.
	// Bodies longer than this are truncated before being sent to the LLM.
	DefaultMaxArticleBodyChars = 3000
)

// PromptInjectionConfig holds settings for prompt-injection mitigation at the collect boundary.
type PromptInjectionConfig struct {
	// OnDetect controls behavior when an injection pattern is detected in a field.
	// Valid values: "" (defaults to "sanitize"), "sanitize", "error".
	OnDetect string `yaml:"on_detect,omitempty"`
	// MaxBodyChars is the maximum number of runes allowed in an article body.
	// 0 means use DefaultMaxArticleBodyChars.
	MaxBodyChars int `yaml:"max_body_chars,omitempty"`
}

// EffectiveOnDetect returns the resolved on_detect policy.
// Empty string defaults to OnDetectSanitize.
func (c PromptInjectionConfig) EffectiveOnDetect() string {
	if c.OnDetect == "" {
		return OnDetectSanitize
	}
	return c.OnDetect
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
	case "", OnDetectSanitize, OnDetectError:
		return nil
	default:
		return fmt.Errorf("security.prompt_injection.on_detect: invalid value %q (must be %q or %q)", cfg.PromptInjection.OnDetect, OnDetectSanitize, OnDetectError)
	}
}
