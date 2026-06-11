package config

import "fmt"

const (
	// DefaultProvider is the default LLM provider when none is specified.
	DefaultProvider = "openai"
	// ProviderDifyChat selects the Dify chat-messages endpoint.
	ProviderDifyChat = "dify-chat"
	// DefaultDifyUser is the default user identifier sent to Dify when not configured.
	DefaultDifyUser = "vox-radio"
	// DifyTemperaturePlaceholder is the placeholder in inputs values replaced with the effective temperature.
	DifyTemperaturePlaceholder = "${temperature}"
	// DefaultMinRequestIntervalMS is the default minimum interval (ms) between LLM API requests.
	// Based on gemini-3.1-flash-lite free tier (15 RPM) with ~10% safety margin: 60000/15 * 1.1 ≈ 4500.
	DefaultMinRequestIntervalMS = 4500
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
