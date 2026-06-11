package slack_test

import (
	"testing"

	"github.com/canpok1/vox-radio/internal/slack"
	"github.com/canpok1/vox-radio/internal/testutil"
)

const validSlackSpecYAML = `
slack:
  channel: "C0123456789"
  message:
    header: "🎙️ {title} 第{episode_number}回「{episode_title}」"
    fallback: "{title} 第{episode_number}回 を配信しました"
    summary: "*今回のまとめ*\n{summary}"
    corner: "*{corner_title}*\n{corner_summary}\n{articles}"
    article: " • <{url}|{title}>"
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
	if spec.Slack.Message.Header == "" {
		t.Error("Slack.Message.Header must not be empty")
	}
	if spec.Slack.Message.Fallback == "" {
		t.Error("Slack.Message.Fallback must not be empty")
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
	if spec.Slack.Message.Header != "" {
		t.Errorf("Message.Header should be empty when omitted, got %q", spec.Slack.Message.Header)
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

func TestSlackSpec_EffectiveMessageTemplate_UsesCustomWhenSet(t *testing.T) {
	spec := slack.SlackSpec{
		Slack: slack.SlackChannelConfig{
			Channel: "C0123456789",
			Message: slack.MessageTemplate{
				Header:   "custom header",
				Fallback: "custom fallback",
				Summary:  "custom summary",
				Corner:   "custom corner",
				Article:  "custom article",
			},
		},
	}
	tmpl := spec.Slack.EffectiveMessageTemplate()
	if tmpl.Header != "custom header" {
		t.Errorf("Header: got %q, want %q", tmpl.Header, "custom header")
	}
	if tmpl.Fallback != "custom fallback" {
		t.Errorf("Fallback: got %q, want %q", tmpl.Fallback, "custom fallback")
	}
	if tmpl.Summary != "custom summary" {
		t.Errorf("Summary: got %q, want %q", tmpl.Summary, "custom summary")
	}
	if tmpl.Corner != "custom corner" {
		t.Errorf("Corner: got %q, want %q", tmpl.Corner, "custom corner")
	}
	if tmpl.Article != "custom article" {
		t.Errorf("Article: got %q, want %q", tmpl.Article, "custom article")
	}
}

func TestSlackSpec_EffectiveMessageTemplate_FallsBackToDefault(t *testing.T) {
	spec := slack.SlackSpec{
		Slack: slack.SlackChannelConfig{Channel: "C0123456789"},
	}
	tmpl := spec.Slack.EffectiveMessageTemplate()
	if tmpl.Header == "" {
		t.Error("Header should fall back to default when not set")
	}
	if tmpl.Fallback == "" {
		t.Error("Fallback should fall back to default when not set")
	}
	if tmpl.Summary == "" {
		t.Error("Summary should fall back to default when not set")
	}
	if tmpl.Corner == "" {
		t.Error("Corner should fall back to default when not set")
	}
	if tmpl.Article == "" {
		t.Error("Article should fall back to default when not set")
	}
}
