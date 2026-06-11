package config

const (
	// DefaultCacheMaxEntries is the default maximum number of episodes to keep in the cache.
	DefaultCacheMaxEntries = 100
	// DefaultCacheRetentionDays is the default number of days to retain cache entries.
	DefaultCacheRetentionDays = 90
	// DefaultCacheLLMContextEntries is the default number of recent episodes to pass to the LLM.
	DefaultCacheLLMContextEntries = 10
)

// CacheConfig controls the episode history cache behavior.
// The cache is always enabled; episodes are keyed by program.id (required).
type CacheConfig struct {
	MaxEntries        int `yaml:"max_entries"`
	RetentionDays     int `yaml:"retention_days"`
	LLMContextEntries int `yaml:"llm_context_entries"`
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
