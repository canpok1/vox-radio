package cli

import (
	"encoding/json"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

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

func TestSetupLogger_DefaultLogDir(t *testing.T) {
	tmpDir := chdirTemp(t)

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

	got := resolveCorners(corners, 2)
	if len(got) != 2 {
		t.Errorf("resolveCorners(_, 2) len = %d, want 2", len(got))
	}

	got = resolveCorners(corners, 3)
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

	got := resolveCorners(corners, 0)
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

func TestLoadCacheEntries_NoCacheFile(t *testing.T) {
	chdirTemp(t)

	entries, n, err := loadCacheEntries("test_program")
	if err != nil {
		t.Fatalf("expected no error when no cache file, got: %v", err)
	}
	if len(entries) != 0 {
		t.Errorf("expected empty entries when no cache file, got %d", len(entries))
	}
	if n != 1 {
		t.Errorf("expected 1 (first episode) when no cache file, got %d", n)
	}
}

func TestLoadCacheEntries_ReturnsEntriesAndEpisodeNumber(t *testing.T) {
	tmpDir := chdirTemp(t)

	writeCacheJSONL(t, tmpDir, "prog", []cache.Entry{
		{EpisodeNumber: 5, Casts: []cache.CastEntry{
			{CharacterID: "zundamon", Type: "regular"},
		}},
	})

	entries, n, err := loadCacheEntries("prog")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
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

func TestLoadCacheEntries_CorruptedCache(t *testing.T) {
	tmpDir := chdirTemp(t)

	cacheDir := filepath.Join(tmpDir, ".vox-radio", "cache")
	if err := os.MkdirAll(cacheDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(cacheDir, "corrupt.jsonl"), []byte("not valid json\n"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}

	_, _, err := loadCacheEntries("corrupt")
	if err == nil {
		t.Error("expected error for corrupted cache, got nil")
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

	selected := selectCasts(casts, 1, counts)

	countByID := make(map[string]int)
	for _, c := range selected {
		countByID[c.CharacterID] = c.AppearanceCount
	}
	if countByID["zundamon"] != 11 {
		t.Errorf("zundamon AppearanceCount: got %d, want 11 (counts[id]+1)", countByID["zundamon"])
	}
	if countByID["guest1"] != 3 {
		t.Errorf("guest1 AppearanceCount: got %d, want 3 (counts[id]+1)", countByID["guest1"])
	}
}

func TestSelectCasts_UnknownCharHasZeroCount(t *testing.T) {
	casts := map[string]config.CastConfig{
		"zundamon": {Role: "MC", Type: config.CastTypeRegular},
	}
	counts := map[string]int{} // no entry for zundamon

	selected := selectCasts(casts, 1, counts)
	if len(selected) == 0 {
		t.Fatal("expected at least one cast member")
	}
	for _, c := range selected {
		if c.CharacterID == "zundamon" && c.AppearanceCount != 1 {
			t.Errorf("zundamon AppearanceCount: got %d, want 1 (not in counts: 0+1)", c.AppearanceCount)
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

func TestLogDirFlag_Default(t *testing.T) {
	root := NewRootCmd()
	got := logDirFlag(root)
	if got != defaultLogDir {
		t.Errorf("logDirFlag(root) = %q, want %q", got, defaultLogDir)
	}
}

func TestLogDirFlag_CustomValue(t *testing.T) {
	root := NewRootCmd()
	if err := root.ParseFlags([]string{"--log-dir", "/custom/logs"}); err != nil {
		t.Fatalf("ParseFlags: %v", err)
	}
	got := logDirFlag(root)
	want := "/custom/logs"
	if got != want {
		t.Errorf("logDirFlag(root) = %q, want %q", got, want)
	}
}

func TestResolveLocation_ValidTimezone(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	program := config.ProgramConfig{Timezone: "Asia/Tokyo"}

	loc := resolveLocation(program, logger)

	want, err := time.LoadLocation("Asia/Tokyo")
	if err != nil {
		t.Fatalf("time.LoadLocation: %v", err)
	}
	if loc.String() != want.String() {
		t.Errorf("resolveLocation = %q, want %q", loc.String(), want.String())
	}
}

func TestResolveLocation_InvalidTimezone_FallsBackToUTC(t *testing.T) {
	var buf strings.Builder
	logger := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelWarn}))
	program := config.ProgramConfig{Timezone: "Invalid/Zone"}

	loc := resolveLocation(program, logger)

	if loc != time.UTC {
		t.Errorf("resolveLocation with invalid timezone = %q, want UTC", loc.String())
	}
	if !strings.Contains(buf.String(), "WARN") {
		t.Errorf("expected WARN log for invalid timezone, got: %q", buf.String())
	}
}
