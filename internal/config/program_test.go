package config_test

import (
	"testing"

	"github.com/canpok1/vox-radio/internal/config"
)

func TestProgramConfig_EffectiveAudioQuality(t *testing.T) {
	tests := []struct {
		name         string
		audioQuality string
		want         string
	}{
		{"未設定はstandardになる", "", config.DefaultAudioQuality},
		{"high", "high", "high"},
		{"standard", "standard", "standard"},
		{"low", "low", "low"},
		{"大文字は小文字正規化", "HIGH", "high"},
		{"混在も小文字正規化", "Standard", "standard"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := config.ProgramConfig{AudioQuality: tt.audioQuality}
			got := p.EffectiveAudioQuality()
			if got != tt.want {
				t.Errorf("EffectiveAudioQuality() = %q, want %q", got, tt.want)
			}
		})
	}
}

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
