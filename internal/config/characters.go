package config

import "fmt"

type CharacterConfig struct {
	// Name is the character's display name shown to the LLM during script
	// generation. Empty (or whitespace-only) means an anonymous character:
	// the name is withheld from the prompt and other cast members are
	// instructed not to address this character at all.
	Name         string         `yaml:"name"`
	Pronoun      string         `yaml:"pronoun"`
	SpeechSuffix []string       `yaml:"speech_suffix"`
	Personality  []string       `yaml:"personality"`
	DefaultStyle string         `yaml:"default_style"`
	Styles       map[string]int `yaml:"styles"`
	Credit       string         `yaml:"credit,omitempty"`
	// Engine is the voicevox.engines name used to synthesize this character's
	// speech. Empty means DefaultEngineName ("default").
	Engine string `yaml:"engine,omitempty"`
}

// EffectiveEngine returns the VOICEVOX engine name for this character,
// falling back to DefaultEngineName when Engine is unset.
func (c CharacterConfig) EffectiveEngine() string {
	if c.Engine == "" {
		return DefaultEngineName
	}
	return c.Engine
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

func validateCharacters(chars map[string]CharacterConfig) error {
	for id, ch := range chars {
		if ch.DefaultStyle != "" {
			if _, ok := ch.Styles[ch.DefaultStyle]; !ok {
				return fmt.Errorf("characters[%q].default_style %q not found in styles", id, ch.DefaultStyle)
			}
		}
	}
	return nil
}

// validateCharacterEngines checks that each character's engine refers to a
// defined voicevox engine. In url-only mode (voicevox.engines unset), only
// the implicit DefaultEngineName is a valid reference.
func validateCharacterEngines(chars map[string]CharacterConfig, voicevox VoicevoxConfig) error {
	urlOnlyMode := len(voicevox.Engines) == 0
	for id, ch := range chars {
		engine := ch.EffectiveEngine()
		if urlOnlyMode {
			if engine != DefaultEngineName {
				return fmt.Errorf("characters[%q].engine %q: voicevox.engines が未定義のため %q 以外は指定できません", id, ch.Engine, DefaultEngineName)
			}
			continue
		}
		if _, ok := voicevox.Engines[engine]; !ok {
			return fmt.Errorf("characters[%q].engine %q: voicevox.engines に定義されていません", id, engine)
		}
	}
	return nil
}
