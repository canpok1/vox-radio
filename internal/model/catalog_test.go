package model_test

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/canpok1/vox-radio/internal/model"
)

func TestAssetCatalog_MarshalJSON_AllCategories(t *testing.T) {
	catalog := model.AssetCatalog{
		SE:     []model.AssetCatalogEntry{{Name: "chime"}, {Name: "transition"}},
		BGM:    []model.AssetCatalogEntry{{Name: "talk_bgm"}, {Name: "news_bgm"}},
		Jingle: []model.AssetCatalogEntry{{Name: "opening"}, {Name: "ending"}},
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
		SE:     []model.AssetCatalogEntry{},
		BGM:    []model.AssetCatalogEntry{},
		Jingle: []model.AssetCatalogEntry{},
	}

	out, err := json.Marshal(catalog)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}

	var got map[string]any
	if err := json.Unmarshal(out, &got); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}

	for _, key := range []string{"se", "bgm", "jingle"} {
		val, ok := got[key]
		if !ok {
			t.Errorf("key %q not found in JSON", key)
			continue
		}
		arr, ok := val.([]any)
		if !ok || arr == nil {
			t.Errorf("key %q: got null, want empty array", key)
		}
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
		SE:     []model.AssetCatalogEntry{{Name: "chime", Description: "テスト"}},
		BGM:    []model.AssetCatalogEntry{{Name: "bgm1", Description: "テストBGM"}},
		Jingle: []model.AssetCatalogEntry{{Name: "eyecatch", Description: "テストJingle"}},
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
