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

func writeCacheRaw(t *testing.T, dir string, programID string, content []byte) {
	t.Helper()
	cacheDir := filepath.Join(dir, ".vox-radio", "cache")
	if err := os.MkdirAll(cacheDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(cacheDir, programID+".jsonl"), content, 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
}

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

	logger, f, err := setupLogger("gather", "")
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
		{ID: "a", Title: "A", LengthSec: 30},
		{ID: "b", Title: "B", LengthSec: 60},
		{ID: "c", Title: "C", LengthSec: 90},
	}
	rd := model.Rundown{
		Corners: []model.RundownCorner{
			{ID: "c", Title: "C"},
			{ID: "a", Title: "A"},
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

func TestResolveCornersByRundown_UnknownID(t *testing.T) {
	corners := []config.CornerConfig{
		{ID: "a", Title: "A", LengthSec: 30},
	}
	rd := model.Rundown{
		Corners: []model.RundownCorner{
			{ID: "x", Title: "X"},
		},
	}

	_, err := resolveCornersByRundown(corners, rd)
	if err == nil {
		t.Error("expected error for unknown id, got nil")
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
	writeCacheRaw(t, tmpDir, "corrupt", []byte("not valid json\n"))

	_, _, err := loadCacheEntries("corrupt")
	if err == nil {
		t.Error("expected error for corrupted cache, got nil")
	}
}

func TestSelectCasts_InjectsAppearanceCountAndLastEpisodeNumber(t *testing.T) {
	// guest は condition が必要（cast.Select の仕様）
	guestCond := &config.EpisodeCondition{Episodes: []int{1}}
	casts := map[string]config.CastConfig{
		"zundamon": {Role: "MC", Type: config.CastTypeRegular},
		"guest1":   {Role: "ゲスト", Type: config.CastTypeGuest, Condition: guestCond},
	}
	appearances := map[string]cache.CastAppearance{
		"zundamon": {Count: 10, LastEpisodeNumber: 12},
		"guest1":   {Count: 2, LastEpisodeNumber: 7},
	}

	selected := selectCasts(casts, 1, appearances)

	byID := make(map[string]model.RundownCast)
	for _, c := range selected {
		byID[c.CharacterID] = c
	}
	if byID["zundamon"].AppearanceCount != 11 {
		t.Errorf("zundamon AppearanceCount: got %d, want 11 (Count+1)", byID["zundamon"].AppearanceCount)
	}
	if byID["zundamon"].LastEpisodeNumber != 12 {
		t.Errorf("zundamon LastEpisodeNumber: got %d, want 12", byID["zundamon"].LastEpisodeNumber)
	}
	if byID["guest1"].AppearanceCount != 3 {
		t.Errorf("guest1 AppearanceCount: got %d, want 3 (Count+1)", byID["guest1"].AppearanceCount)
	}
	if byID["guest1"].LastEpisodeNumber != 7 {
		t.Errorf("guest1 LastEpisodeNumber: got %d, want 7", byID["guest1"].LastEpisodeNumber)
	}
}

func TestSelectCasts_UnknownCharHasZeroAppearance(t *testing.T) {
	casts := map[string]config.CastConfig{
		"zundamon": {Role: "MC", Type: config.CastTypeRegular},
	}
	appearances := map[string]cache.CastAppearance{} // no entry for zundamon

	selected := selectCasts(casts, 1, appearances)
	if len(selected) == 0 {
		t.Fatal("expected at least one cast member")
	}
	for _, c := range selected {
		if c.CharacterID == "zundamon" {
			if c.AppearanceCount != 1 {
				t.Errorf("zundamon AppearanceCount: got %d, want 1 (not in appearances: 0+1)", c.AppearanceCount)
			}
			if c.LastEpisodeNumber != 0 {
				t.Errorf("zundamon LastEpisodeNumber: got %d, want 0", c.LastEpisodeNumber)
			}
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

func TestRequireEnv(t *testing.T) {
	tests := []struct {
		name    string
		envName string
		envVal  string
		dryRun  bool
		wantVal string
		wantErr bool
	}{
		{
			name:    "env set not dry-run",
			envName: "TEST_REQUIRE_ENV_SET",
			envVal:  "myvalue",
			dryRun:  false,
			wantVal: "myvalue",
			wantErr: false,
		},
		{
			name:    "env set dry-run",
			envName: "TEST_REQUIRE_ENV_SET",
			envVal:  "myvalue",
			dryRun:  true,
			wantVal: "myvalue",
			wantErr: false,
		},
		{
			name:    "env not set not dry-run",
			envName: "TEST_REQUIRE_ENV_NOTSET",
			dryRun:  false,
			wantVal: "",
			wantErr: true,
		},
		{
			name:    "env not set dry-run",
			envName: "TEST_REQUIRE_ENV_NOTSET",
			dryRun:  true,
			wantVal: "",
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.envVal != "" {
				t.Setenv(tt.envName, tt.envVal)
			}
			got, err := requireEnv(tt.envName, tt.dryRun)
			if (err != nil) != tt.wantErr {
				t.Errorf("requireEnv() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.wantVal {
				t.Errorf("requireEnv() = %q, want %q", got, tt.wantVal)
			}
		})
	}
}

func TestReferenceURL(t *testing.T) {
	tests := []struct {
		name        string
		version     string
		relPath     string
		wantContain string
	}{
		{
			name:        "semver uses version tag",
			version:     "1.2.3",
			relPath:     "internal/cli/skills/vox-radio/references/manifest.md",
			wantContain: "/blob/v1.2.3/internal/cli/skills/vox-radio/references/manifest.md",
		},
		{
			name:        "dev falls back to main",
			version:     "dev",
			relPath:     "internal/cli/skills/vox-radio/references/manifest.md",
			wantContain: "/blob/main/internal/cli/skills/vox-radio/references/manifest.md",
		},
		{
			name:        "invalid version falls back to main",
			version:     "snapshot-abc",
			relPath:     "internal/cli/skills/vox-radio/references/manifest.md",
			wantContain: "/blob/main/internal/cli/skills/vox-radio/references/manifest.md",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			orig := version
			version = tt.version
			t.Cleanup(func() { version = orig })

			got := referenceURL(tt.relPath)
			if !strings.Contains(got, tt.wantContain) {
				t.Errorf("referenceURL(%q) = %q, want containing %q", tt.relPath, got, tt.wantContain)
			}
		})
	}
}
