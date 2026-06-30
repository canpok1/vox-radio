package config_test

import (
	"testing"

	"github.com/canpok1/vox-radio/internal/config"
)

func TestConfig_EffectivePronunciation_EmptyWhenNotSet(t *testing.T) {
	cfg := config.Config{}
	dict := cfg.EffectivePronunciation()
	if dict == nil {
		t.Fatal("EffectivePronunciation() should never return nil")
	}
	if len(dict) != 0 {
		t.Errorf("EffectivePronunciation(): got %d entries, want 0", len(dict))
	}
}

func TestConfig_EffectivePronunciation_LoadedFromYAML(t *testing.T) {
	cfg, err := config.LoadConfig("testdata/config_with_pronunciation.yaml")
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}
	dict := cfg.EffectivePronunciation()

	want := map[string]string{
		"宮本武蔵": "みやもとむさし",
		"源氏物語": "げんじものがたり",
		"NHK":  "えぬえいちけー",
	}
	if len(dict) != len(want) {
		t.Fatalf("EffectivePronunciation(): got %d entries, want %d", len(dict), len(want))
	}
	for k, v := range want {
		if got := dict[k]; got != v {
			t.Errorf("EffectivePronunciation()[%q]: got %q, want %q", k, got, v)
		}
	}
}

// TestConfig_Pronunciation_DuplicateKeyRejected verifies that defining the same written form
// twice (a reading collision) is rejected at load time by the YAML decoder.
func TestConfig_Pronunciation_DuplicateKeyRejected(t *testing.T) {
	_, err := config.LoadConfig("testdata/config_duplicate_pronunciation.yaml")
	if err == nil {
		t.Fatal("LoadConfig should fail when the same pronunciation key is defined twice")
	}
}
