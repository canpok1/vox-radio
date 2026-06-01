package cli_test

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/canpok1/vox-radio/internal/cli"
)

func profileTestdataPath(rel string) string {
	return filepath.Join(cliTestSrcDir, "..", "config", "testdata", rel)
}

// setupProfileCheckDir creates a temp dir with vox-radio.yaml and changes cwd to it.
// Returns the absolute path of the vox-radio.yaml placed in the temp dir.
func setupProfileCheckDir(t *testing.T, configSrc string) string {
	t.Helper()
	orig, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(orig) })

	data, err := os.ReadFile(configSrc)
	if err != nil {
		t.Fatalf("read config src: %v", err)
	}
	dst := filepath.Join(dir, "vox-radio.yaml")
	if err := os.WriteFile(dst, data, 0600); err != nil {
		t.Fatalf("write vox-radio.yaml: %v", err)
	}
	return dst
}

func TestProfileCheck_ValidProfile_Success(t *testing.T) {
	setupProfileCheckDir(t, profileTestdataPath("config.yaml"))

	cmd := cli.NewRootCmd()
	buf := &bytes.Buffer{}
	cmd.SetOut(buf)
	cmd.SetArgs([]string{"profile", "check", profileTestdataPath("profile.yaml")})
	err := cmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(buf.String(), "OK") {
		t.Errorf("expected OK in output, got: %s", buf.String())
	}
}

func TestProfileCheck_UnknownKey_Error(t *testing.T) {
	setupProfileCheckDir(t, profileTestdataPath("config.yaml"))

	cmd := cli.NewRootCmd()
	cmd.SetArgs([]string{"profile", "check", profileTestdataPath("profile_unknown_key.yaml")})
	err := cmd.Execute()
	if err == nil {
		t.Error("expected error for unknown key in strict mode")
	}
}

func TestProfileCheck_UnknownCast_Error(t *testing.T) {
	setupProfileCheckDir(t, profileTestdataPath("config.yaml"))

	// create a profile with an unknown cast character
	dir := t.TempDir()
	profileContent := []byte(`program:
  title: "テスト"
  description: "テスト"
  segment_pause_sec: 0.3
  length_sec: 60

corners:
  - title: "コーナー1"
    content: "内容"
    cast:
      unknown_character: "役割"
    length_sec: 60

assets:
  jingle: {}
  se: {}
  bgm: {}
`)
	profilePath := filepath.Join(dir, "profile_bad_cast.yaml")
	if err := os.WriteFile(profilePath, profileContent, 0600); err != nil {
		t.Fatal(err)
	}

	cmd := cli.NewRootCmd()
	cmd.SetArgs([]string{"profile", "check", profilePath})
	err := cmd.Execute()
	if err == nil {
		t.Error("expected error for unknown cast character")
	}
}

func TestProfileCheck_MissingConfig_Error(t *testing.T) {
	orig, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(orig) })

	// no vox-radio.yaml in cwd
	cmd := cli.NewRootCmd()
	cmd.SetArgs([]string{"profile", "check", profileTestdataPath("profile.yaml")})
	err = cmd.Execute()
	if err == nil {
		t.Error("expected error when vox-radio.yaml is missing in cwd")
	}
}

func TestProfileCheck_MissingProfileArg_Error(t *testing.T) {
	cmd := cli.NewRootCmd()
	cmd.SetArgs([]string{"profile", "check"})
	err := cmd.Execute()
	if err == nil {
		t.Error("expected error when profile path argument is missing")
	}
}
