package config

// EffectivePronunciation returns the proper-noun reading dictionary, never nil.
// An empty dictionary means no replacement is performed.
//
// Collision detection (the same written form mapped to two readings) is handled
// by the YAML decoder itself, which rejects duplicate mapping keys at load time.
func (c Config) EffectivePronunciation() map[string]string {
	if c.Pronunciation == nil {
		return map[string]string{}
	}
	return c.Pronunciation
}
