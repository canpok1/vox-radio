package config_test

import (
	"os"
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

func TestLoadEpisodeSpec(t *testing.T) {
	spec, err := config.LoadEpisodeSpec("testdata/episode_spec.yaml")
	if err != nil {
		t.Fatalf("LoadEpisodeSpec failed: %v", err)
	}

	t.Run("Program", func(t *testing.T) {
		if spec.Program.Title == "" {
			t.Error("Program.Title must not be empty")
		}
	})

	t.Run("CornerAssets", func(t *testing.T) {
		if len(spec.Corners) == 0 {
			t.Fatal("Corners must not be empty")
		}
		corner0 := spec.Corners[0]
		if corner0.StartJingle == "" {
			t.Error("Corners[0].StartJingle must not be empty")
		}
		if len(spec.Corners) > 1 {
			corner1 := spec.Corners[1]
			if corner1.EndJingle == "" {
				t.Error("Corners[1].EndJingle must not be empty")
			}
			if corner1.BGM == "" {
				t.Error("Corners[1].BGM must not be empty")
			}
		}
	})

	t.Run("Corners", func(t *testing.T) {
		if len(spec.Corners) == 0 {
			t.Error("Corners must not be empty")
		}
		c := spec.Corners[0]
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
		for i := range spec.Corners {
			if spec.Corners[i].Source != nil {
				sourceCorner = &spec.Corners[i]
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
		if len(spec.Assets.Jingle) == 0 {
			t.Error("Assets.Jingle must not be empty")
		}
		if len(spec.Assets.SE) == 0 {
			t.Error("Assets.SE must not be empty")
		}
		if len(spec.Assets.BGM) == 0 {
			t.Error("Assets.BGM must not be empty")
		}
	})

	t.Run("Assets_PathResolution", func(t *testing.T) {
		base := "testdata"

		jingle, ok := spec.Assets.Jingle["opening"]
		if !ok {
			t.Fatal("Assets.Jingle[\"opening\"] not found")
		}
		if want := filepath.Join(base, "assets/jingle/opening.mp3"); jingle.File != want {
			t.Errorf("Jingle[\"opening\"].File: expected %q, got %q", want, jingle.File)
		}

		se, ok := spec.Assets.SE["chime"]
		if !ok {
			t.Fatal("Assets.SE[\"chime\"] not found")
		}
		if want := filepath.Join(base, "assets/se/chime.wav"); se.File != want {
			t.Errorf("SE[\"chime\"].File: expected %q, got %q", want, se.File)
		}

		bgm, ok := spec.Assets.BGM["talk_bgm"]
		if !ok {
			t.Fatal("Assets.BGM[\"talk_bgm\"] not found")
		}
		if want := filepath.Join(base, "assets/bgm/talk.mp3"); bgm.File != want {
			t.Errorf("BGM[\"talk_bgm\"].File: expected %q, got %q", want, bgm.File)
		}
	})
}

func TestLoadEpisodeSpec_AbsolutePaths(t *testing.T) {
	spec, err := config.LoadEpisodeSpec("testdata/episode_spec_abs.yaml")
	if err != nil {
		t.Fatalf("LoadEpisodeSpec failed: %v", err)
	}

	for name, entry := range spec.Assets.Jingle {
		if !filepath.IsAbs(entry.File) {
			t.Errorf("Jingle[%q].File should remain absolute, got %q", name, entry.File)
		}
	}
	for name, entry := range spec.Assets.SE {
		if !filepath.IsAbs(entry.File) {
			t.Errorf("SE[%q].File should remain absolute, got %q", name, entry.File)
		}
	}
	for name, entry := range spec.Assets.BGM {
		if !filepath.IsAbs(entry.File) {
			t.Errorf("BGM[%q].File should remain absolute, got %q", name, entry.File)
		}
	}
}

func TestLoadEpisodeSpec_MissingFile(t *testing.T) {
	_, err := config.LoadEpisodeSpec("testdata/nonexistent.yaml")
	if err == nil {
		t.Error("expected error for missing file")
	}
}

func TestValidateEpisodeSpecAssets_Valid(t *testing.T) {
	p := &config.EpisodeSpec{
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
	if err := config.ValidateEpisodeSpecAssets(p); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestValidateEpisodeSpecAssets_UnknownStartJingle(t *testing.T) {
	p := &config.EpisodeSpec{
		Corners: []config.CornerConfig{
			{Title: "C1", StartJingle: "nonexistent"},
		},
		Assets: config.AssetsConfig{
			Jingle: map[string]config.JingleEntry{},
			BGM:    map[string]config.BGMEntry{},
		},
	}
	if err := config.ValidateEpisodeSpecAssets(p); err == nil {
		t.Error("expected error for unknown start_jingle key")
	}
}

func TestValidateEpisodeSpecAssets_UnknownEndJingle(t *testing.T) {
	p := &config.EpisodeSpec{
		Corners: []config.CornerConfig{
			{Title: "C1", EndJingle: "nonexistent"},
		},
		Assets: config.AssetsConfig{
			Jingle: map[string]config.JingleEntry{},
			BGM:    map[string]config.BGMEntry{},
		},
	}
	if err := config.ValidateEpisodeSpecAssets(p); err == nil {
		t.Error("expected error for unknown end_jingle key")
	}
}

func TestValidateEpisodeSpecAssets_UnknownBGM(t *testing.T) {
	p := &config.EpisodeSpec{
		Corners: []config.CornerConfig{
			{Title: "C1", BGM: "nonexistent"},
		},
		Assets: config.AssetsConfig{
			Jingle: map[string]config.JingleEntry{},
			BGM:    map[string]config.BGMEntry{},
		},
	}
	if err := config.ValidateEpisodeSpecAssets(p); err == nil {
		t.Error("expected error for unknown bgm key")
	}
}

func TestValidateEpisodeSpecAssets_EmptyFields_NoError(t *testing.T) {
	p := &config.EpisodeSpec{
		Corners: []config.CornerConfig{
			{Title: "C1"},
		},
		Assets: config.AssetsConfig{
			Jingle: map[string]config.JingleEntry{},
			BGM:    map[string]config.BGMEntry{},
		},
	}
	if err := config.ValidateEpisodeSpecAssets(p); err != nil {
		t.Errorf("unexpected error for empty fields: %v", err)
	}
}

func TestProgramConfig_EffectiveSummaryLength(t *testing.T) {
	tests := []struct {
		name          string
		summaryLength int
		want          int
	}{
		{name: "unset returns default", summaryLength: 0, want: config.DefaultProgramSummaryLength},
		{name: "explicit value returned", summaryLength: 300, want: 300},
		{name: "minimum value 1", summaryLength: 1, want: 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := config.ProgramConfig{SummaryLength: tt.summaryLength}
			got := p.EffectiveSummaryLength()
			if got != tt.want {
				t.Errorf("EffectiveSummaryLength() = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestCornerConfig_EffectiveSummaryLength(t *testing.T) {
	tests := []struct {
		name          string
		summaryLength int
		want          int
	}{
		{name: "unset returns default", summaryLength: 0, want: config.DefaultCornerSummaryLength},
		{name: "explicit value returned", summaryLength: 200, want: 200},
		{name: "minimum value 1", summaryLength: 1, want: 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := config.CornerConfig{SummaryLength: tt.summaryLength}
			got := c.EffectiveSummaryLength()
			if got != tt.want {
				t.Errorf("EffectiveSummaryLength() = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestEpisodeSpec_CornerSummaryLength(t *testing.T) {
	p := &config.EpisodeSpec{
		Corners: []config.CornerConfig{
			{Title: "テックニュース", SummaryLength: 150},
			{Title: "AI特集", SummaryLength: 0},
		},
	}

	tests := []struct {
		name  string
		title string
		want  int
	}{
		{name: "known corner with explicit length", title: "テックニュース", want: 150},
		{name: "known corner with unset length falls back to default", title: "AI特集", want: config.DefaultCornerSummaryLength},
		{name: "unknown corner falls back to default", title: "存在しないコーナー", want: config.DefaultCornerSummaryLength},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := p.CornerSummaryLength(tt.title)
			if got != tt.want {
				t.Errorf("CornerSummaryLength(%q) = %d, want %d", tt.title, got, tt.want)
			}
		})
	}
}

func TestLoadEpisodeSpec_ValidateEpisodeSpecAssetsIntegration(t *testing.T) {
	spec, err := config.LoadEpisodeSpec("testdata/episode_spec.yaml")
	if err != nil {
		t.Fatalf("LoadEpisodeSpec failed: %v", err)
	}
	if err := config.ValidateEpisodeSpecAssets(spec); err != nil {
		t.Errorf("ValidateEpisodeSpecAssets failed on testdata spec: %v", err)
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

func TestLoadEpisodeSpec_CornerDirection(t *testing.T) {
	spec, err := config.LoadEpisodeSpec("testdata/episode_spec.yaml")
	if err != nil {
		t.Fatalf("LoadEpisodeSpec failed: %v", err)
	}

	var directionCorner *config.CornerConfig
	for i := range spec.Corners {
		if spec.Corners[i].Direction != "" {
			directionCorner = &spec.Corners[i]
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

func TestValidateEpisodeSpecCast_Valid(t *testing.T) {
	p := &config.EpisodeSpec{
		Casts: map[string]config.CastConfig{
			"zundamon": {Type: config.CastTypeRegular, Role: "司会"},
		},
		Corners: []config.CornerConfig{
			{Title: "opening", Cast: map[string]string{"zundamon": "ボケ担当"}},
		},
	}
	if err := config.ValidateEpisodeSpecCast(p); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestValidateEpisodeSpecCast_UndeclaredCastKey(t *testing.T) {
	p := &config.EpisodeSpec{
		Casts: map[string]config.CastConfig{
			"zundamon": {Type: config.CastTypeRegular, Role: "司会"},
		},
		Corners: []config.CornerConfig{
			{Title: "opening", Cast: map[string]string{"unknown_char": "司会"}},
		},
	}
	if err := config.ValidateEpisodeSpecCast(p); err == nil {
		t.Error("expected error for corner cast key not declared in casts")
	}
}

func TestEpisodeCondition_Matches(t *testing.T) {
	tests := []struct {
		name          string
		cond          config.EpisodeCondition
		episodeNumber int
		want          bool
	}{
		// episodes（明示リスト）
		{name: "episodes: 合致する回番号", cond: config.EpisodeCondition{Episodes: []int{3, 10}}, episodeNumber: 3, want: true},
		{name: "episodes: 合致しない回番号", cond: config.EpisodeCondition{Episodes: []int{3, 10}}, episodeNumber: 5, want: false},
		{name: "episodes: 空リスト（肯定条件なし→常に真）", cond: config.EpisodeCondition{Episodes: []int{}}, episodeNumber: 1, want: true},
		// every（周期）
		{name: "every: 倍数回に合致", cond: config.EpisodeCondition{Every: 5}, episodeNumber: 10, want: true},
		{name: "every: 倍数でない回は合致しない", cond: config.EpisodeCondition{Every: 5}, episodeNumber: 7, want: false},
		{name: "every: 0 は未指定（肯定条件なし→常に真）", cond: config.EpisodeCondition{Every: 0}, episodeNumber: 5, want: true},
		// 論理和（両方指定）
		{name: "episodes+every: どちらかに合致すれば true", cond: config.EpisodeCondition{Episodes: []int{3}, Every: 5}, episodeNumber: 5, want: true},
		{name: "episodes+every: episodes のみ合致", cond: config.EpisodeCondition{Episodes: []int{3}, Every: 5}, episodeNumber: 3, want: true},
		{name: "episodes+every: どちらにも合致しない", cond: config.EpisodeCondition{Episodes: []int{3}, Every: 5}, episodeNumber: 2, want: false},
		// 回番号不明（0 以下）
		{name: "episodeNumber=0 は false", cond: config.EpisodeCondition{Episodes: []int{3}, Every: 5}, episodeNumber: 0, want: false},
		{name: "episodeNumber<0 は false", cond: config.EpisodeCondition{Episodes: []int{1}}, episodeNumber: -1, want: false},
		// not（否定）
		{name: "not: 否定条件に合致する回は false", cond: config.EpisodeCondition{Not: &config.EpisodeCondition{Every: 5}}, episodeNumber: 5, want: false},
		{name: "not: 否定条件に合致しない回は true（肯定条件なし→常に真）", cond: config.EpisodeCondition{Not: &config.EpisodeCondition{Every: 5}}, episodeNumber: 7, want: true},
		{name: "not: 肯定条件なしで否定に合致しない回は true", cond: config.EpisodeCondition{Not: &config.EpisodeCondition{Episodes: []int{3}}}, episodeNumber: 1, want: true},
		// every + not（組み合わせ）
		{name: "every+not: 倍数かつ not に非合致 → true", cond: config.EpisodeCondition{Every: 2, Not: &config.EpisodeCondition{Episodes: []int{6}}}, episodeNumber: 4, want: true},
		{name: "every+not: 倍数だが not に合致 → false", cond: config.EpisodeCondition{Every: 2, Not: &config.EpisodeCondition{Episodes: []int{6}}}, episodeNumber: 6, want: false},
		{name: "every+not: 倍数でない → false", cond: config.EpisodeCondition{Every: 2, Not: &config.EpisodeCondition{Episodes: []int{6}}}, episodeNumber: 3, want: false},
		// 肯定条件なし（not 単独）で episodeNumber <= 0 は false
		{name: "not 単独で episodeNumber=0 は false", cond: config.EpisodeCondition{Not: &config.EpisodeCondition{Every: 5}}, episodeNumber: 0, want: false},
		// offset（剰余）
		{name: "every+offset: 余り1に合致", cond: config.EpisodeCondition{Every: 3, Offset: 1}, episodeNumber: 1, want: true},
		{name: "every+offset: 余り1に合致(4回目)", cond: config.EpisodeCondition{Every: 3, Offset: 1}, episodeNumber: 4, want: true},
		{name: "every+offset: 余り1に合致(7回目)", cond: config.EpisodeCondition{Every: 3, Offset: 1}, episodeNumber: 7, want: true},
		{name: "every+offset: 余り2に合致", cond: config.EpisodeCondition{Every: 3, Offset: 2}, episodeNumber: 2, want: true},
		{name: "every+offset: 余り0に合致（offset=0 は従来の倍数回）", cond: config.EpisodeCondition{Every: 3, Offset: 0}, episodeNumber: 3, want: true},
		{name: "every+offset: 合致しない（余り不一致）", cond: config.EpisodeCondition{Every: 3, Offset: 1}, episodeNumber: 3, want: false},
		{name: "offset未指定（0）は every の倍数回（後方互換）", cond: config.EpisodeCondition{Every: 5}, episodeNumber: 10, want: true},
		{name: "offset未指定（0）で倍数でない回は false（後方互換）", cond: config.EpisodeCondition{Every: 5}, episodeNumber: 7, want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.cond.Matches(tt.episodeNumber)
			if got != tt.want {
				t.Errorf("EpisodeCondition%+v.Matches(%d) = %v, want %v", tt.cond, tt.episodeNumber, got, tt.want)
			}
		})
	}
}

func TestValidateEpisodeCondition_Offset(t *testing.T) {
	tests := []struct {
		name    string
		casts   map[string]config.CastConfig
		wantErr bool
	}{
		{
			name: "every+offset 有効（3者ローテ offset=0）",
			casts: map[string]config.CastConfig{
				"zundamon": {Type: config.CastTypeGuest, Role: "ゲスト", Condition: &config.EpisodeCondition{Every: 3, Offset: 0}},
			},
			wantErr: false,
		},
		{
			name: "every+offset 有効（3者ローテ offset=1）",
			casts: map[string]config.CastConfig{
				"zundamon": {Type: config.CastTypeGuest, Role: "ゲスト", Condition: &config.EpisodeCondition{Every: 3, Offset: 1}},
			},
			wantErr: false,
		},
		{
			name: "every+offset 有効（3者ローテ offset=2）",
			casts: map[string]config.CastConfig{
				"zundamon": {Type: config.CastTypeGuest, Role: "ゲスト", Condition: &config.EpisodeCondition{Every: 3, Offset: 2}},
			},
			wantErr: false,
		},
		{
			name: "offset が負数",
			casts: map[string]config.CastConfig{
				"zundamon": {Type: config.CastTypeGuest, Role: "ゲスト", Condition: &config.EpisodeCondition{Every: 3, Offset: -1}},
			},
			wantErr: true,
		},
		{
			name: "offset > 0 かつ every == 0",
			casts: map[string]config.CastConfig{
				"zundamon": {Type: config.CastTypeGuest, Role: "ゲスト", Condition: &config.EpisodeCondition{Offset: 1}},
			},
			wantErr: true,
		},
		{
			name: "offset >= every",
			casts: map[string]config.CastConfig{
				"zundamon": {Type: config.CastTypeGuest, Role: "ゲスト", Condition: &config.EpisodeCondition{Every: 3, Offset: 3}},
			},
			wantErr: true,
		},
		{
			name: "not 内の offset が負数",
			casts: map[string]config.CastConfig{
				"zundamon": {Type: config.CastTypeGuest, Role: "ゲスト", Condition: &config.EpisodeCondition{Not: &config.EpisodeCondition{Every: 3, Offset: -1}}},
			},
			wantErr: true,
		},
		{
			name: "not 内の offset >= every",
			casts: map[string]config.CastConfig{
				"zundamon": {Type: config.CastTypeGuest, Role: "ゲスト", Condition: &config.EpisodeCondition{Not: &config.EpisodeCondition{Every: 3, Offset: 3}}},
			},
			wantErr: true,
		},
	}
	chars := map[string]config.CharacterConfig{
		"zundamon": {Name: "ずんだもん"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &config.EpisodeSpec{Casts: tt.casts}
			err := config.ValidateEpisodeSpecCasts(p, chars)
			if tt.wantErr && err == nil {
				t.Errorf("expected error, got nil")
			} else if !tt.wantErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestValidateEpisodeSpecCasts_Valid(t *testing.T) {
	chars := map[string]config.CharacterConfig{
		"zundamon": {Name: "ずんだもん"},
		"metan":    {Name: "めたん"},
	}
	tests := []struct {
		name  string
		casts map[string]config.CastConfig
	}{
		{
			name: "regular condition 省略（毎回出演）",
			casts: map[string]config.CastConfig{
				"zundamon": {Type: config.CastTypeRegular, Role: "MC"},
			},
		},
		{
			name: "regular condition あり（お休み条件）",
			casts: map[string]config.CastConfig{
				"zundamon": {Type: config.CastTypeRegular, Role: "MC", Condition: &config.EpisodeCondition{Not: &config.EpisodeCondition{Episodes: []int{5}}}},
			},
		},
		{
			name: "guest episodes のみ指定",
			casts: map[string]config.CastConfig{
				"metan": {Type: config.CastTypeGuest, Role: "ゲスト", Condition: &config.EpisodeCondition{Episodes: []int{3, 10}}},
			},
		},
		{
			name: "guest every のみ指定",
			casts: map[string]config.CastConfig{
				"metan": {Type: config.CastTypeGuest, Role: "解説ゲスト", Condition: &config.EpisodeCondition{Every: 5}},
			},
		},
		{
			name:  "casts が空（省略時）",
			casts: nil,
		},
		{
			name: "regular と guest の混在",
			casts: map[string]config.CastConfig{
				"zundamon": {Type: config.CastTypeRegular, Role: "MC"},
				"metan":    {Type: config.CastTypeGuest, Role: "ゲスト", Condition: &config.EpisodeCondition{Episodes: []int{3}}},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &config.EpisodeSpec{Casts: tt.casts}
			if err := config.ValidateEpisodeSpecCasts(p, chars); err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestValidateEpisodeSpecCasts_Error(t *testing.T) {
	chars := map[string]config.CharacterConfig{
		"zundamon": {Name: "ずんだもん"},
	}
	tests := []struct {
		name  string
		casts map[string]config.CastConfig
	}{
		{
			name: "存在しないキャラID",
			casts: map[string]config.CastConfig{
				"unknown_char": {Type: config.CastTypeGuest, Role: "ゲスト", Condition: &config.EpisodeCondition{Episodes: []int{3}}},
			},
		},
		{
			name: "type が不正",
			casts: map[string]config.CastConfig{
				"zundamon": {Type: "invalid", Role: "MC"},
			},
		},
		{
			name: "guest に condition がない",
			casts: map[string]config.CastConfig{
				"zundamon": {Type: config.CastTypeGuest, Role: "ゲスト"},
			},
		},
		{
			name: "condition が空（episodes も every も not も未設定）",
			casts: map[string]config.CastConfig{
				"zundamon": {Type: config.CastTypeGuest, Role: "ゲスト", Condition: &config.EpisodeCondition{}},
			},
		},
		{
			name: "not の中身が空",
			casts: map[string]config.CastConfig{
				"zundamon": {Type: config.CastTypeGuest, Role: "ゲスト", Condition: &config.EpisodeCondition{Not: &config.EpisodeCondition{}}},
			},
		},
		{
			name: "episodes の値が 0",
			casts: map[string]config.CastConfig{
				"zundamon": {Type: config.CastTypeGuest, Role: "ゲスト", Condition: &config.EpisodeCondition{Episodes: []int{0}}},
			},
		},
		{
			name: "every が負数",
			casts: map[string]config.CastConfig{
				"zundamon": {Type: config.CastTypeGuest, Role: "ゲスト", Condition: &config.EpisodeCondition{Every: -1}},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &config.EpisodeSpec{Casts: tt.casts}
			if err := config.ValidateEpisodeSpecCasts(p, chars); err == nil {
				t.Errorf("expected error for %q, got nil", tt.name)
			}
		})
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

func TestLoadEpisodeSpec_ProgramID(t *testing.T) {
	spec, err := config.LoadEpisodeSpec("testdata/episode_spec_with_id.yaml")
	if err != nil {
		t.Fatalf("LoadEpisodeSpec failed: %v", err)
	}
	if spec.Program.ID != "test-program" {
		t.Errorf("Program.ID: got %q, want %q", spec.Program.ID, "test-program")
	}
}

func TestLoadEpisodeSpec_ProgramIDEmpty_NoError(t *testing.T) {
	spec, err := config.LoadEpisodeSpec("testdata/episode_spec.yaml")
	if err != nil {
		t.Fatalf("LoadEpisodeSpec failed: %v", err)
	}
	if spec.Program.ID != "" {
		t.Errorf("Program.ID: expected empty, got %q", spec.Program.ID)
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

func TestLoadEpisodeSpecStrict_UnknownKeyErrors(t *testing.T) {
	_, err := config.LoadEpisodeSpecStrict("testdata/episode_spec_unknown_key.yaml")
	if err == nil {
		t.Error("expected error for unknown key in strict mode")
	}
}

func TestLoadEpisodeSpecStrict_ValidYAML_Success(t *testing.T) {
	_, err := config.LoadEpisodeSpecStrict("testdata/episode_spec.yaml")
	if err != nil {
		t.Errorf("unexpected error for valid spec in strict mode: %v", err)
	}
}

func TestLoadEpisodeSpec_UnknownKey_NoError(t *testing.T) {
	_, err := config.LoadEpisodeSpec("testdata/episode_spec_unknown_key.yaml")
	if err != nil {
		t.Errorf("LoadEpisodeSpec should not error on unknown key (non-strict): %v", err)
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

func TestLoadEpisodeSpec_AssetsDescription(t *testing.T) {
	spec, err := config.LoadEpisodeSpec("testdata/episode_spec.yaml")
	if err != nil {
		t.Fatalf("LoadEpisodeSpec failed: %v", err)
	}

	se, ok := spec.Assets.SE["chime"]
	if !ok {
		t.Fatal("Assets.SE[\"chime\"] not found")
	}
	if se.Description == "" {
		t.Error("SE[\"chime\"].Description must not be empty (testdata should include description)")
	}

	bgm, ok := spec.Assets.BGM["talk_bgm"]
	if !ok {
		t.Fatal("Assets.BGM[\"talk_bgm\"] not found")
	}
	if bgm.Description == "" {
		t.Error("BGM[\"talk_bgm\"].Description must not be empty (testdata should include description)")
	}

	jingle, ok := spec.Assets.Jingle["opening"]
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

func TestLoadConfig_SlackBotTokenEnv(t *testing.T) {
	cfg, err := config.LoadConfig("testdata/config_with_slack.yaml")
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}
	if cfg.Slack.BotTokenEnv != "SLACK_BOT_TOKEN" {
		t.Errorf("Slack.BotTokenEnv = %q, want %q", cfg.Slack.BotTokenEnv, "SLACK_BOT_TOKEN")
	}
}

func TestLoadConfig_SlackAbsent_ZeroValue(t *testing.T) {
	cfg, err := config.LoadConfig("testdata/config.yaml")
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}
	if cfg.Slack.BotTokenEnv != "" {
		t.Errorf("Slack.BotTokenEnv should be empty when not set, got %q", cfg.Slack.BotTokenEnv)
	}
}

func TestLoadConfigStrict_WithSlack_Success(t *testing.T) {
	_, err := config.LoadConfigStrict("testdata/config_with_slack.yaml")
	if err != nil {
		t.Errorf("unexpected error for config with slack in strict mode: %v", err)
	}
}

func TestLoadEpisodeSpec_AssetsFiles_MultipleFileMerge(t *testing.T) {
	spec, err := config.LoadEpisodeSpec("testdata/episode_spec_multi_assets.yaml")
	if err != nil {
		t.Fatalf("LoadEpisodeSpec failed: %v", err)
	}
	// opening should be overridden by second file (assets_override.yaml)
	jingle, ok := spec.Assets.Jingle["opening"]
	if !ok {
		t.Fatal("Assets.Jingle[\"opening\"] not found")
	}
	want := filepath.Join("testdata", "assets", "jingle", "opening_v2.mp3")
	if jingle.File != want {
		t.Errorf("Jingle[\"opening\"].File: expected %q (from override), got %q", want, jingle.File)
	}
	// ending should come from first file (assets.yaml)
	if _, ok := spec.Assets.Jingle["ending"]; !ok {
		t.Error("Assets.Jingle[\"ending\"] should come from first assets file")
	}
}

func TestLoadEpisodeSpec_AssetsFiles_PathRelativeToAssetFile(t *testing.T) {
	spec, err := config.LoadEpisodeSpec("testdata/episode_spec_subdir_assets.yaml")
	if err != nil {
		t.Fatalf("LoadEpisodeSpec failed: %v", err)
	}
	jingle, ok := spec.Assets.Jingle["opening"]
	if !ok {
		t.Fatal("Assets.Jingle[\"opening\"] not found")
	}
	// path resolved relative to subdir/, not testdata/
	want := filepath.Join("testdata", "subdir", "jingle", "opening.mp3")
	if jingle.File != want {
		t.Errorf("Jingle[\"opening\"].File: expected %q, got %q", want, jingle.File)
	}
}

func TestLoadEpisodeSpec_AssetsFiles_Empty_NoError(t *testing.T) {
	spec, err := config.LoadEpisodeSpec("testdata/episode_spec_no_assets.yaml")
	if err != nil {
		t.Fatalf("LoadEpisodeSpec should not error when assets_files is empty: %v", err)
	}
	if len(spec.Assets.Jingle) != 0 || len(spec.Assets.SE) != 0 || len(spec.Assets.BGM) != 0 {
		t.Error("Assets should be empty when assets_files is not specified")
	}
}

func TestLoadEpisodeSpec_AssetsFiles_MissingFile_Error(t *testing.T) {
	_, err := config.LoadEpisodeSpec("testdata/episode_spec_missing_assets_file.yaml")
	if err == nil {
		t.Error("expected error when assets_files references non-existent file")
	}
}

func TestLoadEpisodeSpecStrict_LegacyAssets_Error(t *testing.T) {
	_, err := config.LoadEpisodeSpecStrict("testdata/episode_spec_with_legacy_assets.yaml")
	if err == nil {
		t.Error("expected error when spec has old-style assets: in strict mode")
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

func TestSEEntry_EffectiveOverlay(t *testing.T) {
	cases := []struct {
		ptr  *bool
		want bool
	}{
		{nil, false},
		{boolPtr(false), false},
		{boolPtr(true), true},
	}
	for _, c := range cases {
		e := config.SEEntry{Overlay: c.ptr}
		if got := e.EffectiveOverlay(); got != c.want {
			t.Errorf("ptr=%v: got %v, want %v", c.ptr, got, c.want)
		}
	}
}

func float64Ptr(v float64) *float64 { return &v }

func TestBGMEntry_EffectiveFadeIn(t *testing.T) {
	cases := []struct {
		name string
		ptr  *float64
		want float64
	}{
		{"nil defaults to DefaultBGMFadeSec", nil, config.DefaultBGMFadeSec},
		{"zero disables fade-in", float64Ptr(0.0), 0.0},
		{"positive value used as-is", float64Ptr(2.0), 2.0},
		{"negative clamped to zero", float64Ptr(-1.0), 0.0},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			e := config.BGMEntry{FadeIn: c.ptr}
			if got := e.EffectiveFadeIn(); got != c.want {
				t.Errorf("ptr=%v: got %v, want %v", c.ptr, got, c.want)
			}
		})
	}
}

func TestBGMEntry_EffectiveFadeOut(t *testing.T) {
	cases := []struct {
		name string
		ptr  *float64
		want float64
	}{
		{"nil defaults to DefaultBGMFadeSec", nil, config.DefaultBGMFadeSec},
		{"zero disables fade-out", float64Ptr(0.0), 0.0},
		{"positive value used as-is", float64Ptr(2.0), 2.0},
		{"negative clamped to zero", float64Ptr(-1.0), 0.0},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			e := config.BGMEntry{FadeOut: c.ptr}
			if got := e.EffectiveFadeOut(); got != c.want {
				t.Errorf("ptr=%v: got %v, want %v", c.ptr, got, c.want)
			}
		})
	}
}

// --- LoadAssetsFileStrict ---

func TestLoadAssetsFileStrict_ValidYAML_Success(t *testing.T) {
	_, err := config.LoadAssetsFileStrict("testdata/assets.yaml")
	if err != nil {
		t.Errorf("unexpected error for valid assets.yaml: %v", err)
	}
}

func TestLoadAssetsFileStrict_UnknownKeyErrors(t *testing.T) {
	_, err := config.LoadAssetsFileStrict("testdata/assets_typo.yaml")
	if err == nil {
		t.Error("expected error for unknown key in strict mode, got nil")
	}
}

// --- LoadEpisodeSpecStrict: strict propagation to assets ---

func TestLoadEpisodeSpecStrict_AssetsTypo_Error(t *testing.T) {
	_, err := config.LoadEpisodeSpecStrict("testdata/episode_spec_with_typo_assets.yaml")
	if err == nil {
		t.Error("expected error when assets_files contains typo and strict mode is on, got nil")
	}
}

func TestLoadEpisodeSpec_AssetsTypo_NoError(t *testing.T) {
	_, err := config.LoadEpisodeSpec("testdata/episode_spec_with_typo_assets.yaml")
	if err != nil {
		t.Errorf("LoadEpisodeSpec (non-strict) should not error on assets typo: %v", err)
	}
}

// --- ValidateAssetsConfig ---

func TestValidateAssetsConfig_Valid_Success(t *testing.T) {
	dir := t.TempDir()
	jingleFile := filepath.Join(dir, "opening.mp3")
	seFile := filepath.Join(dir, "chime.wav")
	bgmFile := filepath.Join(dir, "talk.mp3")
	for _, f := range []string{jingleFile, seFile, bgmFile} {
		if err := os.WriteFile(f, []byte{}, 0600); err != nil {
			t.Fatalf("setup: %v", err)
		}
	}

	assets := &config.AssetsConfig{
		Jingle: map[string]config.JingleEntry{
			"opening": {File: jingleFile, FadeIn: 0.5, FadeOut: 0.5},
		},
		SE: map[string]config.SEEntry{
			"chime": {File: seFile, Volume: 0.8},
		},
		BGM: map[string]config.BGMEntry{
			"talk": {File: bgmFile, Volume: 0.3, DuckRatio: 8},
		},
	}
	if err := config.ValidateAssetsConfig(assets); err != nil {
		t.Errorf("unexpected error for valid assets: %v", err)
	}
}

func TestValidateAssetsConfig_FileMissing_Error(t *testing.T) {
	assets := &config.AssetsConfig{
		Jingle: map[string]config.JingleEntry{
			"opening": {File: "/nonexistent/opening.mp3", FadeIn: 0, FadeOut: 0},
		},
	}
	if err := config.ValidateAssetsConfig(assets); err == nil {
		t.Error("expected error for missing file, got nil")
	}
}

func TestValidateAssetsConfig_EmptyFileField_Error(t *testing.T) {
	assets := &config.AssetsConfig{
		SE: map[string]config.SEEntry{
			"chime": {File: "", Volume: 0.8},
		},
	}
	if err := config.ValidateAssetsConfig(assets); err == nil {
		t.Error("expected error for empty file field, got nil")
	}
}

func TestValidateAssetsConfig_InvalidField_Error(t *testing.T) {
	neg1 := -1.0
	neg05 := -0.5
	cases := []struct {
		name   string
		assets func(f string) *config.AssetsConfig
	}{
		{
			name: "jingle fade_in negative",
			assets: func(f string) *config.AssetsConfig {
				return &config.AssetsConfig{Jingle: map[string]config.JingleEntry{"j": {File: f, FadeIn: -1.0, FadeOut: 0}}}
			},
		},
		{
			name: "jingle fade_out negative",
			assets: func(f string) *config.AssetsConfig {
				return &config.AssetsConfig{Jingle: map[string]config.JingleEntry{"j": {File: f, FadeIn: 0, FadeOut: -0.5}}}
			},
		},
		{
			name: "se volume negative",
			assets: func(f string) *config.AssetsConfig {
				return &config.AssetsConfig{SE: map[string]config.SEEntry{"chime": {File: f, Volume: -0.1}}}
			},
		},
		{
			name: "bgm volume negative",
			assets: func(f string) *config.AssetsConfig {
				return &config.AssetsConfig{BGM: map[string]config.BGMEntry{"bgm": {File: f, Volume: -1, DuckRatio: 8}}}
			},
		},
		{
			name: "bgm duck_ratio less than 1",
			assets: func(f string) *config.AssetsConfig {
				return &config.AssetsConfig{BGM: map[string]config.BGMEntry{"bgm": {File: f, Volume: 0.3, DuckRatio: 0}}}
			},
		},
		{
			name: "bgm fade_in negative",
			assets: func(f string) *config.AssetsConfig {
				return &config.AssetsConfig{BGM: map[string]config.BGMEntry{"bgm": {File: f, Volume: 0.3, DuckRatio: 8, FadeIn: &neg1}}}
			},
		},
		{
			name: "bgm fade_out negative",
			assets: func(f string) *config.AssetsConfig {
				return &config.AssetsConfig{BGM: map[string]config.BGMEntry{"bgm": {File: f, Volume: 0.3, DuckRatio: 8, FadeOut: &neg05}}}
			},
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			dir := t.TempDir()
			f := filepath.Join(dir, "stub.mp3")
			if err := os.WriteFile(f, []byte{}, 0600); err != nil {
				t.Fatalf("setup: %v", err)
			}
			if err := config.ValidateAssetsConfig(c.assets(f)); err == nil {
				t.Errorf("expected error for %q, got nil", c.name)
			}
		})
	}
}
