package config

import "fmt"

type CharacterConfig struct {
	Name         string         `yaml:"name"`
	Pronoun      string         `yaml:"pronoun"`
	SpeechSuffix []string       `yaml:"speech_suffix"`
	Personality  []string       `yaml:"personality"`
	DefaultStyle string         `yaml:"default_style"`
	Styles       map[string]int `yaml:"styles"`
	Credit       string         `yaml:"credit,omitempty"`
	// Engine is the voicevox.servers name used to synthesize this character's
	// speech. Empty means DefaultServerName ("default").
	Engine string `yaml:"engine,omitempty"`
}

// EffectiveEngine returns the VOICEVOX server name for this character,
// falling back to DefaultServerName when Engine is unset.
func (c CharacterConfig) EffectiveEngine() string {
	if c.Engine == "" {
		return DefaultServerName
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
// defined voicevox server. In url-only mode (voicevox.servers unset), only
// the implicit DefaultServerName is a valid reference.
func validateCharacterEngines(chars map[string]CharacterConfig, voicevox VoicevoxConfig) error {
	urlOnlyMode := len(voicevox.Servers) == 0
	for id, ch := range chars {
		engine := ch.EffectiveEngine()
		if urlOnlyMode {
			if engine != DefaultServerName {
				return fmt.Errorf("characters[%q].engine %q: voicevox.servers が未定義のため %q 以外は指定できません", id, ch.Engine, DefaultServerName)
			}
			continue
		}
		if _, ok := voicevox.Servers[engine]; !ok {
			return fmt.Errorf("characters[%q].engine %q: voicevox.servers に定義されていません", id, engine)
		}
	}
	return nil
}
