package config

const (
	// DefaultRetroMaxTries is the default maximum number of problems tried at once (ADR-0098):
	// bounding concurrent experiments so a cleared problem's fix can be attributed unambiguously.
	DefaultRetroMaxTries = 3
	// DefaultRetroAnalysisEntries is the default number of recent analyzed episodes retro considers.
	DefaultRetroAnalysisEntries = 5
)

// RetroConfig controls the retro command's automatic improvement loop (ADR-0098).
type RetroConfig struct {
	MaxTries        int `yaml:"max_tries"`
	AnalysisEntries int `yaml:"analysis_entries"`
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
