package config

type CharacterConfig struct {
	Name         string         `yaml:"name"`
	Pronoun      string         `yaml:"pronoun"`
	SpeechSuffix []string       `yaml:"speech_suffix"`
	Personality  []string       `yaml:"personality"`
	DefaultStyle string         `yaml:"default_style"`
	Styles       map[string]int `yaml:"styles"`
}

// DefaultSpeakerID returns the VOICEVOX speaker ID for the character's default style.
func (c CharacterConfig) DefaultSpeakerID() (int, bool) {
	if c.DefaultStyle == "" {
		return 0, false
	}
	id, ok := c.Styles[c.DefaultStyle]
	return id, ok
}

// SpeakerID returns the VOICEVOX speaker ID for the given style name.
// Falls back to the default style if style is empty or not found in Styles.
func (c CharacterConfig) SpeakerID(style string) (int, bool) {
	if style != "" {
		if id, ok := c.Styles[style]; ok {
			return id, true
		}
	}
	return c.DefaultSpeakerID()
}
