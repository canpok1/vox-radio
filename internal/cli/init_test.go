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
	for _, name := range []string{"vox-radio.yaml", "profile.yaml", "feedgen.yaml"} {
		if _, err := os.Stat(filepath.Join(dir, name)); os.IsNotExist(err) {
			t.Errorf("%s was not generated", name)
		}
	}
}

func TestInitCmd_ConfigExists_ProfileGenerated(t *testing.T) {
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
	if _, err := os.Stat(filepath.Join(dir, "profile.yaml")); os.IsNotExist(err) {
		t.Error("profile.yaml was not generated")
	}
	if !strings.Contains(out, "skip") {
		t.Errorf("expected skip message for vox-radio.yaml, got: %s", out)
	}
}

func TestInitCmd_ProfileExists_ConfigGenerated(t *testing.T) {
	dir := chdirTemp(t)
	existingContent := []byte("# existing")
	if err := os.WriteFile(filepath.Join(dir, "profile.yaml"), existingContent, 0644); err != nil {
		t.Fatal(err)
	}
	out, err := runInitCmd(t)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	data, _ := os.ReadFile(filepath.Join(dir, "profile.yaml"))
	if string(data) != string(existingContent) {
		t.Error("profile.yaml should not be overwritten")
	}
	if _, err := os.Stat(filepath.Join(dir, "vox-radio.yaml")); os.IsNotExist(err) {
		t.Error("vox-radio.yaml was not generated")
	}
	if !strings.Contains(out, "skip") {
		t.Errorf("expected skip message for profile.yaml, got: %s", out)
	}
}

func TestInitCmd_AllExist_NothingGenerated(t *testing.T) {
	dir := chdirTemp(t)
	existingContent := []byte("# existing")
	for _, name := range []string{"vox-radio.yaml", "profile.yaml", "feedgen.yaml"} {
		if err := os.WriteFile(filepath.Join(dir, name), existingContent, 0644); err != nil {
			t.Fatal(err)
		}
	}
	_, err := runInitCmd(t)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for _, name := range []string{"vox-radio.yaml", "profile.yaml", "feedgen.yaml"} {
		data, _ := os.ReadFile(filepath.Join(dir, name))
		if string(data) != string(existingContent) {
			t.Errorf("%s should not be overwritten", name)
		}
	}
}

func TestInitCmd_FeedgenExists_Skipped(t *testing.T) {
	dir := chdirTemp(t)
	existingContent := []byte("# existing")
	if err := os.WriteFile(filepath.Join(dir, "feedgen.yaml"), existingContent, 0644); err != nil {
		t.Fatal(err)
	}
	out, err := runInitCmd(t)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	data, _ := os.ReadFile(filepath.Join(dir, "feedgen.yaml"))
	if string(data) != string(existingContent) {
		t.Error("feedgen.yaml should not be overwritten")
	}
	if !strings.Contains(out, "skip") {
		t.Errorf("expected skip message for feedgen.yaml, got: %s", out)
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
	profile, err := config.LoadProfile(filepath.Join(dir, "profile.yaml"))
	if err != nil {
		t.Fatalf("LoadProfile failed on generated template: %v", err)
	}
	if _, err := model.LoadFeedgen(filepath.Join(dir, "feedgen.yaml")); err != nil {
		t.Fatalf("LoadFeedgen failed on generated template: %v", err)
	}
	if err := config.ValidateProfileCast(profile, cfg.Characters); err != nil {
		t.Fatalf("ValidateProfileCast failed: %v", err)
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
