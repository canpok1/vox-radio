package config_test

import (
	"testing"

	"github.com/canpok1/vox-radio/internal/config"
)

func TestEpisodeSpec_Deprecations(t *testing.T) {
	tests := []struct {
		name       string
		spec       *config.EpisodeSpec
		wantFields []string
	}{
		{
			name: "非推奨フィールド未使用なら警告なし",
			spec: &config.EpisodeSpec{
				Corners: []config.CornerConfig{{ID: "c1", TargetChars: 100}},
			},
			wantFields: nil,
		},
		{
			name: "length_sec 使用時は警告",
			spec: &config.EpisodeSpec{
				Corners: []config.CornerConfig{{ID: "c1", LengthSec: 14}},
			},
			wantFields: []string{"corners[].length_sec"},
		},
		{
			name: "chars_per_minute 使用時は警告",
			spec: &config.EpisodeSpec{
				Program: config.ProgramConfig{CharsPerMinute: 300},
			},
			wantFields: []string{"program.chars_per_minute"},
		},
		{
			name: "両方使用時は両方警告",
			spec: &config.EpisodeSpec{
				Program: config.ProgramConfig{CharsPerMinute: 300},
				Corners: []config.CornerConfig{{ID: "c1", LengthSec: 14}},
			},
			wantFields: []string{"corners[].length_sec", "program.chars_per_minute"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.spec.Deprecations()
			if len(got) != len(tt.wantFields) {
				t.Fatalf("Deprecations() returned %d warnings, want %d: %+v", len(got), len(tt.wantFields), got)
			}
			for i, w := range got {
				if w.Field != tt.wantFields[i] {
					t.Errorf("Deprecations()[%d].Field = %q, want %q", i, w.Field, tt.wantFields[i])
				}
				if w.Message == "" {
					t.Errorf("Deprecations()[%d].Message is empty", i)
				}
			}
		})
	}
}
