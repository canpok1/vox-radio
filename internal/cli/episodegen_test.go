package cli_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/canpok1/vox-radio/internal/cli"
)

func TestEpisodegen_ExistingEpisodeCheck(t *testing.T) {
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		t.Skip("ffmpeg not installed")
	}
	if _, err := exec.LookPath("ffprobe"); err != nil {
		t.Skip("ffprobe not installed")
	}

	tests := []struct {
		name          string
		episodeExists bool
		force         bool
		wantForceErr  bool
	}{
		{"既存あり_--force無し", true, false, true},
		{"既存あり_--force指定", true, true, false},
		{"既存なし_--force無し", false, false, false},
		{"既存なし_--force指定", false, true, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := chdirTemp(t)
			outDir := filepath.Join(dir, "output")

			// Generate valid config/spec via init command so loadConfigAndSpec succeeds.
			if _, err := runInitCmd(t); err != nil {
				t.Fatalf("init: %v", err)
			}

			if tt.episodeExists {
				if err := os.MkdirAll(outDir, 0755); err != nil {
					t.Fatalf("mkdir: %v", err)
				}
				// Template episode-spec.yaml has program.id="my-tech-radio"; no cache → episodeNumber=1.
				if err := os.WriteFile(filepath.Join(outDir, "my-tech-radio_ep001.mp3"), []byte("dummy"), 0600); err != nil {
					t.Fatalf("write episode: %v", err)
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
