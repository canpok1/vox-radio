package eval

import (
	"strings"
	"testing"
)

func TestLoadPrompt_ContainsExpectedContent(t *testing.T) {
	content, err := LoadPrompt("proofread")
	if err != nil {
		t.Fatalf("LoadPrompt: %v", err)
	}
	if !strings.Contains(content, "{{lines}}") {
		t.Error("proofread.md should contain {{lines}} placeholder")
	}
	if !strings.Contains(content, "corrections") {
		t.Error("proofread.md should contain 'corrections'")
	}
}

func TestLoadPrompt_NotFound(t *testing.T) {
	_, err := LoadPrompt("nonexistent_prompt_xyz")
	if err == nil {
		t.Error("expected error for nonexistent prompt")
	}
}
