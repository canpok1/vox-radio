package cli

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/canpok1/vox-radio/internal/config"
)

// lookPath is exec.LookPath by default; replaced in tests.
var lookPath = exec.LookPath

// requireLLMKey checks that the API key environment variable for the configured LLM
// provider is set to a non-empty value. It resolves the api_key_env name for the
// active provider (openai / dify-chat) and verifies os.Getenv is non-empty.
//
// This is a cheap, no-cost startup check (no LLM call): an invalid key still surfaces
// on the first real Complete call early in the pipeline, but a missing key is caught
// immediately at startup. os.Getenv is used only in the CLI layer (see architecture.md).
func requireLLMKey(cfg *config.Config) error {
	var envName string
	switch cfg.LLM.EffectiveProvider() {
	case config.ProviderDifyChat:
		if cfg.LLM.DifyChat != nil {
			envName = cfg.LLM.DifyChat.APIKeyEnv
		}
	default:
		if cfg.LLM.OpenAI != nil {
			envName = cfg.LLM.OpenAI.APIKeyEnv
		}
	}
	if envName == "" {
		return fmt.Errorf("LLM の api_key_env が設定されていません。設定ファイルの llm を確認してください")
	}
	if os.Getenv(envName) == "" {
		return fmt.Errorf("LLM API キーの環境変数 %s が設定されていません。%s を設定してください", envName, envName)
	}
	return nil
}

// checkResources runs all given resource checks and aggregates their failures into a
// single error listing each missing resource, so the user sees every problem at once
// instead of fixing them one run at a time. Returns nil when all checks pass.
func checkResources(checks ...func() error) error {
	var errs []error
	for _, check := range checks {
		if err := check(); err != nil {
			errs = append(errs, err)
		}
	}
	if len(errs) == 0 {
		return nil
	}
	// errors.Join で集約し、errors.Is/As のアンラップを保つ（architecture.md §6）。
	// 先頭に見出しを付けて不足リソースを一覧提示する。
	return errors.Join(append([]error{errors.New("起動時リソースチェックに失敗しました")}, errs...)...)
}

// requireMediaTools checks that ffmpeg and ffprobe are available in PATH.
// If any are missing, returns an error listing all absent tools with a link to the README.
func requireMediaTools() error {
	tools := []string{"ffmpeg", "ffprobe"}
	var missing []string
	for _, tool := range tools {
		if _, err := lookPath(tool); err != nil {
			missing = append(missing, tool)
		}
	}
	if len(missing) == 0 {
		return nil
	}
	names := strings.Join(missing, ", ")
	return fmt.Errorf("%s が見つかりません。音声の生成には ffmpeg および ffprobe が必要です。\nインストール手順は vox-radio の README を参照してください:\n%s", names, referenceURL("README.md"))
}
