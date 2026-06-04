package guest_test

import (
	"testing"

	"github.com/canpok1/vox-radio/internal/config"
	"github.com/canpok1/vox-radio/internal/guest"
	"github.com/canpok1/vox-radio/internal/model"
)

func TestSelect_NoGuests(t *testing.T) {
	result := guest.Select(nil, 5)
	if len(result) != 0 {
		t.Errorf("expected empty, got %v", result)
	}
}

func TestSelect_EpisodeNumberUnknown(t *testing.T) {
	guests := map[string]config.GuestConfig{
		"zundamon": {Role: "ゲスト", Condition: config.GuestCondition{Episodes: []int{3}}},
	}
	result := guest.Select(guests, 0)
	if len(result) != 0 {
		t.Errorf("episode number 0 (unknown) should return empty, got %v", result)
	}
}

func TestSelect_ExplicitEpisodeList(t *testing.T) {
	guests := map[string]config.GuestConfig{
		"zundamon": {Role: "ゲスト", Condition: config.GuestCondition{Episodes: []int{3, 10}}},
	}
	// 合致する回
	result := guest.Select(guests, 3)
	if len(result) != 1 || result[0].CharacterID != "zundamon" {
		t.Errorf("expected zundamon at episode 3, got %v", result)
	}
	// 合致しない回
	result2 := guest.Select(guests, 5)
	if len(result2) != 0 {
		t.Errorf("expected no guests at episode 5, got %v", result2)
	}
}

func TestSelect_Every(t *testing.T) {
	guests := map[string]config.GuestConfig{
		"metan": {Role: "解説ゲスト", Condition: config.GuestCondition{Every: 5}},
	}
	// 合致する回（5の倍数）
	result := guest.Select(guests, 10)
	if len(result) != 1 || result[0].CharacterID != "metan" {
		t.Errorf("expected metan at episode 10, got %v", result)
	}
	// 合致しない回
	result2 := guest.Select(guests, 7)
	if len(result2) != 0 {
		t.Errorf("expected no guests at episode 7, got %v", result2)
	}
}

func TestSelect_MultipleGuestsAreSorted(t *testing.T) {
	guests := map[string]config.GuestConfig{
		"zundamon": {Role: "ゲストA", Condition: config.GuestCondition{Episodes: []int{5}}},
		"metan":    {Role: "ゲストB", Condition: config.GuestCondition{Episodes: []int{5}}},
		"aaa":      {Role: "ゲストC", Condition: config.GuestCondition{Episodes: []int{5}}},
	}
	result := guest.Select(guests, 5)
	if len(result) != 3 {
		t.Fatalf("expected 3 guests, got %d", len(result))
	}
	// キャラID昇順になっていること
	expected := []string{"aaa", "metan", "zundamon"}
	for i, want := range expected {
		if result[i].CharacterID != want {
			t.Errorf("result[%d].CharacterID = %q, want %q", i, result[i].CharacterID, want)
		}
	}
}

func TestSelect_DeterministicOutput(t *testing.T) {
	guests := map[string]config.GuestConfig{
		"b_char": {Role: "ゲスト", Condition: config.GuestCondition{Episodes: []int{1}}},
		"a_char": {Role: "ゲスト", Condition: config.GuestCondition{Episodes: []int{1}}},
	}
	r1 := guest.Select(guests, 1)
	r2 := guest.Select(guests, 1)
	if len(r1) != len(r2) {
		t.Fatalf("different lengths: %d vs %d", len(r1), len(r2))
	}
	for i := range r1 {
		if r1[i].CharacterID != r2[i].CharacterID {
			t.Errorf("non-deterministic: r1[%d]=%q, r2[%d]=%q", i, r1[i].CharacterID, i, r2[i].CharacterID)
		}
	}
}

func TestSelect_ReturnsEmptySliceNotNil(t *testing.T) {
	result := guest.Select(nil, 5)
	if result == nil {
		t.Error("Select should return empty slice, not nil")
	}
}

func TestSelect_RoleIsPreserved(t *testing.T) {
	guests := map[string]config.GuestConfig{
		"zundamon": {Role: "古参リスナー出身の常連ゲスト", Condition: config.GuestCondition{Episodes: []int{3}}},
	}
	result := guest.Select(guests, 3)
	if len(result) != 1 {
		t.Fatalf("expected 1 guest, got %d", len(result))
	}
	want := model.RundownGuest{CharacterID: "zundamon", Role: "古参リスナー出身の常連ゲスト"}
	if result[0] != want {
		t.Errorf("got %+v, want %+v", result[0], want)
	}
}
