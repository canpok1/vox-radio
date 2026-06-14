package config_test

import (
	"testing"

	"github.com/canpok1/vox-radio/internal/config"
)

func TestCharacterConfig_Credit(t *testing.T) {
	ch := config.CharacterConfig{Credit: "VOICEVOX:ずんだもん"}
	if ch.Credit != "VOICEVOX:ずんだもん" {
		t.Errorf("CharacterConfig.Credit = %q, want %q", ch.Credit, "VOICEVOX:ずんだもん")
	}
}

func TestCharacterConfig_Credit_OmitWhenEmpty(t *testing.T) {
	// credit が設定されていない場合は空文字列であること
	var ch config.CharacterConfig
	if ch.Credit != "" {
		t.Errorf("CharacterConfig.Credit = %q, want empty (omitempty)", ch.Credit)
	}
}
