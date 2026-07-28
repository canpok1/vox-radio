package cli_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/canpok1/vox-radio/internal/cache"
	"github.com/canpok1/vox-radio/internal/cli"
)

func runRetroCmd(t *testing.T, extraArgs ...string) (string, error) {
	t.Helper()
	cmd := cli.NewRootCmd()
	var buf strings.Builder
	cmd.SetOut(&buf)
	args := append([]string{"retro", "--spec", "episode-spec.yaml"}, extraArgs...)
	cmd.SetArgs(args)
	err := cmd.Execute()
	return buf.String(), err
}

func setSingleShot(t *testing.T, dir string) {
	t.Helper()
	path := filepath.Join(dir, "episode-spec.yaml")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read episode-spec.yaml: %v", err)
	}
	updated := strings.Replace(string(data), `id: "my-tech-radio"`, "id: \"my-tech-radio\"\n  single_shot: true", 1)
	if updated == string(data) {
		t.Fatal("failed to inject single_shot: true into episode-spec.yaml (template changed?)")
	}
	if err := os.WriteFile(path, []byte(updated), 0644); err != nil {
		t.Fatalf("write episode-spec.yaml: %v", err)
	}
}

func TestRetroCmd_SingleShotProgram_ExitsCleanlyWithoutLLMCall(t *testing.T) {
	t.Setenv("GEMINI_API_KEY", "dummy")
	dir := chdirTemp(t)
	if _, err := runInitCmd(t); err != nil {
		t.Fatalf("init: %v", err)
	}
	setSingleShot(t, dir)

	out, err := runRetroCmd(t)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, "single_shot") {
		t.Errorf("output should mention single_shot, got: %q", out)
	}
}

func TestRetroCmd_NoCache_ExitsCleanlyWithoutLLMCall(t *testing.T) {
	t.Setenv("GEMINI_API_KEY", "dummy")
	chdirTemp(t)
	if _, err := runInitCmd(t); err != nil {
		t.Fatalf("init: %v", err)
	}

	out, err := runRetroCmd(t)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, "キャッシュが存在しません") {
		t.Errorf("output should mention missing cache, got: %q", out)
	}
}

func TestRetroCmd_NoAnalyzedEntries_ExitsCleanlyWithoutLLMCall(t *testing.T) {
	t.Setenv("GEMINI_API_KEY", "dummy")
	dir := chdirTemp(t)
	if _, err := runInitCmd(t); err != nil {
		t.Fatalf("init: %v", err)
	}

	// my-tech-radio is the template's program.id; write a cache entry without Analysis
	// (as if produced before analyze was introduced).
	cachePath := filepath.Join(dir, ".vox-radio", "programs", "my-tech-radio", "cache.jsonl")
	mgr := cache.New(cachePath)
	if err := mgr.Append(cache.Entry{ProgramID: "my-tech-radio", EpisodeNumber: 1, Datetime: "2026-01-01T00:00:00Z"}, 100, 90); err != nil {
		t.Fatalf("append cache: %v", err)
	}

	out, err := runRetroCmd(t)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, "分析付きのエピソードがありません") {
		t.Errorf("output should mention no analyzed episodes, got: %q", out)
	}
}
