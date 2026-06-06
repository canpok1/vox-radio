package cli_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/canpok1/vox-radio/internal/cli"
	"github.com/canpok1/vox-radio/internal/config"
	"github.com/canpok1/vox-radio/internal/model"
)

func chdirTemp(t *testing.T) string {
	t.Helper()
	orig, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	dir := t.TempDir()
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	t.Cleanup(func() { _ = os.Chdir(orig) })
	return dir
}

func writeTestFile(t *testing.T, dir, name string, content []byte) {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, content, 0644); err != nil {
		t.Fatal(err)
	}
}

func runInitCmd(t *testing.T) (string, error) {
	t.Helper()
	cmd := cli.NewRootCmd()
	var buf strings.Builder
	cmd.SetOut(&buf)
	cmd.SetArgs([]string{"init"})
	err := cmd.Execute()
	return buf.String(), err
}

func TestInitCmd_AllGenerated(t *testing.T) {
	dir := chdirTemp(t)
	_, err := runInitCmd(t)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for _, name := range []string{"vox-radio.yaml", "episode-spec.yaml", "feed-spec.yaml", "slack-spec.yaml", "assets/assets.yaml"} {
		if _, err := os.Stat(filepath.Join(dir, name)); os.IsNotExist(err) {
			t.Errorf("%s was not generated", name)
		}
	}
}

func TestInitCmd_ConfigExists_EpisodeSpecGenerated(t *testing.T) {
	dir := chdirTemp(t)
	existingContent := []byte("# existing")
	if err := os.WriteFile(filepath.Join(dir, "vox-radio.yaml"), existingContent, 0644); err != nil {
		t.Fatal(err)
	}
	out, err := runInitCmd(t)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	data, _ := os.ReadFile(filepath.Join(dir, "vox-radio.yaml"))
	if string(data) != string(existingContent) {
		t.Error("vox-radio.yaml should not be overwritten")
	}
	if _, err := os.Stat(filepath.Join(dir, "episode-spec.yaml")); os.IsNotExist(err) {
		t.Error("episode-spec.yaml was not generated")
	}
	if _, err := os.Stat(filepath.Join(dir, "feed-spec.yaml")); os.IsNotExist(err) {
		t.Error("feed-spec.yaml was not generated")
	}
	if !strings.Contains(out, "skip") {
		t.Errorf("expected skip message for vox-radio.yaml, got: %s", out)
	}
}

func TestInitCmd_EpisodeSpecExists_ConfigGenerated(t *testing.T) {
	dir := chdirTemp(t)
	existingContent := []byte("# existing")
	if err := os.WriteFile(filepath.Join(dir, "episode-spec.yaml"), existingContent, 0644); err != nil {
		t.Fatal(err)
	}
	out, err := runInitCmd(t)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	data, _ := os.ReadFile(filepath.Join(dir, "episode-spec.yaml"))
	if string(data) != string(existingContent) {
		t.Error("episode-spec.yaml should not be overwritten")
	}
	if _, err := os.Stat(filepath.Join(dir, "vox-radio.yaml")); os.IsNotExist(err) {
		t.Error("vox-radio.yaml was not generated")
	}
	if _, err := os.Stat(filepath.Join(dir, "feed-spec.yaml")); os.IsNotExist(err) {
		t.Error("feed-spec.yaml was not generated")
	}
	if !strings.Contains(out, "skip") {
		t.Errorf("expected skip message for episode-spec.yaml, got: %s", out)
	}
}

func TestInitCmd_AllExist_NothingGenerated(t *testing.T) {
	dir := chdirTemp(t)
	existingContent := []byte("# existing")
	allFiles := []string{"vox-radio.yaml", "episode-spec.yaml", "feed-spec.yaml", "slack-spec.yaml", "assets/assets.yaml"}
	for _, name := range allFiles {
		writeTestFile(t, dir, name, existingContent)
	}
	_, err := runInitCmd(t)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for _, name := range allFiles {
		data, _ := os.ReadFile(filepath.Join(dir, name))
		if string(data) != string(existingContent) {
			t.Errorf("%s should not be overwritten", name)
		}
	}
}

func TestInitCmd_FeedSpecExists_Skipped(t *testing.T) {
	dir := chdirTemp(t)
	existingContent := []byte("# existing")
	if err := os.WriteFile(filepath.Join(dir, "feed-spec.yaml"), existingContent, 0644); err != nil {
		t.Fatal(err)
	}
	out, err := runInitCmd(t)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	data, _ := os.ReadFile(filepath.Join(dir, "feed-spec.yaml"))
	if string(data) != string(existingContent) {
		t.Error("feed-spec.yaml should not be overwritten")
	}
	if !strings.Contains(out, "skip") {
		t.Errorf("expected skip message for feed-spec.yaml, got: %s", out)
	}
}

func TestInitCmd_AssetsYamlExists_Skipped(t *testing.T) {
	dir := chdirTemp(t)
	existingContent := []byte("# existing")
	writeTestFile(t, dir, "assets/assets.yaml", existingContent)
	out, err := runInitCmd(t)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	data, _ := os.ReadFile(filepath.Join(dir, "assets", "assets.yaml"))
	if string(data) != string(existingContent) {
		t.Error("assets/assets.yaml should not be overwritten")
	}
	if !strings.Contains(out, "skip") {
		t.Errorf("expected skip message for assets/assets.yaml, got: %s", out)
	}
}

func TestInitCmd_GeneratedFilesLoadable(t *testing.T) {
	dir := chdirTemp(t)
	_, err := runInitCmd(t)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	cfg, err := config.LoadConfig(filepath.Join(dir, "vox-radio.yaml"))
	if err != nil {
		t.Fatalf("LoadConfig failed on generated template: %v", err)
	}
	spec, err := config.LoadEpisodeSpec(filepath.Join(dir, "episode-spec.yaml"))
	if err != nil {
		t.Fatalf("LoadEpisodeSpec failed on generated template: %v", err)
	}
	if _, err := model.LoadFeedSpec(filepath.Join(dir, "feed-spec.yaml")); err != nil {
		t.Fatalf("LoadFeedSpec failed on generated template: %v", err)
	}
	slackSpec, err := model.LoadSlackSpec(filepath.Join(dir, "slack-spec.yaml"))
	if err != nil {
		t.Fatalf("LoadSlackSpec failed on generated template: %v", err)
	}
	if slackSpec.Slack.Channel == "" {
		t.Error("slack-spec.yaml template should have a channel value")
	}
	if cfg.Slack.BotTokenEnv == "" {
		t.Error("vox-radio.yaml template should have slack.bot_token_env set")
	}
	if err := config.ValidateEpisodeSpecCast(spec); err != nil {
		t.Fatalf("ValidateEpisodeSpecCast failed: %v", err)
	}
	if err := config.ValidateEpisodeSpecAssets(spec); err != nil {
		t.Fatalf("ValidateEpisodeSpecAssets failed: %v", err)
	}

	// cache フィールドのアサート
	if !cfg.Cache.Enabled {
		t.Error("cfg.Cache.Enabled should be true")
	}
	if cfg.Cache.MaxEntries != config.DefaultCacheMaxEntries {
		t.Errorf("cfg.Cache.MaxEntries = %d, want %d", cfg.Cache.MaxEntries, config.DefaultCacheMaxEntries)
	}
	if cfg.Cache.RetentionDays != config.DefaultCacheRetentionDays {
		t.Errorf("cfg.Cache.RetentionDays = %d, want %d", cfg.Cache.RetentionDays, config.DefaultCacheRetentionDays)
	}
	if cfg.Cache.LLMContextEntries != config.DefaultCacheLLMContextEntries {
		t.Errorf("cfg.Cache.LLMContextEntries = %d, want %d", cfg.Cache.LLMContextEntries, config.DefaultCacheLLMContextEntries)
	}

	// voicevox.presets のアサート
	if cfg.Voicevox.Presets == nil {
		t.Fatal("cfg.Voicevox.Presets should not be nil after init")
	}
	presets := cfg.Voicevox.EffectivePresets()
	if v, ok := presets.ResolveIntonation("標準"); !ok || v != 1.0 {
		t.Errorf("presets.Intonation[標準] = %v (ok=%v), want 1.0", v, ok)
	}
	if v, ok := presets.ResolvePitch("標準"); !ok || v != 0.0 {
		t.Errorf("presets.Pitch[標準] = %v (ok=%v), want 0.0", v, ok)
	}
	if v, ok := presets.ResolveSpeed("標準"); !ok || v != 1.0 {
		t.Errorf("presets.Speed[標準] = %v (ok=%v), want 1.0", v, ok)
	}
}

func TestInitCmd_SlackSpecExists_Skipped(t *testing.T) {
	dir := chdirTemp(t)
	existingContent := []byte("# existing")
	if err := os.WriteFile(filepath.Join(dir, "slack-spec.yaml"), existingContent, 0644); err != nil {
		t.Fatal(err)
	}
	out, err := runInitCmd(t)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	data, _ := os.ReadFile(filepath.Join(dir, "slack-spec.yaml"))
	if string(data) != string(existingContent) {
		t.Error("slack-spec.yaml should not be overwritten")
	}
	if !strings.Contains(out, "skip") {
		t.Errorf("expected skip message for slack-spec.yaml, got: %s", out)
	}
}
