package cli

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/canpok1/vox-radio/internal/config"
	"github.com/canpok1/vox-radio/internal/model"
)

func TestBuildAssetCatalog_DescriptionPropagated(t *testing.T) {
	tests := []struct {
		name     string
		assets   config.AssetsConfig
		wantKey  string
		wantDesc string
		getSlice func(model.AssetCatalog) []model.AssetCatalogEntry
	}{
		{
			name: "SE",
			assets: config.AssetsConfig{
				SE:     map[string]config.SEEntry{"chime": {File: "se/chime.mp3", Volume: 0.8, Description: "チャイム音"}},
				BGM:    map[string]config.BGMEntry{},
				Jingle: map[string]config.JingleEntry{},
			},
			wantKey: "chime", wantDesc: "チャイム音",
			getSlice: func(c model.AssetCatalog) []model.AssetCatalogEntry { return c.SE },
		},
		{
			name: "BGM",
			assets: config.AssetsConfig{
				SE:     map[string]config.SEEntry{},
				BGM:    map[string]config.BGMEntry{"coffee_break": {File: "bgm/coffee.mp3", Volume: 0.3, DuckRatio: 8, Loop: true, Description: "カフェ風BGM"}},
				Jingle: map[string]config.JingleEntry{},
			},
			wantKey: "coffee_break", wantDesc: "カフェ風BGM",
			getSlice: func(c model.AssetCatalog) []model.AssetCatalogEntry { return c.BGM },
		},
		{
			name: "Jingle",
			assets: config.AssetsConfig{
				SE:     map[string]config.SEEntry{},
				BGM:    map[string]config.BGMEntry{},
				Jingle: map[string]config.JingleEntry{"opening": {File: "jingle/opening.mp3", FadeIn: 0.5, FadeOut: 0.5, Description: "オープニングジングル"}},
			},
			wantKey: "opening", wantDesc: "オープニングジングル",
			getSlice: func(c model.AssetCatalog) []model.AssetCatalogEntry { return c.Jingle },
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := buildAssetCatalog(tt.assets)
			entries := tt.getSlice(got)
			if len(entries) != 1 {
				t.Fatalf("got %d entries, want 1", len(entries))
			}
			if entries[0].Name != tt.wantKey {
				t.Errorf("Name: got %q, want %q", entries[0].Name, tt.wantKey)
			}
			if entries[0].Description != tt.wantDesc {
				t.Errorf("Description: got %q, want %q", entries[0].Description, tt.wantDesc)
			}
		})
	}
}

func TestBuildAssetCatalog_NoInternalConfigFieldsInJSON(t *testing.T) {
	assets := config.AssetsConfig{
		SE: map[string]config.SEEntry{
			"chime": {File: "se/chime.mp3", Volume: 0.8, Description: "テスト"},
		},
		BGM: map[string]config.BGMEntry{
			"bgm1": {File: "bgm/bgm1.mp3", Volume: 0.3, DuckRatio: 8, Loop: true, Description: "テストBGM"},
		},
		Jingle: map[string]config.JingleEntry{
			"eyecatch": {File: "jingle/eyecatch.mp3", FadeIn: 0.5, FadeOut: 0.5, Description: "テストJingle"},
		},
	}
	catalog := buildAssetCatalog(assets)
	out, err := json.Marshal(catalog)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}
	jsonStr := string(out)

	for _, field := range []string{`"file"`, `"volume"`, `"duck_ratio"`, `"loop"`, `"fade_in"`, `"fade_out"`} {
		if strings.Contains(jsonStr, field) {
			t.Errorf("internal config field %s should not appear in catalog JSON, got: %s", field, jsonStr)
		}
	}
}

func TestBuildAssetCatalog_EmptyMap_ReturnsEmptyNotNull(t *testing.T) {
	assets := config.AssetsConfig{}
	got := buildAssetCatalog(assets)

	out, err := json.Marshal(got)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}
	var parsed map[string]any
	if err := json.Unmarshal(out, &parsed); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}
	for _, key := range []string{"se", "bgm", "jingle"} {
		val, ok := parsed[key]
		if !ok {
			t.Errorf("key %q not found in JSON", key)
			continue
		}
		arr, ok := val.([]any)
		if !ok || arr == nil {
			t.Errorf("key %q: got null, want empty array []", key)
		}
	}
}

func TestBuildAssetCatalog_EmptyDescription_Allowed(t *testing.T) {
	assets := config.AssetsConfig{
		SE: map[string]config.SEEntry{
			"chime": {File: "se/chime.mp3", Volume: 0.8},
		},
		BGM:    map[string]config.BGMEntry{},
		Jingle: map[string]config.JingleEntry{},
	}
	got := buildAssetCatalog(assets)
	if len(got.SE) != 1 {
		t.Fatalf("SE: got %d entries, want 1", len(got.SE))
	}
	if got.SE[0].Description != "" {
		t.Errorf("SE[0].Description: got %q, want empty", got.SE[0].Description)
	}
}

func TestBuildAssetCatalog_SortedByName(t *testing.T) {
	assets := config.AssetsConfig{
		SE: map[string]config.SEEntry{
			"transition": {File: "se/transition.mp3", Volume: 0.8},
			"chime":      {File: "se/chime.mp3", Volume: 0.8},
		},
		BGM:    map[string]config.BGMEntry{},
		Jingle: map[string]config.JingleEntry{},
	}
	got := buildAssetCatalog(assets)
	if len(got.SE) != 2 {
		t.Fatalf("SE: got %d entries, want 2", len(got.SE))
	}
	if got.SE[0].Name != "chime" {
		t.Errorf("SE[0].Name: got %q, want chime (sorted first)", got.SE[0].Name)
	}
	if got.SE[1].Name != "transition" {
		t.Errorf("SE[1].Name: got %q, want transition (sorted second)", got.SE[1].Name)
	}
}

// Verify that model.AssetCatalog type is used correctly.
var _ model.AssetCatalog = model.AssetCatalog{}
