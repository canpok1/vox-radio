package eval

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
)

// LoadPrompt reads the shipped prompt file internal/cli/prompts/{name}.md.
// It resolves the path relative to this source file using runtime.Caller,
// so no duplicate copy of the prompt is created.
func LoadPrompt(name string) (string, error) {
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		return "", fmt.Errorf("runtime.Caller failed")
	}
	// prompt.go lives at internal/eval/prompt.go.
	// Prompts live at internal/cli/prompts/{name}.md.
	evalDir := filepath.Dir(thisFile)
	internalDir := filepath.Dir(evalDir)
	promptPath := filepath.Join(internalDir, "cli", "prompts", name+".md")

	data, err := os.ReadFile(promptPath)
	if err != nil {
		return "", fmt.Errorf("load prompt %q: %w", name, err)
	}
	return string(data), nil
}
