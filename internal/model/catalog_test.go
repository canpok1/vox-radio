package model_test

import (
	"encoding/json"
	"testing"

	"github.com/canpok1/vox-radio/internal/model"
)

func TestAssetCatalog_MarshalJSON_AllCategories(t *testing.T) {
	catalog := model.AssetCatalog{
		SE:     []string{"chime", "transition"},
		BGM:    []string{"talk_bgm", "news_bgm"},
		Jingle: []string{"opening", "ending"},
	}

	out, err := json.Marshal(catalog)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}

	var got map[string]any
	if err := json.Unmarshal(out, &got); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}

	if _, ok := got["se"]; !ok {
		t.Error("expected 'se' key in JSON")
	}
	if _, ok := got["bgm"]; !ok {
		t.Error("expected 'bgm' key in JSON")
	}
	if _, ok := got["jingle"]; !ok {
		t.Error("expected 'jingle' key in JSON")
	}
}

func TestAssetCatalog_Empty_MarshalJSON_EmptyArrays(t *testing.T) {
	catalog := model.AssetCatalog{
		SE:     []string{},
		BGM:    []string{},
		Jingle: []string{},
	}

	out, err := json.Marshal(catalog)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}

	var got map[string][]string
	if err := json.Unmarshal(out, &got); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}

	if got["se"] == nil {
		t.Error("se should be empty array, not null")
	}
	if got["bgm"] == nil {
		t.Error("bgm should be empty array, not null")
	}
	if got["jingle"] == nil {
		t.Error("jingle should be empty array, not null")
	}
}
