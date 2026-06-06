package cli_test

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/canpok1/vox-radio/internal/cli"
)

// setupEpisodegenCheckDir creates a temp dir with vox-radio.yaml and changes cwd to it.
func setupEpisodegenCheckDir(t *testing.T, configSrc string) {
	t.Helper()
	dir := chdirTemp(t)

	data, err := os.ReadFile(configSrc)
	if err != nil {
		t.Fatalf("read config src: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "vox-radio.yaml"), data, 0600); err != nil {
		t.Fatalf("write vox-radio.yaml: %v", err)
	}
}

func TestEpisodegenCheck_ValidSpec_Success(t *testing.T) {
	setupEpisodegenCheckDir(t, configTestdataPath("config.yaml"))

	cmd := cli.NewRootCmd()
	buf := &bytes.Buffer{}
	cmd.SetOut(buf)
	cmd.SetArgs([]string{"episodegen", "check", configTestdataPath("episode_spec.yaml")})
	err := cmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(buf.String(), "OK") {
		t.Errorf("expected OK in output, got: %s", buf.String())
	}
}

func TestEpisodegenCheck_UnknownKey_Error(t *testing.T) {
	setupEpisodegenCheckDir(t, configTestdataPath("config.yaml"))

	cmd := cli.NewRootCmd()
	cmd.SetArgs([]string{"episodegen", "check", configTestdataPath("episode_spec_unknown_key.yaml")})
	err := cmd.Execute()
	if err == nil {
		t.Error("expected error for unknown key in strict mode")
	}
}

func TestEpisodegenCheck_UnknownCast_Error(t *testing.T) {
	setupEpisodegenCheckDir(t, configTestdataPath("config.yaml"))

	// create a spec with an unknown cast character
	dir := t.TempDir()
	specContent := []byte(`program:
  id: "test-program"
  title: "テスト"
  description: "テスト"

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
	specPath := filepath.Join(dir, "episode_spec_bad_cast.yaml")
	if err := os.WriteFile(specPath, specContent, 0600); err != nil {
		t.Fatal(err)
	}

	cmd := cli.NewRootCmd()
	cmd.SetArgs([]string{"episodegen", "check", specPath})
	err := cmd.Execute()
	if err == nil {
		t.Error("expected error for unknown cast character")
	}
}

func TestEpisodegenCheck_MissingProgramID_Error(t *testing.T) {
	setupEpisodegenCheckDir(t, configTestdataPath("config.yaml"))

	dir := t.TempDir()
	specContent := []byte(`program:
  title: "テスト"
  description: "テスト"

corners: []

assets:
  jingle: {}
  se: {}
  bgm: {}
`)
	specPath := filepath.Join(dir, "episode_spec_no_id.yaml")
	if err := os.WriteFile(specPath, specContent, 0600); err != nil {
		t.Fatal(err)
	}

	cmd := cli.NewRootCmd()
	cmd.SetArgs([]string{"episodegen", "check", specPath})
	if err := cmd.Execute(); err == nil {
		t.Error("expected error when program.id is missing")
	}
}

func TestEpisodegenCheck_MissingConfig_Error(t *testing.T) {
	chdirTemp(t) // no vox-radio.yaml in cwd

	cmd := cli.NewRootCmd()
	cmd.SetArgs([]string{"episodegen", "check", configTestdataPath("episode_spec.yaml")})
	err := cmd.Execute()
	if err == nil {
		t.Error("expected error when vox-radio.yaml is missing in cwd")
	}
}

func TestEpisodegenCheck_MissingSpecArg_Error(t *testing.T) {
	cmd := cli.NewRootCmd()
	cmd.SetArgs([]string{"episodegen", "check"})
	err := cmd.Execute()
	if err == nil {
		t.Error("expected error when episode spec path argument is missing")
	}
}

func TestEpisodegenCheck_AssetsTypo_Error(t *testing.T) {
	setupEpisodegenCheckDir(t, configTestdataPath("config.yaml"))

	cmd := cli.NewRootCmd()
	cmd.SetArgs([]string{"episodegen", "check", configTestdataPath("episode_spec_with_typo_assets.yaml")})
	err := cmd.Execute()
	if err == nil {
		t.Error("expected error when assets_files contains typo in strict mode, got nil")
	}
}

func TestEpisodegenCheck_ConfigFlag_DifferentDir(t *testing.T) {
	chdirTemp(t) // cwd に vox-radio.yaml がない状態でも --config で別ディレクトリを指定できる

	cmd := cli.NewRootCmd()
	buf := &bytes.Buffer{}
	cmd.SetOut(buf)
	cmd.SetArgs([]string{"episodegen", "check",
		"--config", configTestdataPath("config.yaml"),
		configTestdataPath("episode_spec.yaml"),
	})
	err := cmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error with --config: %v", err)
	}
	if !strings.Contains(buf.String(), "OK") {
		t.Errorf("expected OK in output, got: %s", buf.String())
	}
}
