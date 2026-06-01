package cli

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/canpok1/vox-radio/internal/config"
	"github.com/canpok1/vox-radio/internal/model"
)

func TestBuildAssetCatalog_DescriptionPropagated(t *testing.T) {
	assets := config.AssetsConfig{
		SE: map[string]config.SEEntry{"chime": {File: "se/chime.mp3", Volume: 0.8, Description: "チャイム音"}},
	}
	got := buildAssetCatalog(assets)
	if len(got.SE) != 1 {
		t.Fatalf("got %d SE entries, want 1", len(got.SE))
	}
	if got.SE[0].Name != "chime" {
		t.Errorf("Name: got %q, want chime", got.SE[0].Name)
	}
	if got.SE[0].Description != "チャイム音" {
		t.Errorf("Description: got %q, want チャイム音", got.SE[0].Description)
	}
}

func TestBuildAssetCatalog_BGMAndJingleNotIncluded(t *testing.T) {
	assets := config.AssetsConfig{
		SE:     map[string]config.SEEntry{"chime": {File: "se/chime.mp3", Volume: 0.8, Description: "テスト"}},
		BGM:    map[string]config.BGMEntry{"bgm1": {File: "bgm/bgm1.mp3", Volume: 0.3, DuckRatio: 8, Loop: true, Description: "テストBGM"}},
		Jingle: map[string]config.JingleEntry{"eyecatch": {File: "jingle/eyecatch.mp3", FadeIn: 0.5, FadeOut: 0.5, Description: "テストJingle"}},
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
	// BGM/Jingle keys must not appear in the SE-only catalog
	for _, key := range []string{`"bgm"`, `"jingle"`} {
		if strings.Contains(jsonStr, key) {
			t.Errorf("key %s should not appear in SE-only catalog JSON, got: %s", key, jsonStr)
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
	val, ok := parsed["se"]
	if !ok {
		t.Error("key 'se' not found in JSON")
	}
	arr, ok := val.([]any)
	if !ok || arr == nil {
		t.Error("key 'se': got null, want empty array []")
	}
}

func TestBuildAssetCatalog_EmptyDescription_Allowed(t *testing.T) {
	assets := config.AssetsConfig{
		SE: map[string]config.SEEntry{
			"chime": {File: "se/chime.mp3", Volume: 0.8},
		},
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
