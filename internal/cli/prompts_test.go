package cli

import (
	"testing"
)

func TestLoadPrompts(t *testing.T) {
	prompts, err := loadPrompts()
	if err != nil {
		t.Fatalf("loadPrompts() error = %v", err)
	}

	expectedKeys := []string{"select", "summarize", "write", "direct", "proofread", "summary", "corner_summary"}
	for _, key := range expectedKeys {
		t.Run(key, func(t *testing.T) {
			val, ok := prompts[key]
			if !ok {
				t.Errorf("prompts[%q] not found", key)
				return
			}
			if val == "" {
				t.Errorf("prompts[%q] is empty", key)
			}
		})
	}
}
