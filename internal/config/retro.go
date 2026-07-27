package config

const (
	// DefaultRetroMaxTries is the default maximum number of problems tried at once (ADR-0098):
	// bounding concurrent experiments so a cleared problem's fix can be attributed unambiguously.
	DefaultRetroMaxTries = 3
	// DefaultRetroAnalysisEntries is the default number of recent analyzed episodes retro considers.
	DefaultRetroAnalysisEntries = 5
	// DefaultRetroKeepThreshold is the default number of consecutive non-recurring episodes
	// required to promote a try problem to keep (ADR-0098).
	DefaultRetroKeepThreshold = 3
	// DefaultRetroKeepLength is the default warning threshold (characters) for the keep file's
	// total content; exceeding it does not truncate or rewrite keep, only warns (ADR-0098).
	DefaultRetroKeepLength = 600
)

// RetroConfig controls the retro command's automatic improvement loop (ADR-0098).
type RetroConfig struct {
	MaxTries        int `yaml:"max_tries"`
	AnalysisEntries int `yaml:"analysis_entries"`
	KeepThreshold   int `yaml:"keep_threshold"`
	KeepLength      int `yaml:"keep_length"`
}

// EffectiveMaxTries returns the configured MaxTries, falling back to DefaultRetroMaxTries.
func (c RetroConfig) EffectiveMaxTries() int {
	if c.MaxTries <= 0 {
		return DefaultRetroMaxTries
	}
	return c.MaxTries
}

// EffectiveAnalysisEntries returns the configured AnalysisEntries, falling back to DefaultRetroAnalysisEntries.
func (c RetroConfig) EffectiveAnalysisEntries() int {
	if c.AnalysisEntries <= 0 {
		return DefaultRetroAnalysisEntries
	}
	return c.AnalysisEntries
}

// EffectiveKeepThreshold returns the configured KeepThreshold, falling back to DefaultRetroKeepThreshold.
func (c RetroConfig) EffectiveKeepThreshold() int {
	if c.KeepThreshold <= 0 {
		return DefaultRetroKeepThreshold
	}
	return c.KeepThreshold
}

// EffectiveKeepLength returns the configured KeepLength, falling back to DefaultRetroKeepLength.
func (c RetroConfig) EffectiveKeepLength() int {
	if c.KeepLength <= 0 {
		return DefaultRetroKeepLength
	}
	return c.KeepLength
}
