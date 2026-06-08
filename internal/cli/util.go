package cli

import (
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
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
// to specPath. Used by episodegen subcommands that load an episode spec (assemble is the
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
func writeEmbeddedTree(cmd *cobra.Command, fsys fs.FS, srcRoot, dstRoot string, force bool) error {
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

// selectCasts selects cast members for the given episode number and injects appearance counts.
func selectCasts(casts map[string]config.CastConfig, episodeNumber int, counts map[string]int) []model.RundownCast {
	selected := cast.Select(casts, episodeNumber)
	for i, c := range selected {
		selected[i].AppearanceCount = counts[c.CharacterID] + 1
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
	if err := config.ValidateEpisodeSpecProgram(p); err != nil {
		return nil, nil, fmt.Errorf("spec validation: %w", err)
	}
	if err := config.ValidateEpisodeSpecCast(p); err != nil {
		return nil, nil, fmt.Errorf("spec validation: %w", err)
	}
	if err := config.ValidateEpisodeSpecAssets(p); err != nil {
		return nil, nil, fmt.Errorf("spec validation: %w", err)
	}
	if err := config.ValidateEpisodeSpecCasts(p, cfg.Characters); err != nil {
		return nil, nil, fmt.Errorf("spec validation: %w", err)
	}
	if err := config.ValidateEpisodeSpecCorners(p); err != nil {
		return nil, nil, fmt.Errorf("spec validation: %w", err)
	}
	return cfg, p, nil
}
