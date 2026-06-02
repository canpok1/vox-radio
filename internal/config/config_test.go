package config_test

import (
	"path/filepath"
	"testing"

	"github.com/canpok1/vox-radio/internal/config"
)

func TestDurationSecToTargetChars(t *testing.T) {
	tests := []struct {
		sec  int
		want int
	}{
		{sec: 0, want: 0},
		{sec: 1, want: 7},
		{sec: 14, want: 98},
		{sec: 30, want: 210},
	}
	for _, tt := range tests {
		got := config.DurationSecToTargetChars(tt.sec)
		if got != tt.want {
			t.Errorf("DurationSecToTargetChars(%d) = %d, want %d", tt.sec, got, tt.want)
		}
	}
}

func TestLoadConfig(t *testing.T) {
	cfg, err := config.LoadConfig("testdata/config.yaml")
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}

	t.Run("LLM", func(t *testing.T) {
		if cfg.LLM.EffectiveProvider() == "" {
			t.Error("LLM.EffectiveProvider() must not be empty")
		}
		if cfg.LLM.OpenAI == nil {
			t.Fatal("LLM.OpenAI block must not be nil")
		}
		if cfg.LLM.OpenAI.BaseURL == "" {
			t.Error("LLM.OpenAI.BaseURL must not be empty")
		}
		if cfg.LLM.OpenAI.APIKeyEnv == "" {
			t.Error("LLM.OpenAI.APIKeyEnv must not be empty")
		}
		if cfg.LLM.OpenAI.Model == "" {
			t.Error("LLM.OpenAI.Model must not be empty")
		}
		if cfg.LLM.MaxRetries <= 0 {
			t.Error("LLM.MaxRetries must be positive")
		}
		if len(cfg.LLM.Steps) == 0 {
			t.Error("LLM.Steps must not be empty")
		}
	})
}

func TestLoadConfig_MissingFile(t *testing.T) {
	_, err := config.LoadConfig("testdata/nonexistent.yaml")
	if err == nil {
		t.Error("expected error for missing file")
	}
}

func TestLoadProfile(t *testing.T) {
	profile, err := config.LoadProfile("testdata/profile.yaml")
	if err != nil {
		t.Fatalf("LoadProfile failed: %v", err)
	}

	t.Run("Program", func(t *testing.T) {
		if profile.Program.Title == "" {
			t.Error("Program.Title must not be empty")
		}
	})

	t.Run("CornerAssets", func(t *testing.T) {
		if len(profile.Corners) == 0 {
			t.Fatal("Corners must not be empty")
		}
		corner0 := profile.Corners[0]
		if corner0.StartJingle == "" {
			t.Error("Corners[0].StartJingle must not be empty")
		}
		if len(profile.Corners) > 1 {
			corner1 := profile.Corners[1]
			if corner1.EndJingle == "" {
				t.Error("Corners[1].EndJingle must not be empty")
			}
			if corner1.BGM == "" {
				t.Error("Corners[1].BGM must not be empty")
			}
		}
	})

	t.Run("Corners", func(t *testing.T) {
		if len(profile.Corners) == 0 {
			t.Error("Corners must not be empty")
		}
		c := profile.Corners[0]
		if c.Title == "" {
			t.Error("Corners[0].Title must not be empty")
		}
		if c.Content == "" {
			t.Error("Corners[0].Content must not be empty")
		}
		if len(c.Cast) == 0 {
			t.Error("Corners[0].Cast must not be empty")
		}
		if c.LengthSec <= 0 {
			t.Error("Corners[0].LengthSec must be positive")
		}
	})

	t.Run("CornerSource", func(t *testing.T) {
		var sourceCorner *config.CornerConfig
		for i := range profile.Corners {
			if profile.Corners[i].Source != nil {
				sourceCorner = &profile.Corners[i]
				break
			}
		}
		if sourceCorner == nil {
			t.Fatal("no corner with source found")
		}
		if len(sourceCorner.Source.Feeds) == 0 {
			t.Error("Source.Feeds must not be empty")
		}
		if sourceCorner.Source.Feeds[0].URL == "" {
			t.Error("Source.Feeds[0].URL must not be empty")
		}
		if len(sourceCorner.Source.Articles) == 0 {
			t.Error("Source.Articles must not be empty")
		}
	})

	t.Run("Assets", func(t *testing.T) {
		if len(profile.Assets.Jingle) == 0 {
			t.Error("Assets.Jingle must not be empty")
		}
		if len(profile.Assets.SE) == 0 {
			t.Error("Assets.SE must not be empty")
		}
		if len(profile.Assets.BGM) == 0 {
			t.Error("Assets.BGM must not be empty")
		}
	})

	t.Run("Assets_PathResolution", func(t *testing.T) {
		base := "testdata"

		jingle, ok := profile.Assets.Jingle["opening"]
		if !ok {
			t.Fatal("Assets.Jingle[\"opening\"] not found")
		}
		if want := filepath.Join(base, "assets/jingle/opening.mp3"); jingle.File != want {
			t.Errorf("Jingle[\"opening\"].File: expected %q, got %q", want, jingle.File)
		}

		se, ok := profile.Assets.SE["chime"]
		if !ok {
			t.Fatal("Assets.SE[\"chime\"] not found")
		}
		if want := filepath.Join(base, "assets/se/chime.wav"); se.File != want {
			t.Errorf("SE[\"chime\"].File: expected %q, got %q", want, se.File)
		}

		bgm, ok := profile.Assets.BGM["talk_bgm"]
		if !ok {
			t.Fatal("Assets.BGM[\"talk_bgm\"] not found")
		}
		if want := filepath.Join(base, "assets/bgm/talk.mp3"); bgm.File != want {
			t.Errorf("BGM[\"talk_bgm\"].File: expected %q, got %q", want, bgm.File)
		}
	})
}

func TestLoadProfile_AbsolutePaths(t *testing.T) {
	profile, err := config.LoadProfile("testdata/profile_abs.yaml")
	if err != nil {
		t.Fatalf("LoadProfile failed: %v", err)
	}

	for name, entry := range profile.Assets.Jingle {
		if !filepath.IsAbs(entry.File) {
			t.Errorf("Jingle[%q].File should remain absolute, got %q", name, entry.File)
		}
	}
	for name, entry := range profile.Assets.SE {
		if !filepath.IsAbs(entry.File) {
			t.Errorf("SE[%q].File should remain absolute, got %q", name, entry.File)
		}
	}
	for name, entry := range profile.Assets.BGM {
		if !filepath.IsAbs(entry.File) {
			t.Errorf("BGM[%q].File should remain absolute, got %q", name, entry.File)
		}
	}
}

func TestLoadProfile_MissingFile(t *testing.T) {
	_, err := config.LoadProfile("testdata/nonexistent.yaml")
	if err == nil {
		t.Error("expected error for missing file")
	}
}

func TestValidateProfileAssets_Valid(t *testing.T) {
	p := &config.Profile{
		Corners: []config.CornerConfig{
			{Title: "C1", StartJingle: "opening", EndJingle: "ending", BGM: "talk_bgm"},
		},
		Assets: config.AssetsConfig{
			Jingle: map[string]config.JingleEntry{
				"opening": {File: "opening.mp3"},
				"ending":  {File: "ending.mp3"},
			},
			BGM: map[string]config.BGMEntry{
				"talk_bgm": {File: "bgm.mp3"},
			},
		},
	}
	if err := config.ValidateProfileAssets(p); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestValidateProfileAssets_UnknownStartJingle(t *testing.T) {
	p := &config.Profile{
		Corners: []config.CornerConfig{
			{Title: "C1", StartJingle: "nonexistent"},
		},
		Assets: config.AssetsConfig{
			Jingle: map[string]config.JingleEntry{},
			BGM:    map[string]config.BGMEntry{},
		},
	}
	if err := config.ValidateProfileAssets(p); err == nil {
		t.Error("expected error for unknown start_jingle key")
	}
}

func TestValidateProfileAssets_UnknownEndJingle(t *testing.T) {
	p := &config.Profile{
		Corners: []config.CornerConfig{
			{Title: "C1", EndJingle: "nonexistent"},
		},
		Assets: config.AssetsConfig{
			Jingle: map[string]config.JingleEntry{},
			BGM:    map[string]config.BGMEntry{},
		},
	}
	if err := config.ValidateProfileAssets(p); err == nil {
		t.Error("expected error for unknown end_jingle key")
	}
}

func TestValidateProfileAssets_UnknownBGM(t *testing.T) {
	p := &config.Profile{
		Corners: []config.CornerConfig{
			{Title: "C1", BGM: "nonexistent"},
		},
		Assets: config.AssetsConfig{
			Jingle: map[string]config.JingleEntry{},
			BGM:    map[string]config.BGMEntry{},
		},
	}
	if err := config.ValidateProfileAssets(p); err == nil {
		t.Error("expected error for unknown bgm key")
	}
}

func TestValidateProfileAssets_EmptyFields_NoError(t *testing.T) {
	p := &config.Profile{
		Corners: []config.CornerConfig{
			{Title: "C1"},
		},
		Assets: config.AssetsConfig{
			Jingle: map[string]config.JingleEntry{},
			BGM:    map[string]config.BGMEntry{},
		},
	}
	if err := config.ValidateProfileAssets(p); err != nil {
		t.Errorf("unexpected error for empty fields: %v", err)
	}
}

func TestLoadProfile_ValidateProfileAssetsIntegration(t *testing.T) {
	profile, err := config.LoadProfile("testdata/profile.yaml")
	if err != nil {
		t.Fatalf("LoadProfile failed: %v", err)
	}
	if err := config.ValidateProfileAssets(profile); err != nil {
		t.Errorf("ValidateProfileAssets failed on testdata profile: %v", err)
	}
}

func TestLoadConfig_Voicevox(t *testing.T) {
	cfg, err := config.LoadConfig("testdata/config.yaml")
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}
	if cfg.Voicevox.URL == "" {
		t.Error("Voicevox.URL must not be empty")
	}
}

func TestLoadConfig_Characters(t *testing.T) {
	cfg, err := config.LoadConfig("testdata/config.yaml")
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}
	if len(cfg.Characters) == 0 {
		t.Error("Characters must not be empty")
	}

	ch, ok := cfg.Characters["zundamon"]
	if !ok {
		t.Fatal("Characters[\"zundamon\"] not found")
	}
	if ch.Name == "" {
		t.Error("CharacterConfig.Name must not be empty")
	}
	if ch.Pronoun == "" {
		t.Error("CharacterConfig.Pronoun must not be empty")
	}
	if len(ch.SpeechSuffix) == 0 {
		t.Error("CharacterConfig.SpeechSuffix must not be empty")
	}
	if len(ch.Personality) == 0 {
		t.Error("CharacterConfig.Personality must not be empty")
	}
	if ch.DefaultStyle == "" {
		t.Error("CharacterConfig.DefaultStyle must not be empty")
	}
	if len(ch.Styles) == 0 {
		t.Error("CharacterConfig.Styles must not be empty")
	}
	if _, ok := ch.Styles[ch.DefaultStyle]; !ok {
		t.Errorf("DefaultStyle %q not found in Styles", ch.DefaultStyle)
	}
}

func TestLoadConfig_ValidationError_DefaultStyleNotInStyles(t *testing.T) {
	_, err := config.LoadConfig("testdata/config_invalid_default_style.yaml")
	if err == nil {
		t.Error("expected error when default_style is not in styles")
	}
}

func TestLLMConfig_EffectiveMinRequestIntervalMS_Unspecified(t *testing.T) {
	c := config.LLMConfig{}
	got := c.EffectiveMinRequestIntervalMS()
	if got != config.DefaultMinRequestIntervalMS {
		t.Errorf("got %d, want DefaultMinRequestIntervalMS=%d", got, config.DefaultMinRequestIntervalMS)
	}
}

func TestLLMConfig_EffectiveMinRequestIntervalMS_Zero(t *testing.T) {
	v := 0
	c := config.LLMConfig{MinRequestIntervalMS: &v}
	got := c.EffectiveMinRequestIntervalMS()
	if got != 0 {
		t.Errorf("got %d, want 0 (throttling disabled)", got)
	}
}

func TestLLMConfig_EffectiveMinRequestIntervalMS_Custom(t *testing.T) {
	v := 1000
	c := config.LLMConfig{MinRequestIntervalMS: &v}
	got := c.EffectiveMinRequestIntervalMS()
	if got != 1000 {
		t.Errorf("got %d, want 1000", got)
	}
}

func TestLoadConfig_LLM_MinRequestIntervalMS(t *testing.T) {
	cfg, err := config.LoadConfig("testdata/config_with_interval.yaml")
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}
	if cfg.LLM.MinRequestIntervalMS == nil {
		t.Fatal("LLM.MinRequestIntervalMS should be non-nil when specified in YAML")
	}
	if *cfg.LLM.MinRequestIntervalMS != 2000 {
		t.Errorf("LLM.MinRequestIntervalMS: got %d, want 2000", *cfg.LLM.MinRequestIntervalMS)
	}
}

func TestLoadProfile_CornerDirection(t *testing.T) {
	profile, err := config.LoadProfile("testdata/profile.yaml")
	if err != nil {
		t.Fatalf("LoadProfile failed: %v", err)
	}

	var directionCorner *config.CornerConfig
	for i := range profile.Corners {
		if profile.Corners[i].Direction != "" {
			directionCorner = &profile.Corners[i]
			break
		}
	}
	if directionCorner == nil {
		t.Fatal("no corner with direction found")
	}
	if directionCorner.Direction != "冒頭でオープニングジングルを流す。" {
		t.Errorf("Direction: got %q, want %q", directionCorner.Direction, "冒頭でオープニングジングルを流す。")
	}
}

func TestValidateProfileCast_Valid(t *testing.T) {
	p := &config.Profile{
		Corners: []config.CornerConfig{
			{Title: "opening", Cast: map[string]string{"zundamon": "司会"}},
		},
	}
	chars := map[string]config.CharacterConfig{
		"zundamon": {Name: "ずんだもん"},
	}
	if err := config.ValidateProfileCast(p, chars); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestValidateProfileCast_UnknownCharacter(t *testing.T) {
	p := &config.Profile{
		Corners: []config.CornerConfig{
			{Title: "opening", Cast: map[string]string{"unknown_char": "司会"}},
		},
	}
	chars := map[string]config.CharacterConfig{
		"zundamon": {Name: "ずんだもん"},
	}
	if err := config.ValidateProfileCast(p, chars); err == nil {
		t.Error("expected error for unknown character in cast")
	}
}

func TestCharacterConfig_SpeakerID_ValidStyle(t *testing.T) {
	ch := config.CharacterConfig{
		DefaultStyle: "ノーマル",
		Styles:       map[string]int{"ノーマル": 3, "なみだめ": 76},
	}
	id, ok := ch.SpeakerID("なみだめ")
	if !ok {
		t.Fatal("SpeakerID: expected ok=true for valid style")
	}
	if id != 76 {
		t.Errorf("SpeakerID: got %d, want 76", id)
	}
}

func TestCharacterConfig_SpeakerID_EmptyStyle(t *testing.T) {
	ch := config.CharacterConfig{
		DefaultStyle: "ノーマル",
		Styles:       map[string]int{"ノーマル": 3, "なみだめ": 76},
	}
	id, ok := ch.SpeakerID("")
	if !ok {
		t.Fatal("SpeakerID: expected ok=true with empty style (fallback to default)")
	}
	if id != 3 {
		t.Errorf("SpeakerID: got %d, want 3 (default)", id)
	}
}

func TestCharacterConfig_SpeakerID_InvalidStyle(t *testing.T) {
	ch := config.CharacterConfig{
		DefaultStyle: "ノーマル",
		Styles:       map[string]int{"ノーマル": 3, "なみだめ": 76},
	}
	id, ok := ch.SpeakerID("存在しない")
	if !ok {
		t.Fatal("SpeakerID: expected ok=true when invalid style falls back to default")
	}
	if id != 3 {
		t.Errorf("SpeakerID: got %d, want 3 (default)", id)
	}
}

func TestVoicevoxConfig_EffectivePresets_UsesDefaultWhenNotSet(t *testing.T) {
	cfg := config.VoicevoxConfig{}
	presets := cfg.EffectivePresets()

	if v, ok := presets.Intonation["標準"]; !ok || v != 1.0 {
		t.Errorf("Intonation[\"標準\"]: got %v (ok=%v), want 1.0", v, ok)
	}
	if v, ok := presets.Pitch["標準"]; !ok || v != 0.0 {
		t.Errorf("Pitch[\"標準\"]: got %v (ok=%v), want 0.0", v, ok)
	}
	if v, ok := presets.Speed["標準"]; !ok || v != 1.0 {
		t.Errorf("Speed[\"標準\"]: got %v (ok=%v), want 1.0", v, ok)
	}
}

func TestVoicevoxConfig_EffectivePresets_UsesYAMLWhenSet(t *testing.T) {
	cfg, err := config.LoadConfig("testdata/config_with_presets.yaml")
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}
	presets := cfg.Voicevox.EffectivePresets()

	if v, ok := presets.Intonation["棒読み"]; !ok || v != 0.0 {
		t.Errorf("Intonation[\"棒読み\"]: got %v (ok=%v), want 0.0", v, ok)
	}
	if v, ok := presets.Pitch["低め"]; !ok || v != -0.05 {
		t.Errorf("Pitch[\"低め\"]: got %v (ok=%v), want -0.05", v, ok)
	}
	if v, ok := presets.Speed["ゆっくり"]; !ok || v != 0.8 {
		t.Errorf("Speed[\"ゆっくり\"]: got %v (ok=%v), want 0.8", v, ok)
	}
}

func TestVoicevoxConfig_EffectivePresets_FallsBackPerAxis(t *testing.T) {
	p := &config.VoicevoxPresets{
		Intonation: map[string]float64{"カスタム": 1.3},
		// Pitch and Speed are nil
	}
	cfg := config.VoicevoxConfig{Presets: p}
	presets := cfg.EffectivePresets()

	if _, ok := presets.Intonation["カスタム"]; !ok {
		t.Error("Intonation[\"カスタム\"] should be present from YAML")
	}
	if v, ok := presets.Pitch["標準"]; !ok || v != 0.0 {
		t.Errorf("Pitch[\"標準\"]: got %v (ok=%v), want 0.0 (fallback to default)", v, ok)
	}
	if v, ok := presets.Speed["標準"]; !ok || v != 1.0 {
		t.Errorf("Speed[\"標準\"]: got %v (ok=%v), want 1.0 (fallback to default)", v, ok)
	}
}

func TestVoicevoxPresets_Resolve_KnownName(t *testing.T) {
	p := config.VoicevoxPresets{
		Intonation: map[string]float64{"標準": 1.0},
		Pitch:      map[string]float64{"標準": 0.0},
		Speed:      map[string]float64{"標準": 1.0},
	}

	if v, ok := p.ResolveIntonation("標準"); !ok || v != 1.0 {
		t.Errorf("ResolveIntonation(\"標準\"): got %v (ok=%v), want 1.0 true", v, ok)
	}
	if v, ok := p.ResolvePitch("標準"); !ok || v != 0.0 {
		t.Errorf("ResolvePitch(\"標準\"): got %v (ok=%v), want 0.0 true", v, ok)
	}
	if v, ok := p.ResolveSpeed("標準"); !ok || v != 1.0 {
		t.Errorf("ResolveSpeed(\"標準\"): got %v (ok=%v), want 1.0 true", v, ok)
	}
}

func TestVoicevoxPresets_Resolve_EmptyName(t *testing.T) {
	p := config.VoicevoxPresets{
		Intonation: map[string]float64{"標準": 1.0},
		Pitch:      map[string]float64{"標準": 0.0},
		Speed:      map[string]float64{"標準": 1.0},
	}

	if _, ok := p.ResolveIntonation(""); ok {
		t.Error("ResolveIntonation(\"\") should return ok=false for empty name")
	}
	if _, ok := p.ResolvePitch(""); ok {
		t.Error("ResolvePitch(\"\") should return ok=false for empty name")
	}
	if _, ok := p.ResolveSpeed(""); ok {
		t.Error("ResolveSpeed(\"\") should return ok=false for empty name")
	}
}

func TestVoicevoxPresets_Resolve_UnknownName(t *testing.T) {
	p := config.VoicevoxPresets{
		Intonation: map[string]float64{"標準": 1.0},
		Pitch:      map[string]float64{"標準": 0.0},
		Speed:      map[string]float64{"標準": 1.0},
	}

	if _, ok := p.ResolveIntonation("存在しない"); ok {
		t.Error("ResolveIntonation(\"存在しない\") should return ok=false for unknown name")
	}
	if _, ok := p.ResolvePitch("存在しない"); ok {
		t.Error("ResolvePitch(\"存在しない\") should return ok=false for unknown name")
	}
	if _, ok := p.ResolveSpeed("存在しない"); ok {
		t.Error("ResolveSpeed(\"存在しない\") should return ok=false for unknown name")
	}
}

func TestLoadConfig_ValidationError_PresetOutOfRange(t *testing.T) {
	_, err := config.LoadConfig("testdata/config_invalid_preset_range.yaml")
	if err == nil {
		t.Error("expected error when preset value is out of range")
	}
}

func TestCacheConfig_EffectiveMaxEntries_Zero(t *testing.T) {
	c := config.CacheConfig{}
	if got := c.EffectiveMaxEntries(); got != config.DefaultCacheMaxEntries {
		t.Errorf("got %d, want DefaultCacheMaxEntries=%d", got, config.DefaultCacheMaxEntries)
	}
}

func TestCacheConfig_EffectiveMaxEntries_Custom(t *testing.T) {
	c := config.CacheConfig{MaxEntries: 50}
	if got := c.EffectiveMaxEntries(); got != 50 {
		t.Errorf("got %d, want 50", got)
	}
}

func TestCacheConfig_EffectiveRetentionDays_Zero(t *testing.T) {
	c := config.CacheConfig{}
	if got := c.EffectiveRetentionDays(); got != config.DefaultCacheRetentionDays {
		t.Errorf("got %d, want DefaultCacheRetentionDays=%d", got, config.DefaultCacheRetentionDays)
	}
}

func TestCacheConfig_EffectiveRetentionDays_Custom(t *testing.T) {
	c := config.CacheConfig{RetentionDays: 30}
	if got := c.EffectiveRetentionDays(); got != 30 {
		t.Errorf("got %d, want 30", got)
	}
}

func TestCacheConfig_EffectiveLLMContextEntries_Zero(t *testing.T) {
	c := config.CacheConfig{}
	if got := c.EffectiveLLMContextEntries(); got != config.DefaultCacheLLMContextEntries {
		t.Errorf("got %d, want DefaultCacheLLMContextEntries=%d", got, config.DefaultCacheLLMContextEntries)
	}
}

func TestCacheConfig_EffectiveLLMContextEntries_Custom(t *testing.T) {
	c := config.CacheConfig{LLMContextEntries: 5}
	if got := c.EffectiveLLMContextEntries(); got != 5 {
		t.Errorf("got %d, want 5", got)
	}
}

func TestLoadProfile_ProgramID(t *testing.T) {
	profile, err := config.LoadProfile("testdata/profile_with_id.yaml")
	if err != nil {
		t.Fatalf("LoadProfile failed: %v", err)
	}
	if profile.Program.ID != "test-program" {
		t.Errorf("Program.ID: got %q, want %q", profile.Program.ID, "test-program")
	}
}

func TestLoadProfile_ProgramIDEmpty_NoError(t *testing.T) {
	profile, err := config.LoadProfile("testdata/profile.yaml")
	if err != nil {
		t.Fatalf("LoadProfile failed: %v", err)
	}
	if profile.Program.ID != "" {
		t.Errorf("Program.ID: expected empty, got %q", profile.Program.ID)
	}
}

func TestLoadConfigStrict_UnknownKeyErrors(t *testing.T) {
	_, err := config.LoadConfigStrict("testdata/config_unknown_key.yaml")
	if err == nil {
		t.Error("expected error for unknown key in strict mode")
	}
}

func TestLoadConfigStrict_ValidYAML_Success(t *testing.T) {
	_, err := config.LoadConfigStrict("testdata/config.yaml")
	if err != nil {
		t.Errorf("unexpected error for valid config in strict mode: %v", err)
	}
}

func TestLoadConfig_UnknownKey_NoError(t *testing.T) {
	_, err := config.LoadConfig("testdata/config_unknown_key.yaml")
	if err != nil {
		t.Errorf("LoadConfig should not error on unknown key (non-strict): %v", err)
	}
}

func TestLoadProfileStrict_UnknownKeyErrors(t *testing.T) {
	_, err := config.LoadProfileStrict("testdata/profile_unknown_key.yaml")
	if err == nil {
		t.Error("expected error for unknown key in strict mode")
	}
}

func TestLoadProfileStrict_ValidYAML_Success(t *testing.T) {
	_, err := config.LoadProfileStrict("testdata/profile.yaml")
	if err != nil {
		t.Errorf("unexpected error for valid profile in strict mode: %v", err)
	}
}

func TestLoadProfile_UnknownKey_NoError(t *testing.T) {
	_, err := config.LoadProfile("testdata/profile_unknown_key.yaml")
	if err != nil {
		t.Errorf("LoadProfile should not error on unknown key (non-strict): %v", err)
	}
}

func TestLoadConfig_Cache_DefaultsToDisabled(t *testing.T) {
	cfg, err := config.LoadConfig("testdata/config.yaml")
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}
	if cfg.Cache.Enabled {
		t.Error("Cache.Enabled should default to false when not set in YAML")
	}
}

func TestLoadProfile_AssetsDescription(t *testing.T) {
	profile, err := config.LoadProfile("testdata/profile.yaml")
	if err != nil {
		t.Fatalf("LoadProfile failed: %v", err)
	}

	se, ok := profile.Assets.SE["chime"]
	if !ok {
		t.Fatal("Assets.SE[\"chime\"] not found")
	}
	if se.Description == "" {
		t.Error("SE[\"chime\"].Description must not be empty (testdata should include description)")
	}

	bgm, ok := profile.Assets.BGM["talk_bgm"]
	if !ok {
		t.Fatal("Assets.BGM[\"talk_bgm\"] not found")
	}
	if bgm.Description == "" {
		t.Error("BGM[\"talk_bgm\"].Description must not be empty (testdata should include description)")
	}

	jingle, ok := profile.Assets.Jingle["opening"]
	if !ok {
		t.Fatal("Assets.Jingle[\"opening\"] not found")
	}
	if jingle.Description == "" {
		t.Error("Jingle[\"opening\"].Description must not be empty (testdata should include description)")
	}
}

func TestLLMConfig_EffectiveProvider_Empty(t *testing.T) {
	c := config.LLMConfig{}
	if got := c.EffectiveProvider(); got != config.DefaultProvider {
		t.Errorf("EffectiveProvider() = %q, want %q", got, config.DefaultProvider)
	}
}

func TestLLMConfig_EffectiveProvider_OpenAI(t *testing.T) {
	c := config.LLMConfig{Provider: "openai"}
	if got := c.EffectiveProvider(); got != "openai" {
		t.Errorf("EffectiveProvider() = %q, want %q", got, "openai")
	}
}

func TestLLMConfig_EffectiveProvider_DifyChat(t *testing.T) {
	c := config.LLMConfig{Provider: config.ProviderDifyChat}
	if got := c.EffectiveProvider(); got != config.ProviderDifyChat {
		t.Errorf("EffectiveProvider() = %q, want %q", got, config.ProviderDifyChat)
	}
}

func TestLoadConfig_DifyChat(t *testing.T) {
	cfg, err := config.LoadConfig("testdata/config_dify_chat.yaml")
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}
	if cfg.LLM.EffectiveProvider() != config.ProviderDifyChat {
		t.Errorf("provider = %q, want %q", cfg.LLM.EffectiveProvider(), config.ProviderDifyChat)
	}
	if cfg.LLM.DifyChat == nil {
		t.Fatal("LLM.DifyChat must not be nil")
	}
	if cfg.LLM.DifyChat.BaseURL == "" {
		t.Error("LLM.DifyChat.BaseURL must not be empty")
	}
	if cfg.LLM.DifyChat.APIKeyEnv == "" {
		t.Error("LLM.DifyChat.APIKeyEnv must not be empty")
	}
	if cfg.LLM.DifyChat.User != "vox-radio" {
		t.Errorf("LLM.DifyChat.User = %q, want %q", cfg.LLM.DifyChat.User, "vox-radio")
	}
	if len(cfg.LLM.DifyChat.Inputs) == 0 {
		t.Error("LLM.DifyChat.Inputs must not be empty")
	}
}

func TestLoadConfig_ValidationError_MissingOpenAIBlock(t *testing.T) {
	_, err := config.LoadConfig("testdata/config_missing_openai_block.yaml")
	if err == nil {
		t.Error("expected error when openai provider has no openai block")
	}
}

func TestLoadConfig_ValidationError_MissingDifyChatBlock(t *testing.T) {
	_, err := config.LoadConfig("testdata/config_missing_dify_block.yaml")
	if err == nil {
		t.Error("expected error when dify-chat provider has no dify-chat block")
	}
}

func boolPtr(v bool) *bool { return &v }

func TestJingleEntry_EffectiveTrimSilence(t *testing.T) {
	cases := []struct {
		ptr  *bool
		want bool
	}{
		{nil, true},
		{boolPtr(false), false},
		{boolPtr(true), true},
	}
	for _, c := range cases {
		e := config.JingleEntry{TrimSilence: c.ptr}
		if got := e.EffectiveTrimSilence(); got != c.want {
			t.Errorf("ptr=%v: got %v, want %v", c.ptr, got, c.want)
		}
	}
}

func TestSEEntry_EffectiveTrimSilence(t *testing.T) {
	cases := []struct {
		ptr  *bool
		want bool
	}{
		{nil, true},
		{boolPtr(false), false},
		{boolPtr(true), true},
	}
	for _, c := range cases {
		e := config.SEEntry{TrimSilence: c.ptr}
		if got := e.EffectiveTrimSilence(); got != c.want {
			t.Errorf("ptr=%v: got %v, want %v", c.ptr, got, c.want)
		}
	}
}
