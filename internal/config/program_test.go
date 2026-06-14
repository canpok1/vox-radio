package config_test

import (
	"testing"

	"github.com/canpok1/vox-radio/internal/config"
)

func TestProgramConfig_Author(t *testing.T) {
	tests := []struct {
		name   string
		author string
		want   string
	}{
		{"set", "テスト放送局", "テスト放送局"},
		{"empty", "", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := config.ProgramConfig{Author: tt.author}
			if p.Author != tt.want {
				t.Errorf("Author = %q, want %q", p.Author, tt.want)
			}
		})
	}
}
