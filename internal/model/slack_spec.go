package model

import (
	"errors"
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

const (
	defaultHeaderTemplate   = "🎙️ {title} 第{episode_number}回「{episode_title}」"
	defaultFallbackTemplate = "{title} 第{episode_number}回 を配信しました"
	defaultSummaryTemplate  = "*今回のまとめ*\n{summary}"
	defaultCornerTemplate   = "*{corner_title}*\n{corner_summary}\n{articles}"
	defaultArticleTemplate  = " • <{url}|{title}>"
)

// MessageTemplate holds template strings for composing Slack messages.
type MessageTemplate struct {
	Header   string `yaml:"header"`
	Fallback string `yaml:"fallback"`
	Summary  string `yaml:"summary"`
	Corner   string `yaml:"corner"`
	Article  string `yaml:"article"`
}

// SlackChannelConfig holds channel and message settings for a program.
type SlackChannelConfig struct {
	Channel string          `yaml:"channel"`
	Message MessageTemplate `yaml:"message"`
}

// EffectiveMessageTemplate returns the configured template, falling back to defaults per field.
func (c SlackChannelConfig) EffectiveMessageTemplate() MessageTemplate {
	tmpl := c.Message
	if tmpl.Header == "" {
		tmpl.Header = defaultHeaderTemplate
	}
	if tmpl.Fallback == "" {
		tmpl.Fallback = defaultFallbackTemplate
	}
	if tmpl.Summary == "" {
		tmpl.Summary = defaultSummaryTemplate
	}
	if tmpl.Corner == "" {
		tmpl.Corner = defaultCornerTemplate
	}
	if tmpl.Article == "" {
		tmpl.Article = defaultArticleTemplate
	}
	return tmpl
}

// SlackSpec is the top-level structure for slack-spec.yaml.
type SlackSpec struct {
	ProgramID string             `yaml:"program_id"`
	Slack     SlackChannelConfig `yaml:"slack"`
}

// LoadSlackSpec reads and parses a slack-spec.yaml file.
func LoadSlackSpec(path string) (SlackSpec, error) {
	return loadSlackSpecWith(path, false)
}

// LoadSlackSpecStrict reads and parses a slack-spec.yaml file with strict mode.
// Unknown keys in the YAML will cause an error (detects typos).
func LoadSlackSpecStrict(path string) (SlackSpec, error) {
	return loadSlackSpecWith(path, true)
}

func loadSlackSpecWith(path string, strict bool) (SlackSpec, error) {
	f, err := os.Open(path)
	if err != nil {
		return SlackSpec{}, fmt.Errorf("read slack spec: %w", err)
	}
	defer func() { _ = f.Close() }()
	dec := yaml.NewDecoder(f)
	if strict {
		dec.KnownFields(true)
	}
	var spec SlackSpec
	if err := dec.Decode(&spec); err != nil {
		return SlackSpec{}, fmt.Errorf("parse slack spec: %w", err)
	}
	return spec, nil
}

// ValidateSlackSpec validates the semantic correctness of a SlackSpec.
func ValidateSlackSpec(spec SlackSpec) error {
	var errs []error
	if spec.Slack.Channel == "" {
		errs = append(errs, errors.New("slack.channel is required"))
	}
	return errors.Join(errs...)
}
