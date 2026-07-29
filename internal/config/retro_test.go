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

func TestRetroConfig_EffectiveKeepThreshold_Default(t *testing.T) {
	c := config.RetroConfig{}
	if got := c.EffectiveKeepThreshold(); got != config.DefaultRetroKeepThreshold {
		t.Errorf("EffectiveKeepThreshold() = %d, want %d", got, config.DefaultRetroKeepThreshold)
	}
}

func TestRetroConfig_EffectiveKeepThreshold_Configured(t *testing.T) {
	c := config.RetroConfig{KeepThreshold: 5}
	if got := c.EffectiveKeepThreshold(); got != 5 {
		t.Errorf("EffectiveKeepThreshold() = %d, want 5", got)
	}
}

func TestRetroConfig_EffectiveKeepLength_Default(t *testing.T) {
	c := config.RetroConfig{}
	if got := c.EffectiveKeepLength(); got != config.DefaultRetroKeepLength {
		t.Errorf("EffectiveKeepLength() = %d, want %d", got, config.DefaultRetroKeepLength)
	}
}

func TestRetroConfig_EffectiveKeepLength_Configured(t *testing.T) {
	c := config.RetroConfig{KeepLength: 1000}
	if got := c.EffectiveKeepLength(); got != 1000 {
		t.Errorf("EffectiveKeepLength() = %d, want 1000", got)
	}
}

func TestRetroConfig_EffectiveMaxFails_Default(t *testing.T) {
	c := config.RetroConfig{}
	if got := c.EffectiveMaxFails(); got != config.DefaultRetroMaxFails {
		t.Errorf("EffectiveMaxFails() = %d, want %d", got, config.DefaultRetroMaxFails)
	}
}

func TestRetroConfig_EffectiveMaxFails_Configured(t *testing.T) {
	c := config.RetroConfig{MaxFails: 10}
	if got := c.EffectiveMaxFails(); got != 10 {
		t.Errorf("EffectiveMaxFails() = %d, want 10", got)
	}
}
