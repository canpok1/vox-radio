package slack_test

import (
	"os"
	"path/filepath"
	"testing"
	"text/template"

	"github.com/canpok1/vox-radio/internal/slack"
	"github.com/canpok1/vox-radio/internal/testutil"
)

const validSlackSpecYAML = `
slack:
  channel: "C0123456789"
`

const validSlackSpecWithPathsYAML = `
slack:
  channel: "C0123456789"
  message:
    parent: "slack-parent.tmpl"
    thread: "slack-thread.tmpl"
    fallback: "slack-fallback.tmpl"
`

func TestLoadSlackSpec_ValidYAML(t *testing.T) {
	path := testutil.WriteTempFile(t, "slack-spec.yaml", []byte(validSlackSpecYAML))

	spec, err := slack.LoadSlackSpec(path)
	if err != nil {
		t.Fatalf("LoadSlackSpec: unexpected error: %v", err)
	}

	if spec.Slack.Channel != "C0123456789" {
		t.Errorf("Slack.Channel: got %q, want %q", spec.Slack.Channel, "C0123456789")
	}
}

func TestLoadSlackSpec_SetsBaseDir(t *testing.T) {
	path := testutil.WriteTempFile(t, "slack-spec.yaml", []byte(validSlackSpecYAML))

	spec, err := slack.LoadSlackSpec(path)
	if err != nil {
		t.Fatalf("LoadSlackSpec: unexpected error: %v", err)
	}

	if spec.BaseDir == "" {
		t.Error("BaseDir must be set after loading")
	}
	if spec.BaseDir != filepath.Dir(path) {
		t.Errorf("BaseDir: got %q, want %q", spec.BaseDir, filepath.Dir(path))
	}
}

func TestLoadSlackSpec_WithMessagePaths(t *testing.T) {
	path := testutil.WriteTempFile(t, "slack-spec.yaml", []byte(validSlackSpecWithPathsYAML))

	spec, err := slack.LoadSlackSpec(path)
	if err != nil {
		t.Fatalf("LoadSlackSpec: unexpected error: %v", err)
	}

	if spec.Slack.Message.Parent != "slack-parent.tmpl" {
		t.Errorf("Message.Parent: got %q, want %q", spec.Slack.Message.Parent, "slack-parent.tmpl")
	}
	if spec.Slack.Message.Thread != "slack-thread.tmpl" {
		t.Errorf("Message.Thread: got %q, want %q", spec.Slack.Message.Thread, "slack-thread.tmpl")
	}
	if spec.Slack.Message.Fallback != "slack-fallback.tmpl" {
		t.Errorf("Message.Fallback: got %q, want %q", spec.Slack.Message.Fallback, "slack-fallback.tmpl")
	}
}

func TestLoadSlackSpec_MissingFile(t *testing.T) {
	_, err := slack.LoadSlackSpec("/nonexistent/slack-spec.yaml")
	if err == nil {
		t.Error("expected error for missing file")
	}
}

func TestLoadSlackSpecStrict_UnknownKeyErrors(t *testing.T) {
	content := validSlackSpecYAML + "\nunknown_key: value\n"
	path := testutil.WriteTempFile(t, "slack-spec.yaml", []byte(content))

	_, err := slack.LoadSlackSpecStrict(path)
	if err == nil {
		t.Error("expected error for unknown key in strict mode")
	}
}

func TestLoadSlackSpecStrict_ValidYAML_Success(t *testing.T) {
	path := testutil.WriteTempFile(t, "slack-spec.yaml", []byte(validSlackSpecYAML))

	_, err := slack.LoadSlackSpecStrict(path)
	if err != nil {
		t.Errorf("unexpected error for valid spec in strict mode: %v", err)
	}
}

// program_id は SlackSpec から削除されたため、strict モードで unknown key エラーになること
func TestLoadSlackSpecStrict_ProgramIDField_RaisesUnknownKey(t *testing.T) {
	content := "program_id: my-radio\n" + validSlackSpecYAML
	path := testutil.WriteTempFile(t, "slack-spec.yaml", []byte(content))

	_, err := slack.LoadSlackSpecStrict(path)
	if err == nil {
		t.Error("expected error for program_id (unknown key) in strict mode, got nil")
	}
}

func TestLoadSlackSpec_MessageOmitted_DefaultsToEmpty(t *testing.T) {
	content := `
slack:
  channel: "C0123456789"
`
	path := testutil.WriteTempFile(t, "slack-spec.yaml", []byte(content))

	spec, err := slack.LoadSlackSpec(path)
	if err != nil {
		t.Fatalf("LoadSlackSpec: unexpected error: %v", err)
	}
	if spec.Slack.Message.Parent != "" {
		t.Errorf("Message.Parent should be empty when omitted, got %q", spec.Slack.Message.Parent)
	}
	if spec.Slack.Message.Thread != "" {
		t.Errorf("Message.Thread should be empty when omitted, got %q", spec.Slack.Message.Thread)
	}
}

func TestValidateSlackSpec_ValidChannel(t *testing.T) {
	spec := slack.SlackSpec{
		Slack: slack.SlackChannelConfig{Channel: "C0123456789"},
	}
	if err := slack.ValidateSlackSpec(spec); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestValidateSlackSpec_EmptyChannel_Error(t *testing.T) {
	spec := slack.SlackSpec{
		Slack: slack.SlackChannelConfig{Channel: ""},
	}
	if err := slack.ValidateSlackSpec(spec); err == nil {
		t.Error("expected error for empty channel")
	}
}

func TestValidateSlackSpec_MissingTemplateFile_Error(t *testing.T) {
	spec := slack.SlackSpec{
		Slack: slack.SlackChannelConfig{
			Channel: "C0123456789",
			Message: slack.MessagePaths{
				Parent: "nonexistent-parent.tmpl",
			},
		},
		BaseDir: t.TempDir(),
	}
	if err := slack.ValidateSlackSpec(spec); err == nil {
		t.Error("expected error for missing template file")
	}
}

func TestValidateSlackSpec_InvalidTemplateSyntax_Error(t *testing.T) {
	dir := t.TempDir()
	badTmpl := "{{invalid template syntax"
	badPath := filepath.Join(dir, "bad.tmpl")
	if err := os.WriteFile(badPath, []byte(badTmpl), 0o644); err != nil {
		t.Fatalf("write bad template: %v", err)
	}

	spec := slack.SlackSpec{
		Slack: slack.SlackChannelConfig{
			Channel: "C0123456789",
			Message: slack.MessagePaths{
				Parent: "bad.tmpl",
			},
		},
		BaseDir: dir,
	}
	if err := slack.ValidateSlackSpec(spec); err == nil {
		t.Error("expected error for invalid template syntax")
	}
}

func TestValidateSlackSpec_ValidTemplatePaths_Success(t *testing.T) {
	dir := t.TempDir()
	tmplContent := `{{.Title}} {{if .EpisodeNumber}}第{{.EpisodeNumber}}回{{end}}`
	if err := os.WriteFile(filepath.Join(dir, "parent.tmpl"), []byte(tmplContent), 0o644); err != nil {
		t.Fatalf("write template: %v", err)
	}

	spec := slack.SlackSpec{
		Slack: slack.SlackChannelConfig{
			Channel: "C0123456789",
			Message: slack.MessagePaths{
				Parent: "parent.tmpl",
			},
		},
		BaseDir: dir,
	}
	if err := slack.ValidateSlackSpec(spec); err != nil {
		t.Errorf("unexpected error for valid template path: %v", err)
	}
}

func TestValidateSlackSpec_TemplateWithCornerFunction_Success(t *testing.T) {
	dir := t.TempDir()
	// Templates using corner/hasLinks must pass validation (render.Parse includes FuncMap).
	tmplContent := `{{with corner "news"}}{{.Title}}{{end}}`
	if err := os.WriteFile(filepath.Join(dir, "thread.tmpl"), []byte(tmplContent), 0o644); err != nil {
		t.Fatalf("write template: %v", err)
	}

	spec := slack.SlackSpec{
		Slack: slack.SlackChannelConfig{
			Channel: "C0123456789",
			Message: slack.MessagePaths{
				Thread: "thread.tmpl",
			},
		},
		BaseDir: dir,
	}
	if err := slack.ValidateSlackSpec(spec); err != nil {
		t.Errorf("ValidateSlackSpec must accept template with corner function: %v", err)
	}
}

func TestLoadTemplates_EmptyPaths_UsesDefaults(t *testing.T) {
	config := slack.SlackChannelConfig{
		Channel: "C0123456789",
	}
	templates, err := config.LoadTemplates("")
	if err != nil {
		t.Fatalf("LoadTemplates: unexpected error: %v", err)
	}
	if templates.Parent == "" {
		t.Error("Parent template must not be empty when using defaults")
	}
	if templates.Thread == "" {
		t.Error("Thread template must not be empty when using defaults")
	}
	if templates.Fallback == "" {
		t.Error("Fallback template must not be empty when using defaults")
	}
	// Defaults must be valid text/template syntax
	if _, err := template.New("").Parse(templates.Parent); err != nil {
		t.Errorf("default parent template parse error: %v", err)
	}
	if _, err := template.New("").Parse(templates.Thread); err != nil {
		t.Errorf("default thread template parse error: %v", err)
	}
	if _, err := template.New("").Parse(templates.Fallback); err != nil {
		t.Errorf("default fallback template parse error: %v", err)
	}
}

func TestLoadTemplates_WithFilePaths_ReadsFiles(t *testing.T) {
	dir := t.TempDir()
	parentContent := `親メッセージ: {{.Title}}`
	threadContent := `スレッド: {{.Summary}}`
	fallbackContent := `フォールバック: {{.Title}}`

	for name, content := range map[string]string{
		"parent.tmpl":   parentContent,
		"thread.tmpl":   threadContent,
		"fallback.tmpl": fallbackContent,
	} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
			t.Fatalf("write %s: %v", name, err)
		}
	}

	config := slack.SlackChannelConfig{
		Channel: "C0123456789",
		Message: slack.MessagePaths{
			Parent:   "parent.tmpl",
			Thread:   "thread.tmpl",
			Fallback: "fallback.tmpl",
		},
	}
	templates, err := config.LoadTemplates(dir)
	if err != nil {
		t.Fatalf("LoadTemplates: unexpected error: %v", err)
	}

	if templates.Parent != parentContent {
		t.Errorf("Parent: got %q, want %q", templates.Parent, parentContent)
	}
	if templates.Thread != threadContent {
		t.Errorf("Thread: got %q, want %q", templates.Thread, threadContent)
	}
	if templates.Fallback != fallbackContent {
		t.Errorf("Fallback: got %q, want %q", templates.Fallback, fallbackContent)
	}
}

func TestLoadTemplates_AbsolutePath(t *testing.T) {
	dir := t.TempDir()
	content := `絶対パステスト: {{.Title}}`
	absPath := filepath.Join(dir, "abs.tmpl")
	if err := os.WriteFile(absPath, []byte(content), 0o644); err != nil {
		t.Fatalf("write template: %v", err)
	}

	config := slack.SlackChannelConfig{
		Channel: "C0123456789",
		Message: slack.MessagePaths{
			Parent: absPath, // absolute path
		},
	}
	templates, err := config.LoadTemplates("/some/other/dir")
	if err != nil {
		t.Fatalf("LoadTemplates: unexpected error: %v", err)
	}
	if templates.Parent != content {
		t.Errorf("Parent: got %q, want %q", templates.Parent, content)
	}
}

func TestLoadTemplates_MissingFile_Error(t *testing.T) {
	config := slack.SlackChannelConfig{
		Channel: "C0123456789",
		Message: slack.MessagePaths{
			Parent: "nonexistent.tmpl",
		},
	}
	_, err := config.LoadTemplates(t.TempDir())
	if err == nil {
		t.Error("expected error for missing template file")
	}
}
