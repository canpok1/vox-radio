package config

// Config holds genre-independent common settings.
// It is loaded from vox-radio.yaml at the repository root.
type Config struct {
	LLM        LLMConfig                  `yaml:"llm"`
	Voicevox   VoicevoxConfig             `yaml:"voicevox"`
	Characters map[string]CharacterConfig `yaml:"characters"`
	Cache      CacheConfig                `yaml:"cache"`
	Slack      SlackConfig                `yaml:"slack"`
	Security   SecurityConfig             `yaml:"security,omitempty"`
	// Pronunciation is a global proper-noun reading dictionary (written form -> reading).
	// It is applied to each speech line before LLM kana conversions during scripting.
	Pronunciation map[string]string `yaml:"pronunciation,omitempty"`
}
