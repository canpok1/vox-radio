package cli_test

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/canpok1/vox-radio/internal/cli"
)

func assetsTestdataPath(rel string) string {
	return filepath.Join(cliTestSrcDir, "..", "config", "testdata", rel)
}

// buildValidAssetsYAML creates a temp dir with stub audio files and an assets.yaml referencing them.
// Returns the path to the created assets.yaml.
func buildValidAssetsYAML(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()

	jinglePath := filepath.Join(dir, "opening.mp3")
	sePath := filepath.Join(dir, "chime.wav")
	bgmPath := filepath.Join(dir, "talk.mp3")
	for _, f := range []string{jinglePath, sePath, bgmPath} {
		if err := os.WriteFile(f, []byte{}, 0600); err != nil {
			t.Fatalf("create stub file: %v", err)
		}
	}

	yamlContent := "jingle:\n" +
		"  opening:\n" +
		"    file: opening.mp3\n" +
		"    fade_in: 0.5\n" +
		"    fade_out: 0.5\n" +
		"se:\n" +
		"  chime:\n" +
		"    file: chime.wav\n" +
		"    volume: 0.8\n" +
		"bgm:\n" +
		"  talk:\n" +
		"    file: talk.mp3\n" +
		"    volume: 0.3\n" +
		"    duck_ratio: 8\n" +
		"    loop: true\n"

	assetsPath := filepath.Join(dir, "assets.yaml")
	if err := os.WriteFile(assetsPath, []byte(yamlContent), 0600); err != nil {
		t.Fatalf("create assets.yaml: %v", err)
	}
	return assetsPath
}

// buildFileMissingAssetsYAML creates an assets.yaml referencing a non-existent file.
func buildFileMissingAssetsYAML(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()

	yamlContent := "jingle:\n" +
		"  opening:\n" +
		"    file: nonexistent.mp3\n" +
		"    fade_in: 0.5\n" +
		"    fade_out: 0.5\n"

	assetsPath := filepath.Join(dir, "assets.yaml")
	if err := os.WriteFile(assetsPath, []byte(yamlContent), 0600); err != nil {
		t.Fatalf("create assets.yaml: %v", err)
	}
	return assetsPath
}

// buildInvalidValueAssetsYAML creates an assets.yaml with duck_ratio < 1.
func buildInvalidValueAssetsYAML(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()

	bgmPath := filepath.Join(dir, "talk.mp3")
	if err := os.WriteFile(bgmPath, []byte{}, 0600); err != nil {
		t.Fatalf("create stub file: %v", err)
	}

	yamlContent := "bgm:\n" +
		"  talk:\n" +
		"    file: talk.mp3\n" +
		"    volume: 0.3\n" +
		"    duck_ratio: 0\n" +
		"    loop: true\n"

	assetsPath := filepath.Join(dir, "assets.yaml")
	if err := os.WriteFile(assetsPath, []byte(yamlContent), 0600); err != nil {
		t.Fatalf("create assets.yaml: %v", err)
	}
	return assetsPath
}

func TestAssetsCheck_ValidYAML_Success(t *testing.T) {
	assetsPath := buildValidAssetsYAML(t)

	cmd := cli.NewRootCmd()
	buf := &bytes.Buffer{}
	cmd.SetOut(buf)
	cmd.SetArgs([]string{"assets", "check", assetsPath})
	err := cmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(buf.String(), "OK") {
		t.Errorf("expected OK in output, got: %s", buf.String())
	}
}

func TestAssetsCheck_Typo_Error(t *testing.T) {
	cmd := cli.NewRootCmd()
	cmd.SetArgs([]string{"assets", "check", assetsTestdataPath("assets_typo.yaml")})
	err := cmd.Execute()
	if err == nil {
		t.Error("expected error for assets.yaml with unknown key (typo), got nil")
	}
}

func TestAssetsCheck_FileMissing_Error(t *testing.T) {
	assetsPath := buildFileMissingAssetsYAML(t)

	cmd := cli.NewRootCmd()
	cmd.SetArgs([]string{"assets", "check", assetsPath})
	err := cmd.Execute()
	if err == nil {
		t.Error("expected error for assets.yaml referencing non-existent file, got nil")
	}
}

func TestAssetsCheck_InvalidValue_Error(t *testing.T) {
	assetsPath := buildInvalidValueAssetsYAML(t)

	cmd := cli.NewRootCmd()
	cmd.SetArgs([]string{"assets", "check", assetsPath})
	err := cmd.Execute()
	if err == nil {
		t.Error("expected error for assets.yaml with invalid duck_ratio, got nil")
	}
}

func TestAssetsCheck_MissingArg_Error(t *testing.T) {
	cmd := cli.NewRootCmd()
	cmd.SetArgs([]string{"assets", "check"})
	err := cmd.Execute()
	if err == nil {
		t.Error("expected error when PATH argument is missing")
	}
}
