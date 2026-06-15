package slack

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/canpok1/vox-radio/internal/fileio"
	"github.com/canpok1/vox-radio/internal/render"
)

const (
	defaultParentTemplate = `🎙️ {{.Title}}{{if .EpisodeNumber}} 第{{.EpisodeNumber}}回{{end}}{{if .EpisodeTitle}}「{{.EpisodeTitle}}」{{end}}`

	defaultFallbackTemplate = `{{.Title}}{{if .EpisodeNumber}} 第{{.EpisodeNumber}}回{{end}} を配信しました`

	defaultThreadTemplate = `{{- with .Summary}}*今回のまとめ*
{{.}}
{{- end}}
{{- range .Corners}}

*{{.Title}}*
{{- with .Summary}}
{{.}}
{{- end}}
{{- range .Articles}}
{{- if .URL}}
 • <{{.URL}}|{{.Title}}>
{{- end}}
{{- end}}
{{- end}}`
)

// MessagePaths holds file paths for template files used in Slack messages.
// Each field is a path to a text/template file. Relative paths are resolved
// against the directory of the slack-spec.yaml file. Omit a field to use the
// built-in default template.
type MessagePaths struct {
	Parent   string `yaml:"parent"`   // parent message (mp3 upload initial comment)
	Thread   string `yaml:"thread"`   // thread reply body
	Fallback string `yaml:"fallback"` // notification plain text
}

// SlackChannelConfig holds channel and message settings for a program.
type SlackChannelConfig struct {
	Channel string       `yaml:"channel"`
	Message MessagePaths `yaml:"message"`
}

// SlackSpec is the top-level structure for slack-spec.yaml.
type SlackSpec struct {
	Slack   SlackChannelConfig `yaml:"slack"`
	BaseDir string             `yaml:"-"` // directory of the spec file; set by LoadSlackSpec
}

// LoadedTemplates holds the loaded text/template source texts for the three message types.
type LoadedTemplates struct {
	Parent   string
	Thread   string
	Fallback string
}

// LoadTemplates returns the template text for each message type. When a path is
// empty the built-in default template is used. When a path is set it is read
// from disk; relative paths are resolved against baseDir.
func (c SlackChannelConfig) LoadTemplates(baseDir string) (LoadedTemplates, error) {
	parent, err := loadTemplateSrc(c.Message.Parent, defaultParentTemplate, baseDir)
	if err != nil {
		return LoadedTemplates{}, fmt.Errorf("load parent template: %w", err)
	}
	thread, err := loadTemplateSrc(c.Message.Thread, defaultThreadTemplate, baseDir)
	if err != nil {
		return LoadedTemplates{}, fmt.Errorf("load thread template: %w", err)
	}
	fallback, err := loadTemplateSrc(c.Message.Fallback, defaultFallbackTemplate, baseDir)
	if err != nil {
		return LoadedTemplates{}, fmt.Errorf("load fallback template: %w", err)
	}
	return LoadedTemplates{Parent: parent, Thread: thread, Fallback: fallback}, nil
}

func resolvePath(path, baseDir string) string {
	if !filepath.IsAbs(path) && baseDir != "" {
		return filepath.Join(baseDir, path)
	}
	return path
}

func loadTemplateSrc(path, defaultText, baseDir string) (string, error) {
	if path == "" {
		return defaultText, nil
	}
	data, err := os.ReadFile(resolvePath(path, baseDir))
	if err != nil {
		return "", err
	}
	return string(data), nil
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
	spec.BaseDir = filepath.Dir(path)
	return spec, nil
}

// ValidateSlackSpec validates the semantic correctness of a SlackSpec.
// It checks that the channel is set and that any specified template file paths
// exist and contain valid text/template syntax.
func ValidateSlackSpec(spec SlackSpec) error {
	var errs []error
	if spec.Slack.Channel == "" {
		errs = append(errs, errors.New("slack.channel is required"))
	}
	m := spec.Slack.Message
	for _, entry := range []struct {
		field string
		path  string
	}{
		{"slack.message.parent", m.Parent},
		{"slack.message.thread", m.Thread},
		{"slack.message.fallback", m.Fallback},
	} {
		if entry.path == "" {
			continue
		}
		data, err := os.ReadFile(resolvePath(entry.path, spec.BaseDir))
		if err != nil {
			errs = append(errs, fmt.Errorf("%s: %w", entry.field, err))
			continue
		}
		if err := render.Parse(string(data)); err != nil {
			errs = append(errs, fmt.Errorf("%s: template parse error: %w", entry.field, err))
		}
	}
	return errors.Join(errs...)
}
