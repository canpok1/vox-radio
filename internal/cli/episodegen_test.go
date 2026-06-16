package cli_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/canpok1/vox-radio/internal/cli"
)

func mustWriteFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatalf("mkdir %s: %v", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, []byte(content), 0600); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func TestEpisodegen_ExistingEpisodeCheck(t *testing.T) {
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		t.Skip("ffmpeg not installed")
	}
	if _, err := exec.LookPath("ffprobe"); err != nil {
		t.Skip("ffprobe not installed")
	}

	// Template episode-spec.yaml has program.id="my-tech-radio"; no cache → episodeNumber=1,
	// so the per-episode base name is "my-tech-radio_ep001".
	tests := []struct {
		name string
		// existing names the artifact pre-created under outDir before running:
		// "" (none), "mp3", "manifest", or "intermediate".
		existing     string
		force        bool
		wantForceErr bool
	}{
		{"mp3既存_--force無し", "mp3", false, true},
		{"mp3既存_--force指定", "mp3", true, false},
		{"manifest既存_--force無し", "manifest", false, true},
		{"manifest既存_--force指定", "manifest", true, false},
		{"中間Dir既存_--force無し", "intermediate", false, true},
		{"中間Dir既存_--force指定", "intermediate", true, false},
		{"既存なし_--force無し", "", false, false},
		{"既存なし_--force指定", "", true, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := chdirTemp(t)
			outDir := filepath.Join(dir, "output")

			// Generate valid config/spec via init command so loadConfigAndSpec succeeds.
			if _, err := runInitCmd(t); err != nil {
				t.Fatalf("init: %v", err)
			}

			switch tt.existing {
			case "mp3":
				mustWriteFile(t, filepath.Join(outDir, "my-tech-radio_ep001.mp3"), "dummy")
			case "manifest":
				mustWriteFile(t, filepath.Join(outDir, "my-tech-radio_ep001_manifest.json"), "{}")
			case "intermediate":
				interDir := filepath.Join(outDir, "intermediate", "my-tech-radio_ep001")
				if err := os.MkdirAll(interDir, 0755); err != nil {
					t.Fatalf("mkdir intermediate: %v", err)
				}
			}

			args := []string{"episodegen", "--spec", "episode-spec.yaml", "--out-dir", outDir}
			if tt.force {
				args = append(args, "--force")
			}

			cmd := cli.NewRootCmd()
			cmd.SetArgs(args)
			err := cmd.Execute()

			if tt.wantForceErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				if !strings.Contains(err.Error(), "--force") {
					t.Errorf("error should mention --force, got: %v", err)
				}
			} else {
				if err != nil && strings.Contains(err.Error(), "--force") {
					t.Errorf("unexpected --force error: %v", err)
				}
			}
		})
	}
}
