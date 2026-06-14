package config_test

import (
	"testing"

	"github.com/canpok1/vox-radio/internal/config"
)

func TestAssetsCredit_JingleEntry(t *testing.T) {
	entry := config.JingleEntry{Credit: "OtoLogic / CC BY 4.0"}
	if entry.Credit != "OtoLogic / CC BY 4.0" {
		t.Errorf("JingleEntry.Credit = %q, want %q", entry.Credit, "OtoLogic / CC BY 4.0")
	}
}

func TestAssetsCredit_SEEntry(t *testing.T) {
	entry := config.SEEntry{Credit: "OtoLogic / CC BY 4.0"}
	if entry.Credit != "OtoLogic / CC BY 4.0" {
		t.Errorf("SEEntry.Credit = %q, want %q", entry.Credit, "OtoLogic / CC BY 4.0")
	}
}

func TestAssetsCredit_BGMEntry(t *testing.T) {
	entry := config.BGMEntry{Credit: "BGMer"}
	if entry.Credit != "BGMer" {
		t.Errorf("BGMEntry.Credit = %q, want %q", entry.Credit, "BGMer")
	}
}

func TestAssetsCredit_LoadFromYAML(t *testing.T) {
	assets, err := config.LoadAssetsFileStrict("testdata/assets_with_credit.yaml")
	if err != nil {
		t.Fatalf("LoadAssetsFileStrict: %v", err)
	}

	jingle, ok := assets.Jingle["opening"]
	if !ok {
		t.Fatal("jingle[opening] not found")
	}
	if jingle.Credit != "OtoLogic / CC BY 4.0" {
		t.Errorf("jingle[opening].credit = %q, want %q", jingle.Credit, "OtoLogic / CC BY 4.0")
	}

	se, ok := assets.SE["chime"]
	if !ok {
		t.Fatal("se[chime] not found")
	}
	if se.Credit != "OtoLogic / CC BY 4.0" {
		t.Errorf("se[chime].credit = %q, want %q", se.Credit, "OtoLogic / CC BY 4.0")
	}

	bgm, ok := assets.BGM["talk_bgm"]
	if !ok {
		t.Fatal("bgm[talk_bgm] not found")
	}
	if bgm.Credit != "BGMer" {
		t.Errorf("bgm[talk_bgm].credit = %q, want %q", bgm.Credit, "BGMer")
	}
}

func TestAssetsCredit_OmitWhenEmpty(t *testing.T) {
	// credit フィールドがない既存の assets.yaml を読み込んでも壊れないこと
	assets, err := config.LoadAssetsFileStrict("testdata/assets.yaml")
	if err != nil {
		t.Fatalf("LoadAssetsFileStrict: %v", err)
	}
	for name, entry := range assets.Jingle {
		if entry.Credit != "" {
			t.Errorf("jingle[%q].credit = %q, want empty (omitempty)", name, entry.Credit)
		}
	}
}
