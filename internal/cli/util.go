package cli

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/canpok1/vox-radio/internal/config"
	"github.com/canpok1/vox-radio/internal/fileio"
	"github.com/canpok1/vox-radio/internal/logging"
	"github.com/spf13/cobra"
)

// registerProfileFlag registers the required --profile flag on cmd, binding it
// to profilePath. Used by every command that loads a profile (assemble is the
// exception: its --profile is optional because assets can be skipped).
func registerProfileFlag(cmd *cobra.Command, profilePath *string) {
	cmd.Flags().StringVar(profilePath, "profile", "", "プロファイル YAML ファイルのパス（必須）")
	_ = cmd.MarkFlagRequired("profile")
}

func writeJSON(path string, v any) error {
	return fileio.WriteJSON(path, v)
}

func readJSON[T any](path string) (T, error) {
	var v T
	data, err := os.ReadFile(path)
	if err != nil {
		return v, err
	}
	if err := json.Unmarshal(data, &v); err != nil {
		return v, err
	}
	return v, nil
}

// setupLogger creates a fan-out logger (stderr INFO+, logFile DEBUG+) and returns it with the
// log file handle. The caller must close the file when done.
// logDir defaults to "./logs" if empty.
func setupLogger(commandName string, logDir string) (*slog.Logger, *os.File, error) {
	if logDir == "" {
		logDir = "logs"
	}
	return logging.NewSetup(time.Now(), commandName, logDir)
}

func loadConfigAndProfile(profilePath string) (*config.Config, *config.Profile, error) {
	cfg, err := config.LoadConfig("vox-radio.yaml")
	if err != nil {
		return nil, nil, fmt.Errorf("load config: %w", err)
	}
	p, err := config.LoadProfile(profilePath)
	if err != nil {
		return nil, nil, fmt.Errorf("load profile: %w", err)
	}
	if err := config.ValidateProfileCast(p, cfg.Characters); err != nil {
		return nil, nil, fmt.Errorf("profile validation: %w", err)
	}
	return cfg, p, nil
}
