package cli_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/canpok1/vox-radio/internal/cli"
)

func runInstallCmd(t *testing.T, args ...string) (string, error) {
	t.Helper()
	cmd := cli.NewRootCmd()
	var buf strings.Builder
	cmd.SetOut(&buf)
	cmd.SetArgs(append([]string{"install"}, args...))
	err := cmd.Execute()
	return buf.String(), err
}

func TestInstallCmd_SkillsGenerated(t *testing.T) {
	dir := chdirTemp(t)
	_, err := runInstallCmd(t, "--skills")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expectedFiles := []string{
		".claude/skills/vox-radio/SKILL.md",
		".claude/skills/vox-radio/references/vox-radio.md",
		".claude/skills/vox-radio/references/episode-spec.md",
		".claude/skills/vox-radio/references/assets.md",
		".claude/skills/vox-radio/references/feed-spec.md",
		".claude/skills/vox-radio/references/slack-spec.md",
	}
	for _, name := range expectedFiles {
		if _, err := os.Stat(filepath.Join(dir, name)); os.IsNotExist(err) {
			t.Errorf("%s was not generated", name)
		}
	}
}

func TestInstallCmd_SkillsExistingSkipped(t *testing.T) {
	dir := chdirTemp(t)
	skillDir := filepath.Join(dir, ".claude/skills/vox-radio")
	if err := os.MkdirAll(skillDir, 0755); err != nil {
		t.Fatal(err)
	}
	existing := filepath.Join(skillDir, "SKILL.md")
	existingContent := []byte("# existing")
	if err := os.WriteFile(existing, existingContent, 0644); err != nil {
		t.Fatal(err)
	}
	out, err := runInstallCmd(t, "--skills")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	data, _ := os.ReadFile(existing)
	if string(data) != string(existingContent) {
		t.Error("SKILL.md should not be overwritten without --force")
	}
	if !strings.Contains(out, "skip") {
		t.Errorf("expected skip message for SKILL.md, got: %s", out)
	}
}

func TestInstallCmd_SkillsForceOverwrites(t *testing.T) {
	dir := chdirTemp(t)
	skillDir := filepath.Join(dir, ".claude/skills/vox-radio")
	if err := os.MkdirAll(skillDir, 0755); err != nil {
		t.Fatal(err)
	}
	existing := filepath.Join(skillDir, "SKILL.md")
	existingContent := []byte("# old content")
	if err := os.WriteFile(existing, existingContent, 0644); err != nil {
		t.Fatal(err)
	}
	_, err := runInstallCmd(t, "--skills", "--force")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	data, _ := os.ReadFile(existing)
	if string(data) == string(existingContent) {
		t.Error("SKILL.md should be overwritten with --force")
	}
}

func TestInstallCmd_SkillMdFrontmatter(t *testing.T) {
	dir := chdirTemp(t)
	_, err := runInstallCmd(t, "--skills")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	data, err := os.ReadFile(filepath.Join(dir, ".claude/skills/vox-radio/SKILL.md"))
	if err != nil {
		t.Fatalf("read SKILL.md: %v", err)
	}
	content := string(data)
	for _, want := range []string{"---", "name: vox-radio", "allowed-tools:"} {
		if !strings.Contains(content, want) {
			t.Errorf("SKILL.md frontmatter missing %q", want)
		}
	}
}
