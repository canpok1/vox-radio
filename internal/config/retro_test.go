package config_test

import (
	"testing"

	"github.com/canpok1/vox-radio/internal/config"
)

func TestRetroConfig_EffectiveMaxTries_Default(t *testing.T) {
	c := config.RetroConfig{}
	if got := c.EffectiveMaxTries(); got != config.DefaultRetroMaxTries {
		t.Errorf("EffectiveMaxTries() = %d, want %d", got, config.DefaultRetroMaxTries)
	}
}

func TestRetroConfig_EffectiveMaxTries_Configured(t *testing.T) {
	c := config.RetroConfig{MaxTries: 5}
	if got := c.EffectiveMaxTries(); got != 5 {
		t.Errorf("EffectiveMaxTries() = %d, want 5", got)
	}
}

func TestRetroConfig_EffectiveAnalysisEntries_Default(t *testing.T) {
	c := config.RetroConfig{}
	if got := c.EffectiveAnalysisEntries(); got != config.DefaultRetroAnalysisEntries {
		t.Errorf("EffectiveAnalysisEntries() = %d, want %d", got, config.DefaultRetroAnalysisEntries)
	}
}

func TestRetroConfig_EffectiveAnalysisEntries_Configured(t *testing.T) {
	c := config.RetroConfig{AnalysisEntries: 8}
	if got := c.EffectiveAnalysisEntries(); got != 8 {
		t.Errorf("EffectiveAnalysisEntries() = %d, want 8", got)
	}
}
