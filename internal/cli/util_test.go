package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/canpok1/vox-radio/internal/cache"
	"github.com/canpok1/vox-radio/internal/config"
)

func writeCacheJSONL(t *testing.T, dir string, programID string, entries []cache.Entry) {
	t.Helper()
	cacheDir := filepath.Join(dir, ".vox-radio", "cache")
	if err := os.MkdirAll(cacheDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	path := filepath.Join(cacheDir, programID+".jsonl")
	f, err := os.Create(path)
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	defer f.Close()
	enc := json.NewEncoder(f)
	for _, e := range entries {
		if err := enc.Encode(e); err != nil {
			t.Fatalf("encode: %v", err)
		}
	}
}

func TestResolveEpisodeNumber_CacheDisabled(t *testing.T) {
	cfg := &config.Config{Cache: config.CacheConfig{Enabled: false}}
	n := resolveEpisodeNumber(cfg, "test_program")
	if n != 0 {
		t.Errorf("expected 0 when cache disabled, got %d", n)
	}
}

func TestResolveEpisodeNumber_EmptyProgramID(t *testing.T) {
	cfg := &config.Config{Cache: config.CacheConfig{Enabled: true}}
	n := resolveEpisodeNumber(cfg, "")
	if n != 0 {
		t.Errorf("expected 0 when programID empty, got %d", n)
	}
}

func TestResolveEpisodeNumber_FirstEpisode(t *testing.T) {
	// os.Chdir はプロセス全体の cwd を変更するため並列禁止
	origDir, _ := os.Getwd()
	tmpDir := t.TempDir()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	defer func() { _ = os.Chdir(origDir) }()

	cfg := &config.Config{Cache: config.CacheConfig{Enabled: true}}
	// キャッシュファイルが存在しない場合は第1回
	n := resolveEpisodeNumber(cfg, "my_program")
	if n != 1 {
		t.Errorf("expected 1 for first episode, got %d", n)
	}
}

func TestResolveEpisodeNumber_WithExistingCache(t *testing.T) {
	origDir, _ := os.Getwd()
	tmpDir := t.TempDir()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	defer func() { _ = os.Chdir(origDir) }()

	writeCacheJSONL(t, tmpDir, "prog", []cache.Entry{
		{EpisodeNumber: 3},
	})

	cfg := &config.Config{Cache: config.CacheConfig{Enabled: true}}
	n := resolveEpisodeNumber(cfg, "prog")
	if n != 4 {
		t.Errorf("expected 4 (next after 3), got %d", n)
	}
}

func TestSetupLogger_DefaultLogDir(t *testing.T) {
	// os.Chdir changes the process-wide cwd, so no t.Parallel().
	origDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	tmpDir := t.TempDir()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("chdir to tmpDir: %v", err)
	}
	defer func() {
		_ = os.Chdir(origDir)
	}()

	logger, f, err := setupLogger("collect", "")
	if err != nil {
		t.Fatalf("setupLogger: %v", err)
	}
	defer f.Close()

	if logger == nil {
		t.Fatal("expected non-nil logger")
	}

	logPath, err := filepath.Abs(f.Name())
	if err != nil {
		t.Fatalf("abs: %v", err)
	}
	wantPrefix := filepath.Join(tmpDir, ".vox-radio", "logs")
	if !strings.HasPrefix(logPath, wantPrefix) {
		t.Errorf("log file path %q does not start with %q", logPath, wantPrefix)
	}
}
