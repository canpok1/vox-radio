package cli

import (
	"encoding/json"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/canpok1/vox-radio/internal/cache"
	"github.com/canpok1/vox-radio/internal/config"
	"github.com/canpok1/vox-radio/internal/model"
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

func TestResolveCorners_WithEpisodeNumber(t *testing.T) {
	cond := &config.EpisodeCondition{Episodes: []int{2, 4}}
	corners := []config.CornerConfig{
		{Title: "固定", LengthSec: 30},
		{Title: "条件付き", LengthSec: 60, Condition: cond},
	}
	logger := slog.Default()

	got := resolveCorners(corners, 2, logger)
	if len(got) != 2 {
		t.Errorf("resolveCorners(_, 2) len = %d, want 2", len(got))
	}

	got = resolveCorners(corners, 3, logger)
	if len(got) != 1 || got[0].Title != "固定" {
		t.Errorf("resolveCorners(_, 3) = %v, want [固定]", got)
	}
}

func TestResolveCorners_UnknownEpisodeReturnsAll(t *testing.T) {
	cond := &config.EpisodeCondition{Every: 2}
	corners := []config.CornerConfig{
		{Title: "固定", LengthSec: 30},
		{Title: "条件付き", LengthSec: 60, Condition: cond},
	}
	logger := slog.Default()

	got := resolveCorners(corners, 0, logger)
	if len(got) != 2 {
		t.Errorf("resolveCorners(_, 0) len = %d, want 2 (all corners)", len(got))
	}
}

func TestResolveCornersByRundown(t *testing.T) {
	corners := []config.CornerConfig{
		{Title: "A", LengthSec: 30},
		{Title: "B", LengthSec: 60},
		{Title: "C", LengthSec: 90},
	}
	rd := model.Rundown{
		Corners: []model.RundownCorner{
			{Title: "C"},
			{Title: "A"},
		},
	}

	got, err := resolveCornersByRundown(corners, rd)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("got %d corners, want 2", len(got))
	}
	if got[0].Title != "C" || got[1].Title != "A" {
		t.Errorf("titles = [%s %s], want [C A]", got[0].Title, got[1].Title)
	}
}

func TestResolveCornersByRundown_UnknownTitle(t *testing.T) {
	corners := []config.CornerConfig{
		{Title: "A", LengthSec: 30},
	}
	rd := model.Rundown{
		Corners: []model.RundownCorner{
			{Title: "X"},
		},
	}

	_, err := resolveCornersByRundown(corners, rd)
	if err == nil {
		t.Error("expected error for unknown title, got nil")
	}
}

func TestLoadCacheEntries_CacheDisabled(t *testing.T) {
	cfg := &config.Config{Cache: config.CacheConfig{Enabled: false}}
	entries, n := loadCacheEntries(cfg, "test_program")
	if len(entries) != 0 {
		t.Errorf("expected empty entries when cache disabled, got %d", len(entries))
	}
	if n != 0 {
		t.Errorf("expected 0 when cache disabled, got %d", n)
	}
}

func TestLoadCacheEntries_EmptyProgramID(t *testing.T) {
	cfg := &config.Config{Cache: config.CacheConfig{Enabled: true}}
	entries, n := loadCacheEntries(cfg, "")
	if len(entries) != 0 {
		t.Errorf("expected empty entries when programID empty, got %d", len(entries))
	}
	if n != 0 {
		t.Errorf("expected 0 when programID empty, got %d", n)
	}
}

func TestLoadCacheEntries_ReturnsEntriesAndEpisodeNumber(t *testing.T) {
	origDir, _ := os.Getwd()
	tmpDir := t.TempDir()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	defer func() { _ = os.Chdir(origDir) }()

	writeCacheJSONL(t, tmpDir, "prog", []cache.Entry{
		{EpisodeNumber: 5, Casts: []cache.CastEntry{
			{CharacterID: "zundamon", Type: "regular"},
		}},
	})

	cfg := &config.Config{Cache: config.CacheConfig{Enabled: true}}
	entries, n := loadCacheEntries(cfg, "prog")
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}
	if n != 6 {
		t.Errorf("expected 6 (next after 5), got %d", n)
	}
	if len(entries[0].Casts) != 1 {
		t.Errorf("expected 1 cast, got %d", len(entries[0].Casts))
	}
}

func TestSelectCasts_InjectsAppearanceCount(t *testing.T) {
	// guest は condition が必要（cast.Select の仕様）
	guestCond := &config.EpisodeCondition{Episodes: []int{1}}
	casts := map[string]config.CastConfig{
		"zundamon": {Role: "MC", Type: config.CastTypeRegular},
		"guest1":   {Role: "ゲスト", Type: config.CastTypeGuest, Condition: guestCond},
	}
	counts := map[string]int{
		"zundamon": 10,
		"guest1":   2,
	}
	logger := slog.Default()

	selected := selectCasts(casts, 1, counts, logger)

	countByID := make(map[string]int)
	for _, c := range selected {
		countByID[c.CharacterID] = c.AppearanceCount
	}
	if countByID["zundamon"] != 10 {
		t.Errorf("zundamon AppearanceCount: got %d, want 10", countByID["zundamon"])
	}
	if countByID["guest1"] != 2 {
		t.Errorf("guest1 AppearanceCount: got %d, want 2", countByID["guest1"])
	}
}

func TestSelectCasts_UnknownCharHasZeroCount(t *testing.T) {
	casts := map[string]config.CastConfig{
		"zundamon": {Role: "MC", Type: config.CastTypeRegular},
	}
	counts := map[string]int{} // no entry for zundamon
	logger := slog.Default()

	selected := selectCasts(casts, 1, counts, logger)
	if len(selected) == 0 {
		t.Fatal("expected at least one cast member")
	}
	for _, c := range selected {
		if c.CharacterID == "zundamon" && c.AppearanceCount != 0 {
			t.Errorf("zundamon AppearanceCount: got %d, want 0 (not in counts)", c.AppearanceCount)
		}
	}
}

func TestConfigPath_Default(t *testing.T) {
	root := NewRootCmd()
	got := configPath(root)
	if got != DefaultConfigPath {
		t.Errorf("configPath(root) = %q, want %q", got, DefaultConfigPath)
	}
}

func TestConfigPath_CustomValue(t *testing.T) {
	root := NewRootCmd()
	if err := root.ParseFlags([]string{"--config", "/custom/vox-radio.yaml"}); err != nil {
		t.Fatalf("ParseFlags: %v", err)
	}
	got := configPath(root)
	want := "/custom/vox-radio.yaml"
	if got != want {
		t.Errorf("configPath(root) = %q, want %q", got, want)
	}
}
