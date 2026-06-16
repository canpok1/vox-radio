package cli_test

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/canpok1/vox-radio/internal/cli"
)

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
	cmd.SetArgs([]string{"assets", "check", configTestdataPath("assets_typo.yaml")})
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

func TestAssetsHelp_ListsPreviewSubcommand(t *testing.T) {
	cmd := cli.NewRootCmd()
	buf := &bytes.Buffer{}
	cmd.SetOut(buf)
	cmd.SetArgs([]string{"assets", "--help"})
	err := cmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "\n  preview ") {
		t.Errorf("assets help should list preview subcommand, got:\n%s", out)
	}
	if !strings.Contains(out, "\n  check ") {
		t.Errorf("assets help should list check subcommand, got:\n%s", out)
	}
}

func TestAssetsPreview_MissingIDFlag_Error(t *testing.T) {
	assetsPath := buildValidAssetsYAML(t)
	dir := filepath.Dir(assetsPath)

	cmd := cli.NewRootCmd()
	cmd.SetArgs([]string{"assets", "preview", assetsPath, "--out", filepath.Join(dir, "out.mp3")})
	err := cmd.Execute()
	if err == nil {
		t.Error("expected error when --id is missing")
	}
}

func TestAssetsPreview_MissingOutFlag_Error(t *testing.T) {
	assetsPath := buildValidAssetsYAML(t)

	cmd := cli.NewRootCmd()
	cmd.SetArgs([]string{"assets", "preview", assetsPath, "--id", "bgm:talk"})
	err := cmd.Execute()
	if err == nil {
		t.Error("expected error when --out is missing")
	}
}

func TestAssetsPreview_MissingPathArg_Error(t *testing.T) {
	cmd := cli.NewRootCmd()
	cmd.SetArgs([]string{"assets", "preview", "--id", "bgm:talk", "--out", "/tmp/out.mp3"})
	err := cmd.Execute()
	if err == nil {
		t.Error("expected error when assets.yaml path argument is missing")
	}
}

func TestAssetsPreview_InvalidIDFormat_Error(t *testing.T) {
	assetsPath := buildValidAssetsYAML(t)
	dir := filepath.Dir(assetsPath)

	cmd := cli.NewRootCmd()
	cmd.SetArgs([]string{"assets", "preview", assetsPath, "--id", "invalidformat", "--out", filepath.Join(dir, "out.mp3")})
	err := cmd.Execute()
	if err == nil {
		t.Error("expected error for --id without colon separator")
	}
}

func TestAssetsPreview_InvalidType_Error(t *testing.T) {
	assetsPath := buildValidAssetsYAML(t)
	dir := filepath.Dir(assetsPath)

	cmd := cli.NewRootCmd()
	cmd.SetArgs([]string{"assets", "preview", assetsPath, "--id", "badtype:opening", "--out", filepath.Join(dir, "out.mp3")})
	err := cmd.Execute()
	if err == nil {
		t.Error("expected error for unknown asset type in --id")
	}
}

// TestAssetsPreview_SuppressesFFmpegLog_Success verifies that a successful preview
// does not leak ffmpeg log output to the terminal (stderr) and instead prints a
// concise user-facing success message to stdout, matching assemble/episodegen.
func TestAssetsPreview_SuppressesFFmpegLog_Success(t *testing.T) {
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		t.Skip("ffmpeg not installed")
	}

	dir := t.TempDir()
	jinglePath := filepath.Join(dir, "opening.wav")
	if err := exec.Command("ffmpeg", "-f", "lavfi", "-i", "sine=frequency=440:duration=1", jinglePath).Run(); err != nil {
		t.Fatalf("generate test audio: %v", err)
	}

	yamlContent := "jingle:\n" +
		"  opening:\n" +
		"    file: opening.wav\n"
	assetsPath := filepath.Join(dir, "assets.yaml")
	if err := os.WriteFile(assetsPath, []byte(yamlContent), 0600); err != nil {
		t.Fatalf("create assets.yaml: %v", err)
	}

	outPath := filepath.Join(dir, "preview.mp3")
	logDir := filepath.Join(dir, "logs")

	cmd := cli.NewRootCmd()
	outBuf := &bytes.Buffer{}
	errBuf := &bytes.Buffer{}
	cmd.SetOut(outBuf)
	cmd.SetErr(errBuf)
	cmd.SetArgs([]string{"assets", "preview", assetsPath, "--id", "jingle:opening", "--out", outPath, "--log-dir", logDir})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v\nstderr: %s", err, errBuf.String())
	}

	// ffmpeg log must NOT leak to the terminal (stderr).
	if strings.Contains(errBuf.String(), "ffmpeg") {
		t.Errorf("ffmpeg log should not appear on stderr, got: %s", errBuf.String())
	}
	// A user-facing success message with the output path must appear on stdout.
	if !strings.Contains(outBuf.String(), outPath) {
		t.Errorf("expected success message with output path on stdout, got: %s", outBuf.String())
	}
	// The preview file must be created.
	if _, err := os.Stat(outPath); err != nil {
		t.Errorf("preview output not created: %v", err)
	}
}

func TestAssetsPreview_KeyNotFound_Error(t *testing.T) {
	assetsPath := buildValidAssetsYAML(t)
	dir := filepath.Dir(assetsPath)

	cmd := cli.NewRootCmd()
	cmd.SetArgs([]string{"assets", "preview", assetsPath, "--id", "jingle:nonexistent", "--out", filepath.Join(dir, "out.mp3")})
	err := cmd.Execute()
	if err == nil {
		t.Error("expected error for nonexistent key in assets.yaml")
	}
}
