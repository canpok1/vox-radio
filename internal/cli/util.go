package cli

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"slices"
	"time"

	"github.com/canpok1/vox-radio/internal/cache"
	"github.com/canpok1/vox-radio/internal/config"
	"github.com/canpok1/vox-radio/internal/fileio"
	"github.com/canpok1/vox-radio/internal/guest"
	"github.com/canpok1/vox-radio/internal/logging"
	"github.com/canpok1/vox-radio/internal/model"
	"github.com/canpok1/vox-radio/internal/script/llm"
	"github.com/spf13/cobra"
)

// registerSpecFlag registers the required --spec flag on cmd, binding it
// to specPath. Used by episodegen subcommands that load an episode spec (assemble is the
// exception: its --spec is optional because assets can be skipped).
// Note: feedgen uses --spec too but registers it inline with a different description.
func registerSpecFlag(cmd *cobra.Command, specPath *string) {
	cmd.Flags().StringVar(specPath, "spec", "", "エピソード仕様 YAML ファイルのパス（必須）")
	_ = cmd.MarkFlagRequired("spec")
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

// selectGuests selects guests for the given episode number and warns when guests are configured
// but the episode number is unknown.
func selectGuests(guests map[string]config.GuestConfig, episodeNumber int, logger *slog.Logger) []model.RundownGuest {
	selected := guest.Select(guests, episodeNumber)
	if len(guests) > 0 && episodeNumber == 0 {
		logger.Warn("ゲストが設定されていますが回番号が不明なため、ゲストは出演しません")
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

// resolveEpisodeNumber returns the next episode number from cache.
// Returns 0 if cache is disabled, programID is empty, or cache fails to load.
func resolveEpisodeNumber(cfg *config.Config, programID string) int {
	if !cfg.Cache.Enabled || programID == "" {
		return 0
	}
	cachePath := filepath.Join(".vox-radio", "cache", programID+".jsonl")
	mgr := cache.New(cachePath)
	entries, err := mgr.Load()
	if err != nil {
		return 0
	}
	return cache.NextEpisodeNumber(entries)
}

func loadConfigAndSpec(specPath string) (*config.Config, *config.EpisodeSpec, error) {
	cfg, err := config.LoadConfig("vox-radio.yaml")
	if err != nil {
		return nil, nil, fmt.Errorf("load config: %w", err)
	}
	p, err := config.LoadEpisodeSpec(specPath)
	if err != nil {
		return nil, nil, fmt.Errorf("load spec: %w", err)
	}
	if err := config.ValidateEpisodeSpecCast(p, cfg.Characters); err != nil {
		return nil, nil, fmt.Errorf("spec validation: %w", err)
	}
	if err := config.ValidateEpisodeSpecAssets(p); err != nil {
		return nil, nil, fmt.Errorf("spec validation: %w", err)
	}
	if err := config.ValidateEpisodeSpecGuests(p, cfg.Characters); err != nil {
		return nil, nil, fmt.Errorf("spec validation: %w", err)
	}
	if err := config.ValidateEpisodeSpecCorners(p); err != nil {
		return nil, nil, fmt.Errorf("spec validation: %w", err)
	}
	return cfg, p, nil
}
