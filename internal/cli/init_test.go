package cli_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/canpok1/vox-radio/internal/cli"
	"github.com/canpok1/vox-radio/internal/config"
	"github.com/canpok1/vox-radio/internal/feed"
	"github.com/canpok1/vox-radio/internal/slack"
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

func runInitCmd(t *testing.T, extraArgs ...string) (string, error) {
	t.Helper()
	cmd := cli.NewRootCmd()
	var buf strings.Builder
	cmd.SetOut(&buf)
	cmd.SetArgs(append([]string{"init"}, extraArgs...))
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
	writeTestFile(t, dir, "vox-radio.yaml", existingContent)
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
	writeTestFile(t, dir, "episode-spec.yaml", existingContent)
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
	writeTestFile(t, dir, "feed-spec.yaml", existingContent)
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
	if _, err := feed.LoadFeedSpec(filepath.Join(dir, "feed-spec.yaml")); err != nil {
		t.Fatalf("LoadFeedSpec failed on generated template: %v", err)
	}
	slackSpec, err := slack.LoadSlackSpec(filepath.Join(dir, "slack-spec.yaml"))
	if err != nil {
		t.Fatalf("LoadSlackSpec failed on generated template: %v", err)
	}
	if slackSpec.Slack.Channel == "" {
		t.Error("slack-spec.yaml template should have a channel value")
	}
	if cfg.Slack.BotTokenEnv == "" {
		t.Error("vox-radio.yaml template should have slack.bot_token_env set")
	}
	if err := spec.ValidateProgram(); err != nil {
		t.Fatalf("ValidateProgram failed (template must set program.id): %v", err)
	}
	if err := spec.ValidateCast(); err != nil {
		t.Fatalf("ValidateCast failed: %v", err)
	}
	if err := spec.ValidateAssets(); err != nil {
		t.Fatalf("ValidateAssets failed: %v", err)
	}

	// cache フィールドのアサート
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

func TestInitCmd_Sample_AllGenerated(t *testing.T) {
	dir := chdirTemp(t)
	_, err := runInitCmd(t, "--sample")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for _, name := range []string{
		"sample/vox-radio.yaml",
		"sample/episode-spec.yaml",
		"sample/feed-spec.yaml",
		"sample/slack-spec.yaml",
		"sample/assets/assets.yaml",
	} {
		if _, err := os.Stat(filepath.Join(dir, name)); os.IsNotExist(err) {
			t.Errorf("%s was not generated", name)
		}
	}
}

func TestInitCmd_Sample_Loadable(t *testing.T) {
	dir := chdirTemp(t)
	_, err := runInitCmd(t, "--sample")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	cfg, err := config.LoadConfig(filepath.Join(dir, "sample", "vox-radio.yaml"))
	if err != nil {
		t.Fatalf("LoadConfig failed on sample: %v", err)
	}

	spec, err := config.LoadEpisodeSpecStrict(filepath.Join(dir, "sample", "episode-spec.yaml"))
	if err != nil {
		t.Fatalf("LoadEpisodeSpecStrict failed on sample: %v", err)
	}
	if err := spec.Validate(cfg.Characters); err != nil {
		t.Fatalf("Validate failed on sample: %v", err)
	}

	// 毎回の放送コーナーの length_sec 合計が 300 秒（尺不変）であること。
	// 固定コーナー（condition なし）＋ ローテ枠どれか1つ = 15+180+90+15 = 300。
	fixed := 0
	var rotation []int
	for _, c := range spec.Corners {
		if c.Condition == nil {
			fixed += c.LengthSec
		} else {
			rotation = append(rotation, c.LengthSec)
		}
	}
	if len(rotation) == 0 {
		t.Fatal("rotation corners (with condition) not found")
	}
	for _, r := range rotation {
		if total := fixed + r; total != 300 {
			t.Errorf("放送コーナーの length_sec 合計 = %d, want 300", total)
		}
	}

	// 第1回はローテ枠のうち地震・火山コーナー（every:3, offset:1）のみが採用され、
	// 採用コーナーは 4 つ（固定3＋ローテ1）であること。
	ep1 := config.ResolveCornersForEpisode(spec.Corners, 1)
	if len(ep1) != 4 {
		t.Errorf("第1回の採用コーナー数 = %d, want 4", len(ep1))
	}
	ep1Titles := make(map[string]bool, len(ep1))
	for _, c := range ep1 {
		ep1Titles[c.Title] = true
	}
	if !ep1Titles["地震・火山コーナー"] {
		t.Error("第1回は地震・火山コーナーが放送されるべき")
	}
	if ep1Titles["お天気豆知識"] || ep1Titles["防災ワンポイント"] {
		t.Error("第1回は地震・火山コーナー以外のローテ枠は放送されないべき")
	}

	if _, err := feed.LoadFeedSpec(filepath.Join(dir, "sample", "feed-spec.yaml")); err != nil {
		t.Fatalf("LoadFeedSpec failed on sample: %v", err)
	}
	if _, err := slack.LoadSlackSpec(filepath.Join(dir, "sample", "slack-spec.yaml")); err != nil {
		t.Fatalf("LoadSlackSpec failed on sample: %v", err)
	}
}

func TestInitCmd_Sample_Skip(t *testing.T) {
	dir := chdirTemp(t)
	existingContent := []byte("# existing")
	writeTestFile(t, dir, "sample/episode-spec.yaml", existingContent)
	out, err := runInitCmd(t, "--sample")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	data, _ := os.ReadFile(filepath.Join(dir, "sample", "episode-spec.yaml"))
	if string(data) != string(existingContent) {
		t.Error("sample/episode-spec.yaml should not be overwritten")
	}
	if !strings.Contains(out, "skip") {
		t.Errorf("expected skip message for sample/episode-spec.yaml, got: %s", out)
	}
	// 他のファイルは生成される。
	if _, err := os.Stat(filepath.Join(dir, "sample", "vox-radio.yaml")); os.IsNotExist(err) {
		t.Error("sample/vox-radio.yaml was not generated")
	}
}

func TestInitCmd_SlackSpecExists_Skipped(t *testing.T) {
	dir := chdirTemp(t)
	existingContent := []byte("# existing")
	writeTestFile(t, dir, "slack-spec.yaml", existingContent)
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
