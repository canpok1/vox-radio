package cli

import (
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/canpok1/vox-radio/internal/config"
	"github.com/canpok1/vox-radio/internal/fileio"
	"github.com/canpok1/vox-radio/internal/logging"
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
	return cfg, p, nil
}
