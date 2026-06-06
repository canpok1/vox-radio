package cli

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"slices"
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

// selectCasts selects cast members for the given episode number, injects appearance counts,
// and warns when condition-based cast members are configured but the episode number is unknown.
func selectCasts(casts map[string]config.CastConfig, episodeNumber int, counts map[string]int, logger *slog.Logger) []model.RundownCast {
	selected := cast.Select(casts, episodeNumber)
	for i, c := range selected {
		selected[i].AppearanceCount = counts[c.CharacterID] + 1
	}
	if episodeNumber == 0 {
		for _, c := range casts {
			if c.Condition != nil || c.Type == config.CastTypeGuest {
				logger.Warn("条件付きキャストが設定されていますが回番号が不明なため、一部のキャストは出演しません")
				break
			}
		}
	}
	return selected
}

// resolveCornersByRundown は rundown のコーナータイトル順に spec のコーナーを再構成する。
// script 系で採用コーナーを再現するために使う（回番号不要・再実行で不変）。
func resolveCornersByRundown(corners []config.CornerConfig, rd model.Rundown) ([]config.CornerConfig, error) {
	titles := make([]string, len(rd.Corners))
	for i, c := range rd.Corners {
		titles[i] = c.Title
	}
	return config.ResolveCornersByTitles(corners, titles)
}

// resolveCorners は回番号で採用コーナーを絞り込む。
// 回番号不明（0）の場合は全コーナーを採用し、条件付きコーナーが存在するときは警告を出す。
func resolveCorners(corners []config.CornerConfig, episodeNumber int, logger *slog.Logger) []config.CornerConfig {
	if episodeNumber == 0 && slices.ContainsFunc(corners, func(c config.CornerConfig) bool { return c.Condition != nil }) {
		logger.Warn("回番号が不明なため、条件付きコーナーを含む全コーナーを採用します")
	}
	return config.ResolveCornersForEpisode(corners, episodeNumber)
}

// loadCacheEntries loads all cache entries for the given program.
// Returns (entries, nextEpisodeNumber). Both are zero values if cache is disabled, programID is empty, or load fails.
func loadCacheEntries(cfg *config.Config, programID string) ([]cache.Entry, int) {
	if !cfg.Cache.Enabled || programID == "" {
		return make([]cache.Entry, 0), 0
	}
	cachePath := filepath.Join(".vox-radio", "cache", programID+".jsonl")
	mgr := cache.New(cachePath)
	entries, err := mgr.Load()
	if err != nil {
		return make([]cache.Entry, 0), 0
	}
	return entries, cache.NextEpisodeNumber(entries)
}

// resolveEpisodeNumber returns the next episode number from cache.
// Returns 0 if cache is disabled, programID is empty, or cache fails to load.
func resolveEpisodeNumber(cfg *config.Config, programID string) int {
	_, n := loadCacheEntries(cfg, programID)
	return n
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
