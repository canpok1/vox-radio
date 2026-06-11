package slack

import (
	"errors"
	"fmt"

	"github.com/canpok1/vox-radio/internal/fileio"
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
	or := func(s, def string) string {
		if s != "" {
			return s
		}
		return def
	}
	return MessageTemplate{
		Header:   or(c.Message.Header, defaultHeaderTemplate),
		Fallback: or(c.Message.Fallback, defaultFallbackTemplate),
		Summary:  or(c.Message.Summary, defaultSummaryTemplate),
		Corner:   or(c.Message.Corner, defaultCornerTemplate),
		Article:  or(c.Message.Article, defaultArticleTemplate),
	}
}

// SlackSpec is the top-level structure for slack-spec.yaml.
type SlackSpec struct {
	Slack SlackChannelConfig `yaml:"slack"`
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
	var spec SlackSpec
	if err := fileio.DecodeYAML(path, &spec, strict); err != nil {
		return SlackSpec{}, fmt.Errorf("load slack spec: %w", err)
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
