package cli

import (
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/canpok1/vox-radio/internal/model"
)

const manifestTestSpec = `program:
  id: "prog"
  title: "テスト番組"
corners:
  - id: "c1"
    title: "コーナー1"
    length_sec: 30
  - id: "c2"
    title: "コーナー2"
    length_sec: 60
`

// runManifestCmd は 2 コーナー（c1, c2）の spec で manifest コマンドを実行し、生成された manifest を返す。
func runManifestCmd(t *testing.T, dir string, extraArgs ...string) model.Manifest {
	t.Helper()

	specPath := filepath.Join(dir, "episode-spec.yaml")
	if err := os.WriteFile(specPath, []byte(manifestTestSpec), 0644); err != nil {
		t.Fatalf("write spec: %v", err)
	}
	outPath := filepath.Join(dir, "manifest.json")

	args := []string{
		"--spec", specPath,
		"--audio", filepath.Join(dir, "episode.mp3"),
		"--out", outPath,
	}
	args = append(args, extraArgs...)

	cmd := newManifestCmd()
	cmd.SetArgs(args)
	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)
	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute manifest: %v", err)
	}

	m, err := readJSON[model.Manifest](outPath)
	if err != nil {
		t.Fatalf("read manifest: %v", err)
	}
	return m
}

// A corner skipped by the rundown was never broadcast, so it must not appear in the manifest
// as a 0-second corner with no articles.
func TestManifestCmd_ExcludesCornersMissingFromRundown(t *testing.T) {
	dir := chdirTemp(t)

	rundownPath := filepath.Join(dir, "02_rundown.json")
	rd := model.Rundown{Corners: []model.RundownCorner{{ID: "c1", Title: "コーナー1"}}}
	if err := writeJSON(rundownPath, rd); err != nil {
		t.Fatalf("write rundown: %v", err)
	}

	m := runManifestCmd(t, dir, "--rundown", rundownPath)

	if len(m.Corners) != 1 || m.Corners[0].ID != "c1" {
		t.Errorf("Corners = %+v, want 1 entry for c1 (skipped c2 must not appear)", m.Corners)
	}
}

func TestManifestCmd_WithoutRundownKeepsAllSpecCorners(t *testing.T) {
	dir := chdirTemp(t)

	m := runManifestCmd(t, dir)

	if len(m.Corners) != 2 || m.Corners[0].ID != "c1" || m.Corners[1].ID != "c2" {
		t.Errorf("Corners = %+v, want c1 and c2 (spec order)", m.Corners)
	}
}
