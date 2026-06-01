package model_test

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/canpok1/vox-radio/internal/model"
)

func TestAssetCatalog_MarshalJSON_SEOnly(t *testing.T) {
	catalog := model.AssetCatalog{
		SE: []model.AssetCatalogEntry{{Name: "chime"}, {Name: "transition"}},
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
	if _, ok := got["bgm"]; ok {
		t.Error("expected no 'bgm' key in JSON (SE-only catalog)")
	}
	if _, ok := got["jingle"]; ok {
		t.Error("expected no 'jingle' key in JSON (SE-only catalog)")
	}
}

func TestAssetCatalog_Empty_MarshalJSON_EmptyArray(t *testing.T) {
	catalog := model.AssetCatalog{
		SE: []model.AssetCatalogEntry{},
	}

	out, err := json.Marshal(catalog)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}

	var got map[string]any
	if err := json.Unmarshal(out, &got); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}

	val, ok := got["se"]
	if !ok {
		t.Error("key 'se' not found in JSON")
	}
	arr, ok := val.([]any)
	if !ok || arr == nil {
		t.Error("key 'se': got null, want empty array")
	}
}

func TestAssetCatalogEntry_WithDescription_IncludedInJSON(t *testing.T) {
	entry := model.AssetCatalogEntry{Name: "chime", Description: "コーナー開始時のチャイム"}
	out, err := json.Marshal(entry)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}
	jsonStr := string(out)
	if !strings.Contains(jsonStr, `"description"`) {
		t.Errorf("expected 'description' key in JSON, got: %s", jsonStr)
	}
	if !strings.Contains(jsonStr, "コーナー開始時のチャイム") {
		t.Errorf("expected description value in JSON, got: %s", jsonStr)
	}
}

func TestAssetCatalogEntry_NoDescription_OmittedFromJSON(t *testing.T) {
	entry := model.AssetCatalogEntry{Name: "chime"}
	out, err := json.Marshal(entry)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}
	jsonStr := string(out)
	if strings.Contains(jsonStr, `"description"`) {
		t.Errorf("'description' key should be omitted when empty, got: %s", jsonStr)
	}
}

func TestAssetCatalog_NoInternalConfigFieldsExposed(t *testing.T) {
	catalog := model.AssetCatalog{
		SE: []model.AssetCatalogEntry{{Name: "chime", Description: "テスト"}},
	}
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
