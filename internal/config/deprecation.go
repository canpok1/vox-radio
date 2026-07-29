package config

// DeprecationWarning describes a single deprecated config field currently in use.
// Field is a stable dotted identifier (e.g. "corners[].length_sec") for callers that want to
// key or dedupe by field; Message is the full human-readable warning text.
type DeprecationWarning struct {
	Field   string
	Message string
}

// Deprecations returns one warning per deprecated field currently in use in p. Add checks
// here as fields are deprecated, rather than growing ad-hoc log calls at each call site, so
// callers that log once per command startup (episodegen / check / analyze / retro) don't need
// to change when a new field is deprecated.
func (p *EpisodeSpec) Deprecations() []DeprecationWarning {
	var warnings []DeprecationWarning
	if p.usesLengthSec() {
		warnings = append(warnings, DeprecationWarning{
			Field:   "corners[].length_sec",
			Message: "corners[].length_sec は非推奨です。corners[].target_chars を使用してください（詳細: references/deprecated.md）",
		})
	}
	if p.Program.CharsPerMinute != 0 {
		warnings = append(warnings, DeprecationWarning{
			Field:   "program.chars_per_minute",
			Message: "program.chars_per_minute は非推奨です。corners[].target_chars を使用してください（詳細: references/deprecated.md）",
		})
	}
	return warnings
}

func (p *EpisodeSpec) usesLengthSec() bool {
	for _, c := range p.Corners {
		if c.LengthSec != 0 {
			return true
		}
	}
	return false
}
