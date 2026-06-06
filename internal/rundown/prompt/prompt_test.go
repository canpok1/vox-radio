package prompt

import (
	"encoding/json"
	"testing"

	"github.com/canpok1/vox-radio/internal/config"
)

func TestNewCornerForPrompt(t *testing.T) {
	corner := config.CornerConfig{Title: "コーナー1", Content: "内容", LengthSec: 30}
	got := NewCornerForPrompt(corner)
	want := CornerForPrompt{Title: "コーナー1", Content: "内容", TargetDurationSeconds: 30}
	if got != want {
		t.Errorf("NewCornerForPrompt() = %+v, want %+v", got, want)
	}
}

func TestCornerForPrompt_JSONTags(t *testing.T) {
	cp := CornerForPrompt{Title: "タイトル", Content: "本文", TargetDurationSeconds: 45}
	b, err := json.Marshal(cp)
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}
	want := `{"title":"タイトル","content":"本文","target_duration_seconds":45}`
	if got := string(b); got != want {
		t.Errorf("Marshal() = %s, want %s", got, want)
	}
}
