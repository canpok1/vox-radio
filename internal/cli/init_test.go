package cli_test

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/canpok1/vox-radio/internal/cli"
	"github.com/canpok1/vox-radio/internal/config"
	"github.com/canpok1/vox-radio/internal/feed"
	"github.com/canpok1/vox-radio/internal/slack"
)

var initTemplateFiles = []string{"vox-radio.yaml", "episode-spec.yaml", "feed-spec.yaml", "slack-spec.yaml", "assets/assets.yaml"}

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
	for _, name := range initTemplateFiles {
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
	for _, name := range initTemplateFiles {
		writeTestFile(t, dir, name, existingContent)
	}
	_, err := runInitCmd(t)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for _, name := range initTemplateFiles {
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

// TestInitCmd_Sample_AllGenerated: --sample alone outputs to current dir (breaking change).
func TestInitCmd_Sample_AllGenerated(t *testing.T) {
	dir := chdirTemp(t)
	_, err := runInitCmd(t, "--sample")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for _, name := range initTemplateFiles {
		if _, err := os.Stat(filepath.Join(dir, name)); os.IsNotExist(err) {
			t.Errorf("%s was not generated", name)
		}
	}
}

// TestInitCmd_Sample_WithOutputDir: --sample --output-dir sample reproduces the old behavior.
func TestInitCmd_Sample_WithOutputDir(t *testing.T) {
	dir := chdirTemp(t)
	_, err := runInitCmd(t, "--sample", "--output-dir", "sample")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for _, name := range initTemplateFiles {
		if _, err := os.Stat(filepath.Join(dir, "sample", name)); os.IsNotExist(err) {
			t.Errorf("sample/%s was not generated", name)
		}
	}
}

func TestInitCmd_Sample_Loadable(t *testing.T) {
	dir := chdirTemp(t)
	_, err := runInitCmd(t, "--sample", "--output-dir", "sample")
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
	writeTestFile(t, dir, "episode-spec.yaml", existingContent)
	out, err := runInitCmd(t, "--sample")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	data, _ := os.ReadFile(filepath.Join(dir, "episode-spec.yaml"))
	if string(data) != string(existingContent) {
		t.Error("episode-spec.yaml should not be overwritten")
	}
	if !strings.Contains(out, "skip") {
		t.Errorf("expected skip message for episode-spec.yaml, got: %s", out)
	}
	// 他のファイルは生成される。
	if _, err := os.Stat(filepath.Join(dir, "vox-radio.yaml")); os.IsNotExist(err) {
		t.Error("vox-radio.yaml was not generated")
	}
}

// TestInitCmd_EpisodeSpecMatchesGolden verifies that --sample and --sample-with-assets each produce
// an episode-spec.yaml byte-identical to the corresponding golden file in testdata/.
func TestInitCmd_EpisodeSpecMatchesGolden(t *testing.T) {
	for _, tc := range []struct {
		flag   string
		golden string
	}{
		{"--sample", "testdata/episode-spec-without-assets.yaml"},
		{"--sample-with-assets", "testdata/episode-spec-with-assets.yaml"},
	} {
		t.Run(tc.flag, func(t *testing.T) {
			// Read golden before chdirTemp changes the working directory.
			want, err := os.ReadFile(tc.golden)
			if err != nil {
				t.Fatalf("read golden: %v", err)
			}
			dir := chdirTemp(t)
			if _, err := runInitCmd(t, tc.flag); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			got, err := os.ReadFile(filepath.Join(dir, "episode-spec.yaml"))
			if err != nil {
				t.Fatalf("read generated: %v", err)
			}
			if !bytes.Equal(got, want) {
				t.Errorf("%s episode-spec.yaml does not match %s golden\ngot:\n%s\nwant:\n%s", tc.flag, tc.golden, got, want)
			}
		})
	}
}

// TestInitCmd_SampleWithAssets_OmitsAssetsYaml: --sample-with-assets generates the shared
// config files but not assets/assets.yaml (the sample-assets pack provides it).
func TestInitCmd_SampleWithAssets_OmitsAssetsYaml(t *testing.T) {
	dir := chdirTemp(t)
	_, err := runInitCmd(t, "--sample-with-assets")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for _, name := range []string{"vox-radio.yaml", "episode-spec.yaml", "feed-spec.yaml", "slack-spec.yaml"} {
		if _, err := os.Stat(filepath.Join(dir, name)); os.IsNotExist(err) {
			t.Errorf("%s was not generated", name)
		}
	}
	if _, err := os.Stat(filepath.Join(dir, "assets", "assets.yaml")); !os.IsNotExist(err) {
		t.Error("assets/assets.yaml should not be generated for --sample-with-assets (pack provides it)")
	}
}

// TestInitCmd_SampleWithAssets_ValidatesAgainstPack: the generated episode-spec references the
// pack ids (theme/switch/coffee_break) and validates once the pack's assets.yaml is present.
func TestInitCmd_SampleWithAssets_ValidatesAgainstPack(t *testing.T) {
	dir := chdirTemp(t)
	_, err := runInitCmd(t, "--sample-with-assets")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// パック展開を模した最小 assets.yaml を用意する（theme/switch/coffee_break）。
	assetsYAML := "jingle:\n  theme:\n    file: theme.mp3\n" +
		"se:\n  switch:\n    file: switch.mp3\n    volume: 0.8\n" +
		"bgm:\n  coffee_break:\n    file: bgm.mp3\n    volume: 0.3\n    duck_ratio: 0\n    loop: true\n"
	writeTestFile(t, dir, "assets/assets.yaml", []byte(assetsYAML))

	cfg, err := config.LoadConfig(filepath.Join(dir, "vox-radio.yaml"))
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}
	spec, err := config.LoadEpisodeSpecStrict(filepath.Join(dir, "episode-spec.yaml"))
	if err != nil {
		t.Fatalf("LoadEpisodeSpecStrict failed: %v", err)
	}
	if err := spec.Validate(cfg.Characters); err != nil {
		t.Fatalf("Validate failed (episode-spec must reference pack assets): %v", err)
	}
}

// TestInitCmd_SampleAndSampleWithAssets_Conflict: the two sample flags are mutually exclusive.
func TestInitCmd_SampleAndSampleWithAssets_Conflict(t *testing.T) {
	chdirTemp(t)
	if _, err := runInitCmd(t, "--sample", "--sample-with-assets"); err == nil {
		t.Fatal("expected error when both --sample and --sample-with-assets are set")
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

// TestInitCmd_OutputDir_Generated: --output-dir DIR outputs to DIR/.
func TestInitCmd_OutputDir_Generated(t *testing.T) {
	dir := chdirTemp(t)
	_, err := runInitCmd(t, "--output-dir", "mydir")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for _, name := range initTemplateFiles {
		if _, err := os.Stat(filepath.Join(dir, "mydir", name)); os.IsNotExist(err) {
			t.Errorf("mydir/%s was not generated", name)
		}
	}
}

// TestInitCmd_OutputDir_Empty_FallsbackToCurrent: --output-dir "" falls back to current dir.
func TestInitCmd_OutputDir_Empty_FallsbackToCurrent(t *testing.T) {
	dir := chdirTemp(t)
	_, err := runInitCmd(t, "--output-dir", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for _, name := range initTemplateFiles {
		if _, err := os.Stat(filepath.Join(dir, name)); os.IsNotExist(err) {
			t.Errorf("%s was not generated", name)
		}
	}
}

// TestInitCmd_OutputDir_NotExist_AutoCreated: non-existent output dir is auto-created.
func TestInitCmd_OutputDir_NotExist_AutoCreated(t *testing.T) {
	dir := chdirTemp(t)
	_, err := runInitCmd(t, "--output-dir", "new/nested/dir")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "new/nested/dir", "vox-radio.yaml")); os.IsNotExist(err) {
		t.Error("vox-radio.yaml was not generated in nested directory")
	}
}
