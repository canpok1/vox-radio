package config_test

import (
	"testing"

	"github.com/canpok1/vox-radio/internal/config"
)

func TestEpisodeSpec_ValidateProgram(t *testing.T) {
	tests := []struct {
		name    string
		spec    *config.EpisodeSpec
		wantErr bool
	}{
		{
			name:    "program.id が設定されている場合はエラーなし",
			spec:    &config.EpisodeSpec{Program: config.ProgramConfig{ID: "my-program"}},
			wantErr: false,
		},
		{
			name:    "program.id が空の場合はエラー",
			spec:    &config.EpisodeSpec{},
			wantErr: true,
		},
		{
			name:    "audio_quality が未設定の場合はエラーなし",
			spec:    &config.EpisodeSpec{Program: config.ProgramConfig{ID: "p", AudioQuality: ""}},
			wantErr: false,
		},
		{
			name:    "audio_quality=high はエラーなし",
			spec:    &config.EpisodeSpec{Program: config.ProgramConfig{ID: "p", AudioQuality: "high"}},
			wantErr: false,
		},
		{
			name:    "audio_quality=standard はエラーなし",
			spec:    &config.EpisodeSpec{Program: config.ProgramConfig{ID: "p", AudioQuality: "standard"}},
			wantErr: false,
		},
		{
			name:    "audio_quality=low はエラーなし",
			spec:    &config.EpisodeSpec{Program: config.ProgramConfig{ID: "p", AudioQuality: "low"}},
			wantErr: false,
		},
		{
			name:    "audio_quality が不正値の場合はエラー",
			spec:    &config.EpisodeSpec{Program: config.ProgramConfig{ID: "p", AudioQuality: "ultra"}},
			wantErr: true,
		},
		{
			name:    "audio_quality が大文字でもエラーなし（正規化後に判定）",
			spec:    &config.EpisodeSpec{Program: config.ProgramConfig{ID: "p", AudioQuality: "HIGH"}},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.spec.ValidateProgram()
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateProgram() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestEpisodeSpec_ValidateCorners(t *testing.T) {
	tests := []struct {
		name    string
		spec    *config.EpisodeSpec
		wantErr bool
	}{
		{
			name: "有効な corners はエラーなし",
			spec: &config.EpisodeSpec{
				Corners: []config.CornerConfig{{ID: "c1", Title: "Corner 1"}},
			},
			wantErr: false,
		},
		{
			name: "id が空のコーナーはエラー",
			spec: &config.EpisodeSpec{
				Corners: []config.CornerConfig{{Title: "Corner 1"}},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.spec.ValidateCorners()
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateCorners() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestEpisodeSpec_ValidateSingleShot(t *testing.T) {
	every2 := &config.EpisodeCondition{Every: 2}
	tests := []struct {
		name    string
		spec    *config.EpisodeSpec
		wantErr bool
	}{
		{
			name: "single_shot=false なら condition があってもエラーなし",
			spec: &config.EpisodeSpec{
				Program: config.ProgramConfig{SingleShot: false},
				Corners: []config.CornerConfig{{ID: "c1", Title: "C1", Condition: every2}},
			},
			wantErr: false,
		},
		{
			name: "single_shot=true で condition 無しならエラーなし",
			spec: &config.EpisodeSpec{
				Program: config.ProgramConfig{SingleShot: true},
				Corners: []config.CornerConfig{{ID: "c1", Title: "C1"}},
				Casts:   map[string]config.CastConfig{"zundamon": {Type: config.CastTypeRegular, Role: "司会"}},
			},
			wantErr: false,
		},
		{
			name: "single_shot=true でコーナーに condition があるとエラー",
			spec: &config.EpisodeSpec{
				Program: config.ProgramConfig{SingleShot: true},
				Corners: []config.CornerConfig{{ID: "c1", Title: "C1", Condition: every2}},
			},
			wantErr: true,
		},
		{
			name: "single_shot=true でキャストに condition があるとエラー",
			spec: &config.EpisodeSpec{
				Program: config.ProgramConfig{SingleShot: true},
				Casts:   map[string]config.CastConfig{"guest1": {Type: config.CastTypeGuest, Role: "ゲスト", Condition: every2}},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.spec.ValidateSingleShot()
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateSingleShot() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestEpisodeSpec_ValidateCast(t *testing.T) {
	tests := []struct {
		name    string
		spec    *config.EpisodeSpec
		wantErr bool
	}{
		{
			name: "コーナーのキャストが casts に宣言済みの場合はエラーなし",
			spec: &config.EpisodeSpec{
				Casts: map[string]config.CastConfig{
					"zundamon": {Type: config.CastTypeRegular, Role: "司会"},
				},
				Corners: []config.CornerConfig{
					{Title: "opening", Cast: map[string]string{"zundamon": "ボケ担当"}},
				},
			},
			wantErr: false,
		},
		{
			name: "コーナーのキャストが casts に未宣言の場合はエラー",
			spec: &config.EpisodeSpec{
				Casts: map[string]config.CastConfig{},
				Corners: []config.CornerConfig{
					{Title: "opening", Cast: map[string]string{"unknown": "司会"}},
				},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.spec.ValidateCast()
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateCast() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestEpisodeSpec_ValidateAssets(t *testing.T) {
	tests := []struct {
		name    string
		spec    *config.EpisodeSpec
		wantErr bool
	}{
		{
			name: "有効なアセット参照はエラーなし",
			spec: &config.EpisodeSpec{
				Corners: []config.CornerConfig{
					{Title: "C1", StartAudio: &config.AudioRef{Type: "jingle", ID: "opening"}},
				},
				Assets: config.AssetsConfig{
					Jingle: map[string]config.JingleEntry{"opening": {File: "opening.mp3"}},
					BGM:    map[string]config.BGMEntry{},
				},
			},
			wantErr: false,
		},
		{
			name: "存在しない jingle キーはエラー",
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

func TestEpisodeSpec_ValidateCasts(t *testing.T) {
	chars := map[string]config.CharacterConfig{
		"zundamon": {},
	}
	tests := []struct {
		name    string
		spec    *config.EpisodeSpec
		wantErr bool
	}{
		{
			name: "casts のキャラが characters に存在する場合はエラーなし",
			spec: &config.EpisodeSpec{
				Casts: map[string]config.CastConfig{
					"zundamon": {Type: config.CastTypeRegular, Role: "司会"},
				},
			},
			wantErr: false,
		},
		{
			name: "casts のキャラが characters に存在しない場合はエラー",
			spec: &config.EpisodeSpec{
				Casts: map[string]config.CastConfig{
					"unknown_char": {Type: config.CastTypeRegular, Role: "司会"},
				},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.spec.ValidateCasts(chars)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateCasts() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestEpisodeSpec_ValidateSource(t *testing.T) {
	tests := []struct {
		name    string
		spec    *config.EpisodeSpec
		wantErr bool
	}{
		{
			name: "source なしのコーナーはエラーなし",
			spec: &config.EpisodeSpec{
				Corners: []config.CornerConfig{{ID: "c1", Title: "C1"}},
			},
			wantErr: false,
		},
		{
			name: "type=feed, url あり はエラーなし",
			spec: &config.EpisodeSpec{
				Corners: []config.CornerConfig{{
					ID: "c1", Title: "C1",
					Source: config.SourceConfig{
						{Type: config.SourceTypeFeed, URL: "https://example.com/feed.xml"},
					},
				}},
			},
			wantErr: false,
		},
		{
			name: "type=web, url あり はエラーなし",
			spec: &config.EpisodeSpec{
				Corners: []config.CornerConfig{{
					ID: "c1", Title: "C1",
					Source: config.SourceConfig{
						{Type: config.SourceTypeWeb, URL: "https://example.com/article"},
					},
				}},
			},
			wantErr: false,
		},
		{
			name: "type=feed, max_items あり はエラーなし",
			spec: &config.EpisodeSpec{
				Corners: []config.CornerConfig{{
					ID: "c1", Title: "C1",
					Source: config.SourceConfig{
						{Type: config.SourceTypeFeed, URL: "https://example.com/feed.xml", MaxItems: 5},
					},
				}},
			},
			wantErr: false,
		},
		{
			name: "type が空の場合はエラー",
			spec: &config.EpisodeSpec{
				Corners: []config.CornerConfig{{
					ID: "c1", Title: "C1",
					Source: config.SourceConfig{
						{Type: "", URL: "https://example.com/feed.xml"},
					},
				}},
			},
			wantErr: true,
		},
		{
			name: "type が不正値の場合はエラー",
			spec: &config.EpisodeSpec{
				Corners: []config.CornerConfig{{
					ID: "c1", Title: "C1",
					Source: config.SourceConfig{
						{Type: "rss", URL: "https://example.com/feed.xml"},
					},
				}},
			},
			wantErr: true,
		},
		{
			name: "url が空の場合はエラー",
			spec: &config.EpisodeSpec{
				Corners: []config.CornerConfig{{
					ID: "c1", Title: "C1",
					Source: config.SourceConfig{
						{Type: config.SourceTypeFeed, URL: ""},
					},
				}},
			},
			wantErr: true,
		},
		{
			name: "type=web で max_items が非ゼロの場合はエラー",
			spec: &config.EpisodeSpec{
				Corners: []config.CornerConfig{{
					ID: "c1", Title: "C1",
					Source: config.SourceConfig{
						{Type: config.SourceTypeWeb, URL: "https://example.com/article", MaxItems: 3},
					},
				}},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.spec.ValidateSource()
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateSource() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestEpisodeSpec_Validate_CallsAllValidations は Validate が全検証を通過することを確認する。
func TestEpisodeSpec_Validate_CallsAllValidations(t *testing.T) {
	chars := map[string]config.CharacterConfig{
		"zundamon": {},
	}
	validSpec := &config.EpisodeSpec{
		Program: config.ProgramConfig{ID: "my-program"},
		Corners: []config.CornerConfig{
			{
				ID:    "c1",
				Title: "opening",
				Cast:  map[string]string{"zundamon": "司会"},
			},
		},
		Casts: map[string]config.CastConfig{
			"zundamon": {Type: config.CastTypeRegular, Role: "司会"},
		},
		Assets: config.AssetsConfig{
			Jingle: map[string]config.JingleEntry{},
			BGM:    map[string]config.BGMEntry{},
		},
	}
	if err := validSpec.Validate(chars); err != nil {
		t.Errorf("Validate() unexpected error = %v", err)
	}
}

// TestEpisodeSpec_Validate_ErrorCases は各バリデーションのエラーが Validate から伝播することを確認する。
func TestEpisodeSpec_Validate_ErrorCases(t *testing.T) {
	chars := map[string]config.CharacterConfig{
		"zundamon": {},
	}
	tests := []struct {
		name string
		spec *config.EpisodeSpec
	}{
		{
			name: "program.id が空のとき Validate はエラーを返す",
			spec: &config.EpisodeSpec{
				Program: config.ProgramConfig{},
				Assets:  config.AssetsConfig{Jingle: map[string]config.JingleEntry{}, BGM: map[string]config.BGMEntry{}},
			},
		},
		{
			name: "corner の id が空のとき Validate はエラーを返す",
			spec: &config.EpisodeSpec{
				Program: config.ProgramConfig{ID: "prog"},
				Corners: []config.CornerConfig{{Title: "opening"}},
				Assets:  config.AssetsConfig{Jingle: map[string]config.JingleEntry{}, BGM: map[string]config.BGMEntry{}},
			},
		},
		{
			name: "corner cast が未宣言のとき Validate はエラーを返す",
			spec: &config.EpisodeSpec{
				Program: config.ProgramConfig{ID: "prog"},
				Corners: []config.CornerConfig{
					{ID: "c1", Title: "opening", Cast: map[string]string{"ghost": "司会"}},
				},
				Casts:  map[string]config.CastConfig{},
				Assets: config.AssetsConfig{Jingle: map[string]config.JingleEntry{}, BGM: map[string]config.BGMEntry{}},
			},
		},
		{
			name: "casts のキャラが characters に存在しないとき Validate はエラーを返す",
			spec: &config.EpisodeSpec{
				Program: config.ProgramConfig{ID: "prog"},
				Corners: []config.CornerConfig{{ID: "c1", Title: "opening"}},
				Casts: map[string]config.CastConfig{
					"unknown": {Type: config.CastTypeRegular, Role: "司会"},
				},
				Assets: config.AssetsConfig{Jingle: map[string]config.JingleEntry{}, BGM: map[string]config.BGMEntry{}},
			},
		},
		{
			name: "存在しないアセットを参照するとき Validate はエラーを返す",
			spec: &config.EpisodeSpec{
				Program: config.ProgramConfig{ID: "prog"},
				Corners: []config.CornerConfig{
					{ID: "c1", Title: "opening", StartAudio: &config.AudioRef{Type: "jingle", ID: "nonexistent"}},
				},
				Casts:  map[string]config.CastConfig{},
				Assets: config.AssetsConfig{Jingle: map[string]config.JingleEntry{}, BGM: map[string]config.BGMEntry{}},
			},
		},
		{
			name: "source の type が不正なとき Validate はエラーを返す",
			spec: &config.EpisodeSpec{
				Program: config.ProgramConfig{ID: "prog"},
				Corners: []config.CornerConfig{
					{ID: "c1", Title: "opening", Source: config.SourceConfig{{Type: "invalid", URL: "https://example.com/feed"}}},
				},
				Casts:  map[string]config.CastConfig{},
				Assets: config.AssetsConfig{Jingle: map[string]config.JingleEntry{}, BGM: map[string]config.BGMEntry{}},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.spec.Validate(chars); err == nil {
				t.Error("Validate() expected error but got nil")
			}
		})
	}
}
