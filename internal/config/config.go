package config

// Config holds genre-independent common settings.
// It is loaded from vox-radio.yaml at the repository root.
type Config struct {
	LLM        LLMConfig                  `yaml:"llm"`
	Voicevox   VoicevoxConfig             `yaml:"voicevox"`
	Characters map[string]CharacterConfig `yaml:"characters"`
	Cache      CacheConfig                `yaml:"cache"`
	Slack      SlackConfig                `yaml:"slack"`
}
