package cast_test

import (
	"testing"

	"github.com/canpok1/vox-radio/internal/cast"
	"github.com/canpok1/vox-radio/internal/config"
	"github.com/canpok1/vox-radio/internal/model"
)

func TestSelect_NoCasts(t *testing.T) {
	result := cast.Select(nil, 5)
	if len(result) != 0 {
		t.Errorf("expected empty, got %v", result)
	}
}

func TestSelect_ReturnsEmptySliceNotNil(t *testing.T) {
	result := cast.Select(nil, 5)
	if result == nil {
		t.Error("Select should return empty slice, not nil")
	}
}

func TestSelect_Regular_NoCondition_AlwaysSelected(t *testing.T) {
	casts := map[string]config.CastConfig{
		"zundamon": {Type: config.CastTypeRegular, Role: "MC"},
	}
	// 回番号不明でも採用
	result := cast.Select(casts, 0)
	if len(result) != 1 || result[0].CharacterID != "zundamon" {
		t.Errorf("regular with no condition should always be selected, got %v", result)
	}
	// 回番号あっても採用
	result2 := cast.Select(casts, 5)
	if len(result2) != 1 || result2[0].CharacterID != "zundamon" {
		t.Errorf("regular with no condition should be selected at ep5, got %v", result2)
	}
}

func TestSelect_Regular_WithCondition_MatchesCondition(t *testing.T) {
	casts := map[string]config.CastConfig{
		"metan": {Type: config.CastTypeRegular, Role: "MC", Condition: &config.EpisodeCondition{Not: &config.EpisodeCondition{Episodes: []int{5}}}},
	}
	// 5回目以外は出演
	result := cast.Select(casts, 3)
	if len(result) != 1 || result[0].CharacterID != "metan" {
		t.Errorf("regular with not-ep5 condition should be selected at ep3, got %v", result)
	}
	// 5回目はお休み
	result2 := cast.Select(casts, 5)
	if len(result2) != 0 {
		t.Errorf("regular with not-ep5 condition should not be selected at ep5, got %v", result2)
	}
}

func TestSelect_Guest_EpisodeNumberUnknown_NotSelected(t *testing.T) {
	casts := map[string]config.CastConfig{
		"zunko": {Type: config.CastTypeGuest, Role: "ゲスト", Condition: &config.EpisodeCondition{Episodes: []int{3}}},
	}
	result := cast.Select(casts, 0)
	if len(result) != 0 {
		t.Errorf("guest with unknown episode number should not be selected, got %v", result)
	}
}

func TestSelect_Guest_ConditionMatches(t *testing.T) {
	casts := map[string]config.CastConfig{
		"zunko": {Type: config.CastTypeGuest, Role: "ゲスト", Condition: &config.EpisodeCondition{Episodes: []int{3, 10}}},
	}
	// 合致する回
	result := cast.Select(casts, 3)
	if len(result) != 1 || result[0].CharacterID != "zunko" {
		t.Errorf("guest should be selected at ep3, got %v", result)
	}
	// 合致しない回
	result2 := cast.Select(casts, 5)
	if len(result2) != 0 {
		t.Errorf("guest should not be selected at ep5, got %v", result2)
	}
}

func TestSelect_TypeFieldPreserved(t *testing.T) {
	casts := map[string]config.CastConfig{
		"zundamon": {Type: config.CastTypeRegular, Role: "MC"},
		"zunko":    {Type: config.CastTypeGuest, Role: "ゲスト", Condition: &config.EpisodeCondition{Episodes: []int{3}}},
	}
	result := cast.Select(casts, 3)
	if len(result) != 2 {
		t.Fatalf("expected 2 cast members, got %d", len(result))
	}
	byID := make(map[string]model.RundownCast)
	for _, r := range result {
		byID[r.CharacterID] = r
	}
	if byID["zundamon"].Type != config.CastTypeRegular {
		t.Errorf("zundamon type should be regular, got %q", byID["zundamon"].Type)
	}
	if byID["zunko"].Type != config.CastTypeGuest {
		t.Errorf("zunko type should be guest, got %q", byID["zunko"].Type)
	}
}

func TestSelect_RolePreserved(t *testing.T) {
	casts := map[string]config.CastConfig{
		"zundamon": {Type: config.CastTypeRegular, Role: "番組MC。進行役。"},
	}
	result := cast.Select(casts, 1)
	if len(result) != 1 {
		t.Fatalf("expected 1, got %d", len(result))
	}
	if result[0].Role != "番組MC。進行役。" {
		t.Errorf("role = %q, want %q", result[0].Role, "番組MC。進行役。")
	}
}

func TestSelect_SortedByCharID(t *testing.T) {
	casts := map[string]config.CastConfig{
		"zundamon": {Type: config.CastTypeRegular, Role: "MC"},
		"metan":    {Type: config.CastTypeRegular, Role: "MC"},
		"aaa":      {Type: config.CastTypeRegular, Role: "MC"},
	}
	result := cast.Select(casts, 1)
	if len(result) != 3 {
		t.Fatalf("expected 3, got %d", len(result))
	}
	expected := []string{"aaa", "metan", "zundamon"}
	for i, want := range expected {
		if result[i].CharacterID != want {
			t.Errorf("result[%d].CharacterID = %q, want %q", i, result[i].CharacterID, want)
		}
	}
}

func TestSelect_DeterministicOutput(t *testing.T) {
	casts := map[string]config.CastConfig{
		"b_char": {Type: config.CastTypeRegular, Role: "ゲスト"},
		"a_char": {Type: config.CastTypeRegular, Role: "ゲスト"},
	}
	r1 := cast.Select(casts, 1)
	r2 := cast.Select(casts, 1)
	if len(r1) != len(r2) {
		t.Fatalf("different lengths: %d vs %d", len(r1), len(r2))
	}
	for i := range r1 {
		if r1[i].CharacterID != r2[i].CharacterID {
			t.Errorf("non-deterministic: r1[%d]=%q, r2[%d]=%q", i, r1[i].CharacterID, i, r2[i].CharacterID)
		}
	}
}

func TestSelect_RegularAndGuest_Mix(t *testing.T) {
	casts := map[string]config.CastConfig{
		"zundamon": {Type: config.CastTypeRegular, Role: "MC"},
		"metan":    {Type: config.CastTypeRegular, Role: "MC", Condition: &config.EpisodeCondition{Not: &config.EpisodeCondition{Episodes: []int{5}}}},
		"zunko":    {Type: config.CastTypeGuest, Role: "ゲスト", Condition: &config.EpisodeCondition{Episodes: []int{3, 10, 20}}},
	}

	// ep3: zundamon(regular+nil), metan(regular+condition 5以外=true), zunko(guest ep3=true)
	result3 := cast.Select(casts, 3)
	if len(result3) != 3 {
		t.Errorf("ep3: expected 3 cast, got %d: %v", len(result3), result3)
	}
	// ep5: zundamon(true), metan(お休み=false), zunko(false)
	result5 := cast.Select(casts, 5)
	if len(result5) != 1 || result5[0].CharacterID != "zundamon" {
		t.Errorf("ep5: expected only zundamon, got %v", result5)
	}
	// ep0: zundamon(regular+nil=true), metan(regular+condition: ep0→false), zunko(guest: ep0→false)
	result0 := cast.Select(casts, 0)
	if len(result0) != 1 || result0[0].CharacterID != "zundamon" {
		t.Errorf("ep0: expected only zundamon (regular with no condition), got %v", result0)
	}
}

func TestSelect_Guest_ThreePersonRotation(t *testing.T) {
	casts := map[string]config.CastConfig{
		"alice": {Type: config.CastTypeGuest, Role: "ゲストA", Condition: &config.EpisodeCondition{Every: 3, Offset: 1}},
		"bob":   {Type: config.CastTypeGuest, Role: "ゲストB", Condition: &config.EpisodeCondition{Every: 3, Offset: 2}},
		"carol": {Type: config.CastTypeGuest, Role: "ゲストC", Condition: &config.EpisodeCondition{Every: 3, Offset: 0}},
	}
	tests := []struct {
		ep     int
		wantID string
	}{
		{1, "alice"},
		{2, "bob"},
		{3, "carol"},
		{4, "alice"},
		{5, "bob"},
		{6, "carol"},
	}
	for _, tt := range tests {
		result := cast.Select(casts, tt.ep)
		if len(result) != 1 {
			t.Errorf("ep%d: got %d guests, want 1", tt.ep, len(result))
			continue
		}
		if result[0].CharacterID != tt.wantID {
			t.Errorf("ep%d: got %q, want %q", tt.ep, result[0].CharacterID, tt.wantID)
		}
	}
}
