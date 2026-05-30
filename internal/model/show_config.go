package model

type ShowConfig struct {
	TitleFormat     string         `yaml:"title_format"     json:"title_format"`
	TargetChars     int            `yaml:"target_chars"     json:"target_chars"`
	Corners         int            `yaml:"corners"          json:"corners"`
	DefaultSpeaker  int            `yaml:"default_speaker"  json:"default_speaker"`
	Speakers        map[string]int `yaml:"speakers"         json:"speakers"`
	Persona         string         `yaml:"persona"          json:"persona"`
	SegmentPauseSec float64        `yaml:"segment_pause_sec" json:"segment_pause_sec"`
}
