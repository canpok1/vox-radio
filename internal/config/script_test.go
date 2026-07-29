package config_test

import (
	"testing"

	"github.com/canpok1/vox-radio/internal/config"
)

func TestScriptConfig_EffectiveRegenThreshold_Default(t *testing.T) {
	c := config.ScriptConfig{}
	if got := c.EffectiveRegenThreshold(); got != config.DefaultScriptRegenThreshold {
		t.Errorf("EffectiveRegenThreshold() = %v, want %v", got, config.DefaultScriptRegenThreshold)
	}
}

func TestScriptConfig_EffectiveRegenThreshold_Configured(t *testing.T) {
	c := config.ScriptConfig{RegenThreshold: 0.35}
	if got := c.EffectiveRegenThreshold(); got != 0.35 {
		t.Errorf("EffectiveRegenThreshold() = %v, want 0.35", got)
	}
}

func TestScriptConfig_EffectiveRegenMaxRetries_Default(t *testing.T) {
	c := config.ScriptConfig{}
	if got := c.EffectiveRegenMaxRetries(); got != config.DefaultScriptRegenMaxRetries {
		t.Errorf("EffectiveRegenMaxRetries() = %d, want %d", got, config.DefaultScriptRegenMaxRetries)
	}
}

func TestScriptConfig_EffectiveRegenMaxRetries_Configured(t *testing.T) {
	c := config.ScriptConfig{RegenMaxRetries: 3}
	if got := c.EffectiveRegenMaxRetries(); got != 3 {
		t.Errorf("EffectiveRegenMaxRetries() = %d, want 3", got)
	}
}
