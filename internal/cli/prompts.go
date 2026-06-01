package cli

import (
	"embed"
	"fmt"
)

//go:embed prompts/*.md
var promptsFS embed.FS

func loadPrompts() (map[string]string, error) {
	names := []string{"select", "summarize", "write", "direct", "summary", "corner_summary"}
	prompts := make(map[string]string, len(names))
	for _, name := range names {
		data, err := promptsFS.ReadFile("prompts/" + name + ".md")
		if err != nil {
			return nil, fmt.Errorf("read %s.md: %w", name, err)
		}
		prompts[name] = string(data)
	}
	return prompts, nil
}
