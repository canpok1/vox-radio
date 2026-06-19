package cli

import (
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"regexp"
	"time"

	"github.com/canpok1/vox-radio/internal/cache"
	"github.com/canpok1/vox-radio/internal/cast"
	"github.com/canpok1/vox-radio/internal/config"
	"github.com/canpok1/vox-radio/internal/fileio"
	"github.com/canpok1/vox-radio/internal/logging"
	"github.com/canpok1/vox-radio/internal/model"
	"github.com/canpok1/vox-radio/internal/script/llm"
	"github.com/spf13/cobra"
)

// DefaultConfigPath is the default path for the shared config file (vox-radio.yaml).
const DefaultConfigPath = "vox-radio.yaml"

// configPath returns the value of the --config persistent flag, falling back to DefaultConfigPath.
func configPath(cmd *cobra.Command) string {
	p, _ := cmd.Flags().GetString("config")
	if p == "" {
		return DefaultConfigPath
	}
	return p
}

// logDirFlag returns the value of the --log-dir persistent flag, falling back to defaultLogDir.
func logDirFlag(cmd *cobra.Command) string {
	d, _ := cmd.Flags().GetString("log-dir")
	if d == "" {
		return defaultLogDir
	}
	return d
}

// registerSpecFlag registers the required --spec flag on cmd, binding it
// to specPath. Used by episodegen subcommands that load an episode spec (mix is the
// exception: its --spec is optional because assets can be skipped).
// Note: feedgen uses --spec too but registers it inline with a different description.
func registerSpecFlag(cmd *cobra.Command, specPath *string) {
	cmd.Flags().StringVar(specPath, "spec", "", "エピソード仕様 YAML ファイルのパス（必須）")
	_ = cmd.MarkFlagRequired("spec")
}

// writeFile writes content to path. If the file already exists and force is
// false, it prints a skip message and returns nil. Otherwise it creates
// parent directories as needed and overwrites the file.
func writeFile(cmd *cobra.Command, path string, content []byte, force bool) error {
	if _, err := os.Stat(path); err == nil && !force {
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "skip: %s already exists\n", path)
		return nil
	}
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("mkdir %s: %w", dir, err)
	}
	if err := os.WriteFile(path, content, 0644); err != nil {
		return fmt.Errorf("write %s: %w", path, err)
	}
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "created: %s\n", path)
	return nil
}

// writeEmbeddedTree copies every file under srcRoot in fsys to dstRoot, preserving
// the relative directory structure. Directories are skipped (writeFile creates parents
// as needed). Each file is written via writeFile, so existing files are skipped
// individually when force is false. Used by both `init` and `install --skills`.
//
// When skip is non-nil, files whose slash-separated relative path returns true are not
// written. This lets callers overlay an alternate file (e.g. a different episode-spec)
// or omit part of a shared template tree.
func writeEmbeddedTree(cmd *cobra.Command, fsys fs.FS, srcRoot, dstRoot string, force bool, skip func(rel string) bool) error {
	return fs.WalkDir(fsys, srcRoot, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(srcRoot, path)
		if err != nil {
			return fmt.Errorf("rel path: %w", err)
		}
		if skip != nil && skip(filepath.ToSlash(rel)) {
			return nil
		}
		content, err := fs.ReadFile(fsys, path)
		if err != nil {
			return fmt.Errorf("read %s: %w", path, err)
		}
		return writeFile(cmd, filepath.Join(dstRoot, rel), content, force)
	})
}

func writeJSON(path string, v any) error {
	return fileio.WriteJSON(path, v)
}

func readJSON[T any](path string) (T, error) {
	var v T
	if err := fileio.ReadJSON(path, &v); err != nil {
		return v, err
	}
	return v, nil
}

const defaultLogDir = ".vox-radio/logs"

var semverRe = regexp.MustCompile(`^\d+\.\d+\.\d+$`)

// referenceURL returns a versioned GitHub URL for relPath under the vox-radio repository.
// When version is a semver (e.g. "1.2.3"), the URL points to that tagged commit.
// Otherwise (e.g. "dev", snapshot builds), it falls back to the main branch.
func referenceURL(relPath string) string {
	ref := "main"
	if semverRe.MatchString(version) {
		ref = "v" + version
	}
	return fmt.Sprintf("https://github.com/canpok1/vox-radio/blob/%s/%s", ref, relPath)
}

// setupLogger creates a fan-out logger (stderr INFO+, logFile DEBUG+) and returns it with the
// log file handle. The caller must close the file when done.
// logDir defaults to ".vox-radio/logs" if empty.
func setupLogger(commandName string, logDir string) (*slog.Logger, *os.File, error) {
	if logDir == "" {
		logDir = defaultLogDir
	}
	return logging.NewSetup(time.Now(), commandName, logDir)
}

func newLLMClient(cfg *config.Config) llm.Client {
	llmCfg := llm.Config{
		Provider:             cfg.LLM.EffectiveProvider(),
		Temperature:          cfg.LLM.Temperature,
		MaxRetries:           cfg.LLM.MaxRetries,
		MinRequestIntervalMS: cfg.LLM.EffectiveMinRequestIntervalMS(),
	}

	switch llmCfg.Provider {
	case config.ProviderDifyChat:
		if dc := cfg.LLM.DifyChat; dc != nil {
			user := dc.User
			if user == "" {
				user = config.DefaultDifyUser
			}
			llmCfg.DifyChat = &llm.DifyChatClientConfig{
				BaseURL: dc.BaseURL,
				APIKey:  os.Getenv(dc.APIKeyEnv),
				User:    user,
				Inputs:  dc.Inputs,
			}
		}
	default: // openai
		if oa := cfg.LLM.OpenAI; oa != nil {
			llmCfg.BaseURL = oa.BaseURL
			llmCfg.APIKey = os.Getenv(oa.APIKeyEnv)
			llmCfg.Model = oa.Model
		}
	}

	return llm.NewClient(llmCfg)
}

// selectCasts selects cast members for the given episode number and injects appearance stats.
// AppearanceCount includes the current episode (past count + 1); LastEpisodeNumber is the most
// recent past appearance (0 if none).
func selectCasts(casts map[string]config.CastConfig, episodeNumber int, appearances map[string]cache.CastAppearance) []model.RundownCast {
	selected := cast.Select(casts, episodeNumber)
	for i, c := range selected {
		a := appearances[c.CharacterID]
		selected[i].AppearanceCount = a.Count + 1
		selected[i].LastEpisodeNumber = a.LastEpisodeNumber
	}
	return selected
}

// resolveLocation resolves program timezone to *time.Location.
// On invalid timezone, logs a WARN and falls back to time.UTC.
func resolveLocation(program config.ProgramConfig, logger *slog.Logger) *time.Location {
	loc, err := program.Location()
	if err != nil {
		logger.Warn("番組タイムゾーンが不正なため UTC にフォールバックします", "timezone", program.EffectiveTimezone(), "err", err)
		return time.UTC
	}
	return loc
}

// resolveCornersByRundown は rundown のコーナー id 順に spec のコーナーを再構成する。
// script 系で採用コーナーを再現するために使う（回番号不要・再実行で不変・タイトル変更に頑健）。
func resolveCornersByRundown(corners []config.CornerConfig, rd model.Rundown) ([]config.CornerConfig, error) {
	ids := make([]string, len(rd.Corners))
	for i, c := range rd.Corners {
		ids[i] = c.ID
	}
	return config.ResolveCornersByIDs(corners, ids)
}

// resolveCorners は回番号で採用コーナーを絞り込む。
func resolveCorners(corners []config.CornerConfig, episodeNumber int) []config.CornerConfig {
	return config.ResolveCornersForEpisode(corners, episodeNumber)
}

// requireEnv returns the value of the environment variable named by name.
// If the variable is unset or empty and dryRun is false, it returns an error.
// In dry-run mode an empty value is allowed (the caller won't use it).
func requireEnv(name string, dryRun bool) (string, error) {
	v := os.Getenv(name)
	if v == "" && !dryRun {
		return "", fmt.Errorf("env var %q is not set", name)
	}
	return v, nil
}

func programCachePath(programID string) string {
	return filepath.Join(".vox-radio", "cache", programID+".jsonl")
}

// loadCacheEntries loads all cache entries for the given program.
// Returns (entries, nextEpisodeNumber, error). File-not-found is not an error (returns episode 1).
// program.id is required (validated at load time), so the cache is always consulted.
func loadCacheEntries(programID string) ([]cache.Entry, int, error) {
	mgr := cache.New(programCachePath(programID))
	entries, err := mgr.Load()
	if err != nil {
		return nil, 0, fmt.Errorf("load cache: %w", err)
	}
	return entries, cache.NextEpisodeNumber(entries), nil
}

func loadConfigAndSpec(cfgPath, specPath string) (*config.Config, *config.EpisodeSpec, error) {
	cfg, err := config.LoadConfig(cfgPath)
	if err != nil {
		return nil, nil, fmt.Errorf("load config: %w", err)
	}
	p, err := config.LoadEpisodeSpec(specPath)
	if err != nil {
		return nil, nil, fmt.Errorf("load spec: %w", err)
	}
	if err := p.Validate(cfg.Characters); err != nil {
		return nil, nil, fmt.Errorf("spec validation: %w", err)
	}
	return cfg, p, nil
}
