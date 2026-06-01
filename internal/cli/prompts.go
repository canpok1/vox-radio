package cli

import (
	"embed"
	"fmt"
	"strings"
)

//go:embed prompts/*.md
var promptsFS embed.FS

func loadPrompts() (map[string]string, error) {
	entries, err := promptsFS.ReadDir("prompts")
	if err != nil {
		return nil, fmt.Errorf("read prompts dir: %w", err)
	}
	prompts := make(map[string]string, len(entries))
	for _, e := range entries {
		name := strings.TrimSuffix(e.Name(), ".md")
		data, err := promptsFS.ReadFile("prompts/" + e.Name())
		if err != nil {
			return nil, fmt.Errorf("read %s: %w", e.Name(), err)
		}
		prompts[name] = string(data)
	}
	return prompts, nil
}
