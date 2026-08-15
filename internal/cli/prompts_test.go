package cli

import (
	"regexp"
	"strings"
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

// backtickPlaceholderRef matches a placeholder name referenced by its
// template name (e.g. `{{analyses}}`) rather than embedded as a value.
// Rendering replaces {{name}} with the actual data, so a description that
// refers to a block this way resolves to nothing the LLM can read (see
// ADR discussion in tasks/vox-radio/20260815175858).
var backtickPlaceholderRef = regexp.MustCompile("`\\{\\{[a-z_]*\\}\\}`")

func TestPromptsNoBacktickPlaceholderReferences(t *testing.T) {
	prompts, err := loadPrompts()
	if err != nil {
		t.Fatalf("loadPrompts() error = %v", err)
	}

	for name, content := range prompts {
		for i, line := range strings.Split(content, "\n") {
			if m := backtickPlaceholderRef.FindString(line); m != "" {
				t.Errorf("%s.md:%d: %q refers to a placeholder by name; use the section heading it renders under instead", name, i+1, m)
			}
		}
	}
}
