package config_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/canpok1/vox-radio/internal/config"
	"github.com/canpok1/vox-radio/internal/testutil"
)

func TestDurationSecToTargetChars(t *testing.T) {
	tests := []struct {
		sec            int
		charsPerMinute int
		want           int
	}{
		// デフォルト値 420/分（= 旧 7文字/秒）での従来値一致
		{sec: 0, charsPerMinute: 420, want: 0},
		{sec: 14, charsPerMinute: 420, want: 98},
		{sec: 30, charsPerMinute: 420, want: 210},
		// 任意値での確認
		{sec: 60, charsPerMinute: 300, want: 300},
		{sec: 120, charsPerMinute: 300, want: 600},
	}
	for _, tt := range tests {
		got := config.DurationSecToTargetChars(tt.sec, tt.charsPerMinute)
		if got != tt.want {
			t.Errorf("DurationSecToTargetChars(%d, %d) = %d, want %d", tt.sec, tt.charsPerMinute, got, tt.want)
		}
	}
}

func TestProgramConfig_EffectiveCharsPerMinute(t *testing.T) {
	tests := []struct {
		name           string
		charsPerMinute int
		want           int
	}{
		{name: "unset (0) returns default", charsPerMinute: 0, want: config.DefaultCharsPerMinute},
		{name: "negative returns default", charsPerMinute: -1, want: config.DefaultCharsPerMinute},
		{name: "explicit value returned", charsPerMinute: 300, want: 300},
		{name: "explicit default value returned", charsPerMinute: 420, want: 420},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := config.ProgramConfig{CharsPerMinute: tt.charsPerMinute}
			got := p.EffectiveCharsPerMinute()
			if got != tt.want {
				t.Errorf("EffectiveCharsPerMinute() = %d, want %d", got, tt.want)
			}
		})
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

	t.Run("ProgramDirection", func(t *testing.T) {
		if spec.Program.Direction == "" {
			t.Error("Program.Direction must not be empty")
		}
	})

	t.Run("ProgramScriptNote", func(t *testing.T) {
		if spec.Program.ScriptNote == "" {
			t.Error("Program.ScriptNote must not be empty")
		}
	})

	t.Run("CornerScriptNote", func(t *testing.T) {
		if len(spec.Corners) == 0 {
			t.Fatal("Corners must not be empty")
		}
		if spec.Corners[0].ScriptNote == "" {
			t.Error("Corners[0].ScriptNote must not be empty")
		}
	})

	t.Run("CornerAssets", func(t *testing.T) {
		checkCornerAssets(t, spec)
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

func checkCornerAssets(t *testing.T, spec *config.EpisodeSpec) {
	t.Helper()
	if len(spec.Corners) == 0 {
		t.Fatal("Corners must not be empty")
	}
	corner0 := spec.Corners[0]
	if corner0.StartAudio == nil {
		t.Error("Corners[0].StartAudio must not be nil")
	} else {
		if corner0.StartAudio.Type != "jingle" {
			t.Errorf("Corners[0].StartAudio.Type: got %q, want jingle", corner0.StartAudio.Type)
		}
		if corner0.StartAudio.ID != "opening" {
			t.Errorf("Corners[0].StartAudio.ID: got %q, want opening", corner0.StartAudio.ID)
		}
	}
	if len(spec.Corners) <= 1 {
		return
	}
	corner1 := spec.Corners[1]
	if corner1.EndAudio == nil {
		t.Error("Corners[1].EndAudio must not be nil")
	} else {
		if corner1.EndAudio.Type != "jingle" {
			t.Errorf("Corners[1].EndAudio.Type: got %q, want jingle", corner1.EndAudio.Type)
		}
		if corner1.EndAudio.ID != "ending" {
			t.Errorf("Corners[1].EndAudio.ID: got %q, want ending", corner1.EndAudio.ID)
		}
	}
	if corner1.BGM == nil || *corner1.BGM == "" {
		t.Error("Corners[1].BGM must not be empty")
	}
}

func TestValidateEpisodeSpecAssets(t *testing.T) {
	tests := []struct {
		name    string
		spec    *config.EpisodeSpec
		wantErr bool
	}{
		{
			name: "valid start_audio jingle",
			spec: &config.EpisodeSpec{
				Corners: []config.CornerConfig{
					{Title: "C1", StartAudio: &config.AudioRef{Type: "jingle", ID: "opening"}, BGM: testutil.Ptr("talk_bgm")},
				},
				Assets: config.AssetsConfig{
					Jingle: map[string]config.JingleEntry{"opening": {File: "opening.mp3"}},
					BGM:    map[string]config.BGMEntry{"talk_bgm": {File: "bgm.mp3"}},
				},
			},
		},
		{
			name: "valid end_audio se",
			spec: &config.EpisodeSpec{
				Corners: []config.CornerConfig{
					{Title: "C1", EndAudio: &config.AudioRef{Type: "se", ID: "chime"}},
				},
				Assets: config.AssetsConfig{
					Jingle: map[string]config.JingleEntry{},
					SE:     map[string]config.SEEntry{"chime": {File: "chime.wav"}},
					BGM:    map[string]config.BGMEntry{},
				},
			},
		},
		{
			name: "empty fields",
			spec: &config.EpisodeSpec{
				Corners: []config.CornerConfig{{Title: "C1"}},
				Assets: config.AssetsConfig{
					Jingle: map[string]config.JingleEntry{},
					BGM:    map[string]config.BGMEntry{},
				},
			},
		},
		{
			name: "start_audio unknown jingle id",
			spec: &config.EpisodeSpec{
				Corners: []config.CornerConfig{
					{Title: "C1", StartAudio: &config.AudioRef{Type: "jingle", ID: "nonexistent"}},
				},
				Assets: config.AssetsConfig{
					Jingle: map[string]config.JingleEntry{},
					BGM:    map[string]config.BGMEntry{},
				},
			},
			wantErr: true,
		},
		{
			name: "end_audio unknown jingle id",
			spec: &config.EpisodeSpec{
				Corners: []config.CornerConfig{
					{Title: "C1", EndAudio: &config.AudioRef{Type: "jingle", ID: "nonexistent"}},
				},
				Assets: config.AssetsConfig{
					Jingle: map[string]config.JingleEntry{},
					BGM:    map[string]config.BGMEntry{},
				},
			},
			wantErr: true,
		},
		{
			name: "start_audio unknown se id",
			spec: &config.EpisodeSpec{
				Corners: []config.CornerConfig{
					{Title: "C1", StartAudio: &config.AudioRef{Type: "se", ID: "nonexistent"}},
				},
				Assets: config.AssetsConfig{
					Jingle: map[string]config.JingleEntry{},
					SE:     map[string]config.SEEntry{},
					BGM:    map[string]config.BGMEntry{},
				},
			},
			wantErr: true,
		},
		{
			name: "start_audio invalid type",
			spec: &config.EpisodeSpec{
				Corners: []config.CornerConfig{
					{Title: "C1", StartAudio: &config.AudioRef{Type: "bgm", ID: "something"}},
				},
				Assets: config.AssetsConfig{
					Jingle: map[string]config.JingleEntry{},
					BGM:    map[string]config.BGMEntry{},
				},
			},
			wantErr: true,
		},
		{
			name: "unknown bgm",
			spec: &config.EpisodeSpec{
				Corners: []config.CornerConfig{{Title: "C1", BGM: testutil.Ptr("nonexistent")}},
				Assets: config.AssetsConfig{
					Jingle: map[string]config.JingleEntry{},
					BGM:    map[string]config.BGMEntry{},
				},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.spec.ValidateAssets()
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateAssets() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestProgramConfig_EffectiveTimezone(t *testing.T) {
	tests := []struct {
		name   string
		cfg    config.ProgramConfig
		wantTZ string
	}{
		{"default (empty)", config.ProgramConfig{}, config.DefaultProgramTimezone},
		{"custom", config.ProgramConfig{Timezone: "America/New_York"}, "America/New_York"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.cfg.EffectiveTimezone()
			if got != tt.wantTZ {
				t.Errorf("EffectiveTimezone() = %q, want %q", got, tt.wantTZ)
			}
		})
	}
}

func TestProgramConfig_Location(t *testing.T) {
	t.Run("default loads Asia/Tokyo", func(t *testing.T) {
		cfg := config.ProgramConfig{}
		loc, err := cfg.Location()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		want, _ := time.LoadLocation("Asia/Tokyo")
		if loc.String() != want.String() {
			t.Errorf("Location() = %q, want %q", loc.String(), want.String())
		}
	})
	t.Run("custom timezone loads correctly", func(t *testing.T) {
		cfg := config.ProgramConfig{Timezone: "America/New_York"}
		loc, err := cfg.Location()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if loc.String() != "America/New_York" {
			t.Errorf("Location() = %q, want %q", loc.String(), "America/New_York")
		}
	})
	t.Run("invalid timezone returns error", func(t *testing.T) {
		cfg := config.ProgramConfig{Timezone: "Invalid/Timezone"}
		_, err := cfg.Location()
		if err == nil {
			t.Error("expected error for invalid timezone, got nil")
		}
	})
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
	if err := spec.ValidateAssets(); err != nil {
		t.Errorf("ValidateAssets failed on testdata spec: %v", err)
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

func TestVoicevoxConfig_EffectiveURL(t *testing.T) {
	// t.Setenv を使うため t.Parallel() は併用しない。
	tests := []struct {
		name   string
		env    string // setEnv が true のときに環境変数へ設定する値
		setEnv bool   // false なら環境変数を未設定にして検証する
		url    string
		want   string
	}{
		{
			name:   "環境変数が設定値・デフォルトより優先される",
			env:    "http://voicevox:50021",
			setEnv: true,
			url:    "http://localhost:50021",
			want:   "http://voicevox:50021",
		},
		{
			name: "環境変数なしなら設定値を使う",
			url:  "http://example.com:50021",
			want: "http://example.com:50021",
		},
		{
			name: "環境変数も設定値もなければデフォルトを使う",
			want: config.DefaultVoicevoxURL,
		},
		{
			name:   "環境変数が空文字なら設定値を使う",
			env:    "",
			setEnv: true,
			url:    "http://example.com:50021",
			want:   "http://example.com:50021",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setEnv {
				t.Setenv(config.VoicevoxURLEnv, tt.env)
			} else {
				// 他テストの影響を排除するため明示的に未設定化する。
				t.Setenv(config.VoicevoxURLEnv, "")
				if err := os.Unsetenv(config.VoicevoxURLEnv); err != nil {
					t.Fatalf("Unsetenv failed: %v", err)
				}
			}
			c := config.VoicevoxConfig{URL: tt.url}
			if got := c.EffectiveURL(); got != tt.want {
				t.Errorf("EffectiveURL() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestSlackConfig_EffectiveAPIURL(t *testing.T) {
	// t.Setenv を使うため t.Parallel() は併用しない。
	tests := []struct {
		name   string
		env    string // setEnv が true のときに環境変数へ設定する値
		setEnv bool   // false なら環境変数を未設定にして検証する
		want   string
	}{
		{
			name:   "環境変数の値を返す（末尾スラッシュを補う）",
			env:    "http://127.0.0.1:8080/api",
			setEnv: true,
			want:   "http://127.0.0.1:8080/api/",
		},
		{
			name:   "末尾スラッシュ付きはそのまま返す",
			env:    "http://127.0.0.1:8080/api/",
			setEnv: true,
			want:   "http://127.0.0.1:8080/api/",
		},
		{
			name: "環境変数なしなら空文字（slack-go デフォルトを使う）",
			want: "",
		},
		{
			name:   "環境変数が空文字なら空文字",
			env:    "",
			setEnv: true,
			want:   "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setEnv {
				t.Setenv(config.SlackAPIURLEnv, tt.env)
			} else {
				// 他テストの影響を排除するため明示的に未設定化する。
				t.Setenv(config.SlackAPIURLEnv, "")
				if err := os.Unsetenv(config.SlackAPIURLEnv); err != nil {
					t.Fatalf("Unsetenv failed: %v", err)
				}
			}
			c := config.SlackConfig{}
			if got := c.EffectiveAPIURL(); got != tt.want {
				t.Errorf("EffectiveAPIURL() = %q, want %q", got, tt.want)
			}
		})
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
	if err := p.ValidateCast(); err != nil {
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
	if err := p.ValidateCast(); err == nil {
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
			err := p.ValidateCasts(chars)
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
			if err := p.ValidateCasts(chars); err != nil {
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
			if err := p.ValidateCasts(chars); err == nil {
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

func TestValidateEpisodeSpecProgram_EmptyID_Error(t *testing.T) {
	p := &config.EpisodeSpec{Program: config.ProgramConfig{Title: "t", Description: "d"}}
	if err := p.ValidateProgram(); err == nil {
		t.Error("ValidateProgram should error when program.id is empty")
	}
}

func TestValidateEpisodeSpecProgram_WithID_NoError(t *testing.T) {
	p := &config.EpisodeSpec{Program: config.ProgramConfig{ID: "my-program", Title: "t", Description: "d"}}
	if err := p.ValidateProgram(); err != nil {
		t.Errorf("ValidateProgram should not error when program.id is set: %v", err)
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

func TestJingleEntry_EffectiveTrimSilence(t *testing.T) {
	cases := []struct {
		ptr  *bool
		want bool
	}{
		{nil, true},
		{testutil.Ptr(false), false},
		{testutil.Ptr(true), true},
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
		{testutil.Ptr(false), false},
		{testutil.Ptr(true), true},
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
		{testutil.Ptr(false), false},
		{testutil.Ptr(true), true},
	}
	for _, c := range cases {
		e := config.SEEntry{Overlay: c.ptr}
		if got := e.EffectiveOverlay(); got != c.want {
			t.Errorf("ptr=%v: got %v, want %v", c.ptr, got, c.want)
		}
	}
}

func TestCornerConfig_EffectiveBGM(t *testing.T) {
	tests := []struct {
		name string
		bgm  *string
		want string
	}{
		{name: "nil returns empty", bgm: nil, want: ""},
		{name: "empty string returns empty (disabled)", bgm: testutil.Ptr(""), want: ""},
		{name: "key returned as-is", bgm: testutil.Ptr("talk_bgm"), want: "talk_bgm"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := config.CornerConfig{BGM: tt.bgm}
			if got := c.EffectiveBGM(); got != tt.want {
				t.Errorf("EffectiveBGM() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestCornerConfig_EffectiveStartPauseSec(t *testing.T) {
	tests := []struct {
		name string
		sec  *float64
		want float64
	}{
		{name: "nil returns 0", sec: nil, want: 0},
		{name: "explicit value returned", sec: testutil.Ptr(1.5), want: 1.5},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := config.CornerConfig{StartPauseSec: tt.sec}
			if got := c.EffectiveStartPauseSec(); got != tt.want {
				t.Errorf("EffectiveStartPauseSec() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCornerConfig_EffectiveEndPauseSec(t *testing.T) {
	tests := []struct {
		name string
		sec  *float64
		want float64
	}{
		{name: "nil returns 0", sec: nil, want: 0},
		{name: "explicit value returned", sec: testutil.Ptr(2.0), want: 2.0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := config.CornerConfig{EndPauseSec: tt.sec}
			if got := c.EffectiveEndPauseSec(); got != tt.want {
				t.Errorf("EffectiveEndPauseSec() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestLoadEpisodeSpec_CornerDefaults_Inherits(t *testing.T) {
	spec, err := config.LoadEpisodeSpec("testdata/episode_spec_corner_defaults.yaml")
	if err != nil {
		t.Fatalf("LoadEpisodeSpec failed: %v", err)
	}
	c := spec.Corners[0]
	// コーナー1はデフォルトを継承する
	if c.BGM == nil || *c.BGM != "talk_bgm" {
		got := "<nil>"
		if c.BGM != nil {
			got = *c.BGM
		}
		t.Errorf("Corners[0].BGM: got %q, want %q (inherited from corner_defaults)", got, "talk_bgm")
	}
	if c.StartAudio == nil {
		t.Error("Corners[0].StartAudio: got nil, want non-nil (inherited from corner_defaults)")
	} else {
		if c.StartAudio.Type != "jingle" {
			t.Errorf("Corners[0].StartAudio.Type: got %q, want jingle", c.StartAudio.Type)
		}
		if c.StartAudio.ID != "opening" {
			t.Errorf("Corners[0].StartAudio.ID: got %q, want opening", c.StartAudio.ID)
		}
	}
	if c.StartPauseSec == nil || *c.StartPauseSec != 1.0 {
		var got interface{} = "<nil>"
		if c.StartPauseSec != nil {
			got = *c.StartPauseSec
		}
		t.Errorf("Corners[0].StartPauseSec: got %v, want 1.0 (inherited from corner_defaults)", got)
	}
	if c.EndPauseSec == nil || *c.EndPauseSec != 2.0 {
		var got interface{} = "<nil>"
		if c.EndPauseSec != nil {
			got = *c.EndPauseSec
		}
		t.Errorf("Corners[0].EndPauseSec: got %v, want 2.0 (inherited from corner_defaults)", got)
	}
}

func TestLoadEpisodeSpec_CornerDefaults_Overrides(t *testing.T) {
	spec, err := config.LoadEpisodeSpec("testdata/episode_spec_corner_defaults.yaml")
	if err != nil {
		t.Fatalf("LoadEpisodeSpec failed: %v", err)
	}
	c := spec.Corners[1]
	// コーナー2はBGMとstart_pause_secを上書き
	if c.BGM == nil || *c.BGM != "other_bgm" {
		got := "<nil>"
		if c.BGM != nil {
			got = *c.BGM
		}
		t.Errorf("Corners[1].BGM: got %q, want %q (overridden)", got, "other_bgm")
	}
	if c.StartPauseSec == nil || *c.StartPauseSec != 3.0 {
		var got interface{} = "<nil>"
		if c.StartPauseSec != nil {
			got = *c.StartPauseSec
		}
		t.Errorf("Corners[1].StartPauseSec: got %v, want 3.0 (overridden)", got)
	}
	// end_pause_sec はデフォルト継承
	if c.EndPauseSec == nil || *c.EndPauseSec != 2.0 {
		var got interface{} = "<nil>"
		if c.EndPauseSec != nil {
			got = *c.EndPauseSec
		}
		t.Errorf("Corners[1].EndPauseSec: got %v, want 2.0 (inherited)", got)
	}
}

func TestLoadEpisodeSpec_CornerDefaults_Disables(t *testing.T) {
	spec, err := config.LoadEpisodeSpec("testdata/episode_spec_corner_defaults.yaml")
	if err != nil {
		t.Fatalf("LoadEpisodeSpec failed: %v", err)
	}
	c := spec.Corners[2]
	// コーナー3はBGMを空文字で無効化
	if c.BGM == nil || *c.BGM != "" {
		got := "<nil>"
		if c.BGM != nil {
			got = *c.BGM
		}
		t.Errorf("Corners[2].BGM: got %q, want empty string (disabled)", got)
	}
	// start_audio は {} で無効化 → nil に正規化
	if c.StartAudio != nil {
		t.Errorf("Corners[2].StartAudio: got %+v, want nil (disabled by empty mapping)", c.StartAudio)
	}
}

func TestLoadEpisodeSpec_CornerDefaults_BackwardCompat(t *testing.T) {
	// corner_defaults なし既存 spec が従来通り動作する
	spec, err := config.LoadEpisodeSpec("testdata/episode_spec.yaml")
	if err != nil {
		t.Fatalf("LoadEpisodeSpec failed: %v", err)
	}
	if spec.CornerDefaults != nil {
		t.Error("CornerDefaults should be nil when not specified in YAML")
	}
	// BGMを持つコーナー（corners[1]）はそのままの値を保つ
	if len(spec.Corners) < 2 {
		t.Fatal("expected at least 2 corners in episode_spec.yaml")
	}
	c := spec.Corners[1]
	if c.BGM == nil || *c.BGM != "talk_bgm" {
		got := "<nil>"
		if c.BGM != nil {
			got = *c.BGM
		}
		t.Errorf("Corners[1].BGM: got %q, want talk_bgm (backward compat)", got)
	}
}

func TestValidateAssets_CornerDefaults_EmptyBGM_Error(t *testing.T) {
	spec := &config.EpisodeSpec{
		CornerDefaults: &config.CornerDefaults{
			BGM: testutil.Ptr(""),
		},
		Assets: config.AssetsConfig{
			BGM: map[string]config.BGMEntry{},
		},
	}
	if err := spec.ValidateAssets(); err == nil {
		t.Error("ValidateAssets: expected error for corner_defaults.bgm empty string, got nil")
	}
}

func TestValidateAssets_CornerDefaults_UnknownBGM_Error(t *testing.T) {
	spec := &config.EpisodeSpec{
		CornerDefaults: &config.CornerDefaults{
			BGM: testutil.Ptr("nonexistent"),
		},
		Assets: config.AssetsConfig{
			BGM: map[string]config.BGMEntry{},
		},
	}
	if err := spec.ValidateAssets(); err == nil {
		t.Error("ValidateAssets: expected error for corner_defaults.bgm with unknown key, got nil")
	}
}

func TestValidateAssets_CornerDefaults_Valid(t *testing.T) {
	spec := &config.EpisodeSpec{
		CornerDefaults: &config.CornerDefaults{
			BGM:        testutil.Ptr("talk_bgm"),
			StartAudio: &config.AudioRef{Type: "jingle", ID: "opening"},
		},
		Assets: config.AssetsConfig{
			Jingle: map[string]config.JingleEntry{"opening": {File: "opening.mp3"}},
			BGM:    map[string]config.BGMEntry{"talk_bgm": {File: "bgm.mp3"}},
		},
	}
	if err := spec.ValidateAssets(); err != nil {
		t.Errorf("ValidateAssets: unexpected error for valid corner_defaults: %v", err)
	}
}

func TestValidateAssets_CornerDefaults_EmptyStartAudioType_Error(t *testing.T) {
	spec := &config.EpisodeSpec{
		CornerDefaults: &config.CornerDefaults{
			StartAudio: &config.AudioRef{Type: "", ID: ""},
		},
		Assets: config.AssetsConfig{
			BGM: map[string]config.BGMEntry{},
		},
	}
	if err := spec.ValidateAssets(); err == nil {
		t.Error("ValidateAssets: expected error for corner_defaults.start_audio with empty type, got nil")
	}
}

// --- SecurityConfig ---

func TestPromptInjectionConfig_EffectiveOnDetect_Default(t *testing.T) {
	c := config.PromptInjectionConfig{}
	if got := c.EffectiveOnDetect(); got != config.OnDetectExclude {
		t.Errorf("EffectiveOnDetect() = %q, want %q", got, config.OnDetectExclude)
	}
}

func TestPromptInjectionConfig_EffectiveOnDetect_Exclude(t *testing.T) {
	c := config.PromptInjectionConfig{OnDetect: config.OnDetectExclude}
	if got := c.EffectiveOnDetect(); got != config.OnDetectExclude {
		t.Errorf("EffectiveOnDetect() = %q, want %q", got, config.OnDetectExclude)
	}
}

func TestPromptInjectionConfig_EffectiveOnDetect_SanitizeLegacy(t *testing.T) {
	// "sanitize" is a deprecated alias for "exclude" (backward compatibility)
	c := config.PromptInjectionConfig{OnDetect: "sanitize"}
	if got := c.EffectiveOnDetect(); got != config.OnDetectExclude {
		t.Errorf("EffectiveOnDetect() with legacy 'sanitize' = %q, want %q", got, config.OnDetectExclude)
	}
}

func TestPromptInjectionConfig_EffectiveOnDetect_Error(t *testing.T) {
	c := config.PromptInjectionConfig{OnDetect: config.OnDetectError}
	if got := c.EffectiveOnDetect(); got != config.OnDetectError {
		t.Errorf("EffectiveOnDetect() = %q, want %q", got, config.OnDetectError)
	}
}

func TestPromptInjectionConfig_EffectiveMaxBodyChars_Default(t *testing.T) {
	c := config.PromptInjectionConfig{}
	if got := c.EffectiveMaxBodyChars(); got != config.DefaultMaxArticleBodyChars {
		t.Errorf("EffectiveMaxBodyChars() = %d, want DefaultMaxArticleBodyChars=%d", got, config.DefaultMaxArticleBodyChars)
	}
}

func TestPromptInjectionConfig_EffectiveMaxBodyChars_Custom(t *testing.T) {
	c := config.PromptInjectionConfig{MaxBodyChars: 500}
	if got := c.EffectiveMaxBodyChars(); got != 500 {
		t.Errorf("EffectiveMaxBodyChars() = %d, want 500", got)
	}
}

func TestLoadConfig_ValidationError_InvalidOnDetect(t *testing.T) {
	_, err := config.LoadConfig("testdata/config_invalid_on_detect.yaml")
	if err == nil {
		t.Error("expected error when on_detect is invalid")
	}
}

func TestLoadConfig_Security_ZeroValue(t *testing.T) {
	cfg, err := config.LoadConfig("testdata/config.yaml")
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}
	if cfg.Security.PromptInjection.EffectiveOnDetect() != config.OnDetectExclude {
		t.Errorf("default on_detect should be %q", config.OnDetectExclude)
	}
	if cfg.Security.PromptInjection.EffectiveMaxBodyChars() != config.DefaultMaxArticleBodyChars {
		t.Errorf("default max_body_chars should be DefaultMaxArticleBodyChars=%d", config.DefaultMaxArticleBodyChars)
	}
}

func TestBGMEntry_EffectiveFadeIn(t *testing.T) {
	cases := []struct {
		name string
		ptr  *float64
		want float64
	}{
		{"nil defaults to DefaultBGMFadeSec", nil, config.DefaultBGMFadeSec},
		{"zero disables fade-in", testutil.Ptr(0.0), 0.0},
		{"positive value used as-is", testutil.Ptr(2.0), 2.0},
		{"negative clamped to zero", testutil.Ptr(-1.0), 0.0},
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
		{"zero disables fade-out", testutil.Ptr(0.0), 0.0},
		{"positive value used as-is", testutil.Ptr(2.0), 2.0},
		{"negative clamped to zero", testutil.Ptr(-1.0), 0.0},
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
	zero := 0.0
	pos1 := 1.0
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
			name: "jingle trim_silence_threshold zero",
			assets: func(f string) *config.AssetsConfig {
				return &config.AssetsConfig{Jingle: map[string]config.JingleEntry{"j": {File: f, TrimSilenceThreshold: &zero}}}
			},
		},
		{
			name: "jingle trim_silence_threshold positive",
			assets: func(f string) *config.AssetsConfig {
				return &config.AssetsConfig{Jingle: map[string]config.JingleEntry{"j": {File: f, TrimSilenceThreshold: &pos1}}}
			},
		},
		{
			name: "se volume negative",
			assets: func(f string) *config.AssetsConfig {
				return &config.AssetsConfig{SE: map[string]config.SEEntry{"chime": {File: f, Volume: -0.1}}}
			},
		},
		{
			name: "se trim_silence_threshold zero",
			assets: func(f string) *config.AssetsConfig {
				return &config.AssetsConfig{SE: map[string]config.SEEntry{"chime": {File: f, Volume: 0.8, TrimSilenceThreshold: &zero}}}
			},
		},
		{
			name: "se trim_silence_threshold positive",
			assets: func(f string) *config.AssetsConfig {
				return &config.AssetsConfig{SE: map[string]config.SEEntry{"chime": {File: f, Volume: 0.8, TrimSilenceThreshold: &pos1}}}
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

func TestJingleEntry_EffectiveTrimSilenceThresholdDB(t *testing.T) {
	cases := []struct {
		name string
		ptr  *float64
		want float64
	}{
		{"nil uses default", nil, -50.0},
		{"explicit -40", testutil.Ptr(-40.0), -40.0},
		{"explicit -47.5", testutil.Ptr(-47.5), -47.5},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			e := config.JingleEntry{TrimSilenceThreshold: c.ptr}
			if got := e.EffectiveTrimSilenceThresholdDB(); got != c.want {
				t.Errorf("got %v, want %v", got, c.want)
			}
		})
	}
}

func TestSEEntry_EffectiveTrimSilenceThresholdDB(t *testing.T) {
	cases := []struct {
		name string
		ptr  *float64
		want float64
	}{
		{"nil uses default", nil, -50.0},
		{"explicit -40", testutil.Ptr(-40.0), -40.0},
		{"explicit -47.5", testutil.Ptr(-47.5), -47.5},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			e := config.SEEntry{TrimSilenceThreshold: c.ptr}
			if got := e.EffectiveTrimSilenceThresholdDB(); got != c.want {
				t.Errorf("got %v, want %v", got, c.want)
			}
		})
	}
}

func TestJingleEntry_Validate(t *testing.T) {
	cases := []struct {
		name    string
		entry   config.JingleEntry
		wantErr bool
	}{
		{"valid zero values", config.JingleEntry{FadeIn: 0, FadeOut: 0}, false},
		{"valid positive values", config.JingleEntry{FadeIn: 0.5, FadeOut: 1.0}, false},
		{"fade_in negative", config.JingleEntry{FadeIn: -1.0, FadeOut: 0}, true},
		{"fade_out negative", config.JingleEntry{FadeIn: 0, FadeOut: -0.5}, true},
		{"trim_silence_threshold nil is valid", config.JingleEntry{TrimSilenceThreshold: nil}, false},
		{"trim_silence_threshold negative is valid", config.JingleEntry{TrimSilenceThreshold: testutil.Ptr(-40.0)}, false},
		{"trim_silence_threshold zero is invalid", config.JingleEntry{TrimSilenceThreshold: testutil.Ptr(0.0)}, true},
		{"trim_silence_threshold positive is invalid", config.JingleEntry{TrimSilenceThreshold: testutil.Ptr(1.0)}, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			err := c.entry.Validate()
			if (err != nil) != c.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, c.wantErr)
			}
		})
	}
}

func TestSEEntry_Validate(t *testing.T) {
	cases := []struct {
		name    string
		entry   config.SEEntry
		wantErr bool
	}{
		{"valid zero volume", config.SEEntry{Volume: 0}, false},
		{"valid positive volume", config.SEEntry{Volume: 0.8}, false},
		{"volume negative", config.SEEntry{Volume: -0.1}, true},
		{"trim_silence_threshold nil is valid", config.SEEntry{TrimSilenceThreshold: nil}, false},
		{"trim_silence_threshold negative is valid", config.SEEntry{TrimSilenceThreshold: testutil.Ptr(-40.0)}, false},
		{"trim_silence_threshold zero is invalid", config.SEEntry{TrimSilenceThreshold: testutil.Ptr(0.0)}, true},
		{"trim_silence_threshold positive is invalid", config.SEEntry{TrimSilenceThreshold: testutil.Ptr(1.0)}, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			err := c.entry.Validate()
			if (err != nil) != c.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, c.wantErr)
			}
		})
	}
}

func TestBGMEntry_Validate(t *testing.T) {
	cases := []struct {
		name    string
		entry   config.BGMEntry
		wantErr bool
	}{
		{"valid minimum values", config.BGMEntry{Volume: 0, DuckRatio: 1}, false},
		{"valid typical values", config.BGMEntry{Volume: 0.3, DuckRatio: 8}, false},
		{"volume negative", config.BGMEntry{Volume: -1, DuckRatio: 8}, true},
		{"duck_ratio zero", config.BGMEntry{Volume: 0.3, DuckRatio: 0}, true},
		{"duck_ratio less than 1", config.BGMEntry{Volume: 0.3, DuckRatio: 0.5}, true},
		{"fade_in nil is valid", config.BGMEntry{Volume: 0.3, DuckRatio: 8, FadeIn: nil}, false},
		{"fade_in zero is valid", config.BGMEntry{Volume: 0.3, DuckRatio: 8, FadeIn: testutil.Ptr(0.0)}, false},
		{"fade_in positive is valid", config.BGMEntry{Volume: 0.3, DuckRatio: 8, FadeIn: testutil.Ptr(1.0)}, false},
		{"fade_in negative", config.BGMEntry{Volume: 0.3, DuckRatio: 8, FadeIn: testutil.Ptr(-1.0)}, true},
		{"fade_out nil is valid", config.BGMEntry{Volume: 0.3, DuckRatio: 8, FadeOut: nil}, false},
		{"fade_out zero is valid", config.BGMEntry{Volume: 0.3, DuckRatio: 8, FadeOut: testutil.Ptr(0.0)}, false},
		{"fade_out negative", config.BGMEntry{Volume: 0.3, DuckRatio: 8, FadeOut: testutil.Ptr(-0.5)}, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			err := c.entry.Validate()
			if (err != nil) != c.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, c.wantErr)
			}
		})
	}
}

// TestLoadConfigStrict_UnknownKey_NoGoTypeName verifies that strict-mode errors
// do not expose internal Go type names (e.g. "in type config.LLMConfig").
func TestLoadConfigStrict_UnknownKey_NoGoTypeName(t *testing.T) {
	_, err := config.LoadConfigStrict("testdata/config_unknown_key.yaml")
	if err == nil {
		t.Fatal("expected error for unknown key in strict mode")
	}
	if strings.Contains(err.Error(), " in type ") {
		t.Errorf("error should not expose Go type names, got: %v", err)
	}
}

// TestLoadConfig_ValidationError_FieldPathPresent verifies that validation
// errors include the relevant YAML field path so users can locate the problem.
func TestLoadConfig_ValidationError_FieldPathPresent(t *testing.T) {
	cases := []struct {
		name        string
		file        string
		wantInError string
	}{
		{
			name:        "default_style not in styles",
			file:        "testdata/config_invalid_default_style.yaml",
			wantInError: "default_style",
		},
		{
			name:        "preset out of range",
			file:        "testdata/config_invalid_preset_range.yaml",
			wantInError: "voicevox.presets",
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			_, err := config.LoadConfig(c.file)
			if err == nil {
				t.Fatalf("expected validation error for %s", c.file)
			}
			if !strings.Contains(err.Error(), c.wantInError) {
				t.Errorf("error should contain %q, got: %v", c.wantInError, err)
			}
		})
	}
}

func TestLoadEpisodeSpec_FileURLResolution(t *testing.T) {
	spec, err := config.LoadEpisodeSpec("testdata/episode_spec_file_url.yaml")
	if err != nil {
		t.Fatalf("LoadEpisodeSpec failed: %v", err)
	}

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

	absTestdata, err := filepath.Abs("testdata")
	if err != nil {
		t.Fatal(err)
	}

	feeds := sourceCorner.Source.Feeds
	if len(feeds) < 3 {
		t.Fatalf("expected 3 feeds, got %d", len(feeds))
	}

	// relative file:// URL resolved to specDir-based absolute
	wantRelFeed := "file://" + filepath.Join(absTestdata, "feeds/feed.xml")
	if feeds[0].URL != wantRelFeed {
		t.Errorf("feeds[0].URL (relative file://): got %q, want %q", feeds[0].URL, wantRelFeed)
	}

	// absolute file:// URL stays as-is
	if feeds[1].URL != "file:///abs/feed.xml" {
		t.Errorf("feeds[1].URL (absolute file://): got %q, want %q", feeds[1].URL, "file:///abs/feed.xml")
	}

	// https:// passthrough
	if feeds[2].URL != "https://example.com/rss.xml" {
		t.Errorf("feeds[2].URL (https://): got %q, want %q", feeds[2].URL, "https://example.com/rss.xml")
	}

	articles := sourceCorner.Source.Articles
	if len(articles) < 2 {
		t.Fatalf("expected 2 articles, got %d", len(articles))
	}

	// relative file:// article URL resolved
	wantRelArticle := "file://" + filepath.Join(absTestdata, "articles/article.html")
	if articles[0] != wantRelArticle {
		t.Errorf("articles[0] (relative file://): got %q, want %q", articles[0], wantRelArticle)
	}

	// https:// article passthrough
	if articles[1] != "https://example.com/articles/1" {
		t.Errorf("articles[1] (https://): got %q, want %q", articles[1], "https://example.com/articles/1")
	}
}
