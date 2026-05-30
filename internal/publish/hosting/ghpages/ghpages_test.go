package ghpages_test

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/canpok1/vox-radio/internal/model"
	"github.com/canpok1/vox-radio/internal/publish/hosting/ghpages"
)

func TestHosting_PutAudio(t *testing.T) {
	dir := t.TempDir()
	h := ghpages.New(dir, "https://owner.github.io/repo")

	content := []byte("fake mp3 data")
	url, err := h.PutAudio(context.Background(), "episode_2026-05-30.mp3", bytes.NewReader(content))
	if err != nil {
		t.Fatalf("PutAudio: %v", err)
	}

	want := "https://owner.github.io/repo/audio/episode_2026-05-30.mp3"
	if url != want {
		t.Errorf("url = %q, want %q", url, want)
	}

	got, err := os.ReadFile(filepath.Join(dir, "audio", "episode_2026-05-30.mp3"))
	if err != nil {
		t.Fatalf("read file: %v", err)
	}
	if !bytes.Equal(got, content) {
		t.Errorf("file content mismatch")
	}
}

func TestHosting_PutAudio_TrailingSlash(t *testing.T) {
	dir := t.TempDir()
	h := ghpages.New(dir, "https://owner.github.io/repo/")

	url, err := h.PutAudio(context.Background(), "ep.mp3", bytes.NewReader([]byte{}))
	if err != nil {
		t.Fatalf("PutAudio: %v", err)
	}

	if strings.Contains(url, "//audio/") {
		t.Errorf("double slash in url: %q", url)
	}
	want := "https://owner.github.io/repo/audio/ep.mp3"
	if url != want {
		t.Errorf("url = %q, want %q", url, want)
	}
}

func TestHosting_PutFeed(t *testing.T) {
	dir := t.TempDir()
	h := ghpages.New(dir, "https://owner.github.io/repo")

	feedXML := []byte(`<?xml version="1.0"?><rss></rss>`)
	url, err := h.PutFeed(context.Background(), feedXML)
	if err != nil {
		t.Fatalf("PutFeed: %v", err)
	}

	want := "https://owner.github.io/repo/feed.xml"
	if url != want {
		t.Errorf("url = %q, want %q", url, want)
	}

	got, err := os.ReadFile(filepath.Join(dir, "feed.xml"))
	if err != nil {
		t.Fatalf("read file: %v", err)
	}
	if !bytes.Equal(got, feedXML) {
		t.Errorf("feed.xml content mismatch")
	}
}

func TestHosting_LoadEpisodes_NotFound(t *testing.T) {
	dir := t.TempDir()
	h := ghpages.New(dir, "https://owner.github.io/repo")

	eps, err := h.LoadEpisodes(context.Background())
	if err != nil {
		t.Fatalf("LoadEpisodes (not found): %v", err)
	}
	if len(eps.Episodes) != 0 {
		t.Errorf("expected empty episodes, got %d", len(eps.Episodes))
	}
}

func TestHosting_LoadEpisodes_Existing(t *testing.T) {
	dir := t.TempDir()
	h := ghpages.New(dir, "https://owner.github.io/repo")

	data := `{"episodes":[{"guid":"ep1","title":"T","description":"D","pub_date":"2026-05-30T21:00:00Z","audio_url":"https://owner.github.io/repo/a.mp3","bytes":100,"duration":"00:01:00"}]}`
	if err := os.WriteFile(filepath.Join(dir, "episodes.json"), []byte(data), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}

	eps, err := h.LoadEpisodes(context.Background())
	if err != nil {
		t.Fatalf("LoadEpisodes: %v", err)
	}
	if len(eps.Episodes) != 1 {
		t.Fatalf("expected 1 episode, got %d", len(eps.Episodes))
	}
	if eps.Episodes[0].GUID != "ep1" {
		t.Errorf("guid = %q, want ep1", eps.Episodes[0].GUID)
	}
}

func TestHosting_SaveEpisodes(t *testing.T) {
	dir := t.TempDir()
	h := ghpages.New(dir, "https://owner.github.io/repo")

	eps := model.Episodes{
		Episodes: []model.Episode{
			{
				GUID:        "ep1",
				Title:       "T",
				Description: "D",
				PubDate:     "2026-05-30T21:00:00Z",
				AudioURL:    "https://owner.github.io/repo/a.mp3",
				Bytes:       100,
				Duration:    "00:01:00",
			},
		},
	}
	if err := h.SaveEpisodes(context.Background(), eps); err != nil {
		t.Fatalf("SaveEpisodes: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(dir, "episodes.json"))
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if !strings.Contains(string(data), "ep1") {
		t.Errorf("episodes.json does not contain ep1")
	}
}

func TestHosting_SaveLoadRoundTrip(t *testing.T) {
	dir := t.TempDir()
	h := ghpages.New(dir, "https://owner.github.io/repo")

	original := model.Episodes{
		Episodes: []model.Episode{
			{
				GUID:        "ep1",
				Title:       "T",
				Description: "D",
				PubDate:     "2026-05-30T21:00:00Z",
				AudioURL:    "https://owner.github.io/repo/a.mp3",
				Bytes:       100,
				Duration:    "00:01:00",
			},
		},
	}
	if err := h.SaveEpisodes(context.Background(), original); err != nil {
		t.Fatalf("SaveEpisodes: %v", err)
	}

	loaded, err := h.LoadEpisodes(context.Background())
	if err != nil {
		t.Fatalf("LoadEpisodes: %v", err)
	}
	if len(loaded.Episodes) != 1 {
		t.Fatalf("expected 1 episode, got %d", len(loaded.Episodes))
	}
	if loaded.Episodes[0].GUID != "ep1" {
		t.Errorf("guid = %q, want ep1", loaded.Episodes[0].GUID)
	}
}

func TestHosting_DeleteAudio(t *testing.T) {
	dir := t.TempDir()
	h := ghpages.New(dir, "https://owner.github.io/repo")

	audioDir := filepath.Join(dir, "audio")
	if err := os.MkdirAll(audioDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(audioDir, "ep.mp3"), []byte("data"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}

	if err := h.DeleteAudio(context.Background(), "ep.mp3"); err != nil {
		t.Fatalf("DeleteAudio: %v", err)
	}
	if _, err := os.Stat(filepath.Join(audioDir, "ep.mp3")); !os.IsNotExist(err) {
		t.Error("file should be deleted")
	}
}

func TestHosting_DeleteAudio_NonExistent(t *testing.T) {
	dir := t.TempDir()
	h := ghpages.New(dir, "https://owner.github.io/repo")

	if err := h.DeleteAudio(context.Background(), "nonexistent.mp3"); err != nil {
		t.Errorf("DeleteAudio (non-existent) should be no-op: %v", err)
	}
}

func checkGit(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not in PATH")
	}
}

func gitRun(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
}

func setupBareRemote(t *testing.T) string {
	t.Helper()
	remoteDir := t.TempDir()
	gitRun(t, remoteDir, "init", "--bare")
	return remoteDir
}

func setupWorkDir(t *testing.T, remoteDir string) string {
	t.Helper()
	workDir := t.TempDir()
	gitRun(t, workDir, "init")
	gitRun(t, workDir, "remote", "add", "origin", remoteDir)
	return workDir
}

func TestHosting_Push(t *testing.T) {
	checkGit(t)

	remoteDir := setupBareRemote(t)
	workDir := setupWorkDir(t, remoteDir)

	h := ghpages.New(workDir, "https://owner.github.io/repo")

	ctx := context.Background()
	_, err := h.PutAudio(ctx, "ep.mp3", bytes.NewReader([]byte("audio data")))
	if err != nil {
		t.Fatalf("PutAudio: %v", err)
	}
	if err := h.SaveEpisodes(ctx, model.Episodes{
		Episodes: []model.Episode{{GUID: "ep1", Title: "T", Description: "D", PubDate: "2026-05-30T21:00:00Z", AudioURL: "https://owner.github.io/repo/audio/ep.mp3", Bytes: 10, Duration: "00:00:01"}},
	}); err != nil {
		t.Fatalf("SaveEpisodes: %v", err)
	}
	if _, err := h.PutFeed(ctx, []byte(`<?xml version="1.0"?><rss></rss>`)); err != nil {
		t.Fatalf("PutFeed: %v", err)
	}

	if err := h.Push(ctx); err != nil {
		t.Fatalf("Push: %v", err)
	}

	// Verify files are published to remote by cloning gh-pages branch
	cloneDir := t.TempDir()
	gitRun(t, cloneDir, "clone", "--branch", "gh-pages", remoteDir, ".")

	for _, f := range []string{
		filepath.Join("audio", "ep.mp3"),
		"episodes.json",
		"feed.xml",
	} {
		if _, err := os.Stat(filepath.Join(cloneDir, f)); err != nil {
			t.Errorf("expected file %q in remote, got: %v", f, err)
		}
	}
}

func TestHosting_Push_Orphan(t *testing.T) {
	checkGit(t)

	remoteDir := setupBareRemote(t)
	workDir := setupWorkDir(t, remoteDir)

	h := ghpages.New(workDir, "https://owner.github.io/repo")

	ctx := context.Background()
	_, err := h.PutFeed(ctx, []byte(`<?xml version="1.0"?><rss></rss>`))
	if err != nil {
		t.Fatalf("PutFeed: %v", err)
	}

	// Push once
	if err := h.Push(ctx); err != nil {
		t.Fatalf("first Push: %v", err)
	}

	// Clone gh-pages and count commits — orphan should have exactly 1 commit
	cloneDir := t.TempDir()
	gitRun(t, cloneDir, "clone", "--branch", "gh-pages", remoteDir, ".")

	cmd := exec.Command("git", "log", "--oneline")
	cmd.Dir = cloneDir
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("git log: %v", err)
	}
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	if len(lines) != 1 {
		t.Errorf("expected 1 commit (orphan), got %d: %s", len(lines), out)
	}
}
