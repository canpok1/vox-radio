package manifest_test

import (
	"testing"

	"github.com/canpok1/vox-radio/internal/config"
	"github.com/canpok1/vox-radio/internal/manifest"
	"github.com/canpok1/vox-radio/internal/model"
)

func TestCollectCredits_Empty(t *testing.T) {
	// 全て空のとき空スライス（nil でない）を返すこと
	got := manifest.CollectCredits(manifest.CreditParams{})
	if got == nil {
		t.Error("CollectCredits() = nil, want []string{}")
	}
	if len(got) != 0 {
		t.Errorf("CollectCredits() len = %d, want 0", len(got))
	}
}

func TestCollectCredits_FromJingle(t *testing.T) {
	params := manifest.CreditParams{
		Assets: config.AssetsConfig{
			Jingle: map[string]config.JingleEntry{
				"opening": {Credit: "OtoLogic / CC BY 4.0"},
			},
		},
		Lines: &model.ScriptLines{
			Corners: []model.CornerLines{
				{
					StartAudio: &model.CornerAudio{Type: model.SegmentTypeJingle, AssetName: "opening"},
				},
			},
		},
	}
	got := manifest.CollectCredits(params)
	if len(got) != 1 || got[0] != "OtoLogic / CC BY 4.0" {
		t.Errorf("CollectCredits() = %v, want [OtoLogic / CC BY 4.0]", got)
	}
}

func TestCollectCredits_FromSE_ViaLines(t *testing.T) {
	params := manifest.CreditParams{
		Assets: config.AssetsConfig{
			SE: map[string]config.SEEntry{
				"chime": {Credit: "OtoLogic / CC BY 4.0"},
			},
		},
		Lines: &model.ScriptLines{
			Corners: []model.CornerLines{
				{
					EndAudio: &model.CornerAudio{Type: model.SegmentTypeSE, AssetName: "chime"},
				},
			},
		},
	}
	got := manifest.CollectCredits(params)
	if len(got) != 1 || got[0] != "OtoLogic / CC BY 4.0" {
		t.Errorf("CollectCredits() = %v, want [OtoLogic / CC BY 4.0]", got)
	}
}

func TestCollectCredits_FromBGM(t *testing.T) {
	params := manifest.CreditParams{
		Assets: config.AssetsConfig{
			BGM: map[string]config.BGMEntry{
				"talk_bgm": {Credit: "BGMer"},
			},
		},
		Lines: &model.ScriptLines{
			Corners: []model.CornerLines{
				{BGM: "talk_bgm"},
			},
		},
	}
	got := manifest.CollectCredits(params)
	if len(got) != 1 || got[0] != "BGMer" {
		t.Errorf("CollectCredits() = %v, want [BGMer]", got)
	}
}

func TestCollectCredits_FromSE_ViaScript(t *testing.T) {
	params := manifest.CreditParams{
		Assets: config.AssetsConfig{
			SE: map[string]config.SEEntry{
				"chime": {Credit: "OtoLogic / CC BY 4.0"},
			},
		},
		Script: &model.Script{
			Segments: []model.ScriptSegment{
				{Type: model.SegmentTypeSE, AssetName: "chime"},
			},
		},
	}
	got := manifest.CollectCredits(params)
	if len(got) != 1 || got[0] != "OtoLogic / CC BY 4.0" {
		t.Errorf("CollectCredits() = %v, want [OtoLogic / CC BY 4.0]", got)
	}
}

func TestCollectCredits_FromCharacters(t *testing.T) {
	params := manifest.CreditParams{
		Characters: map[string]config.CharacterConfig{
			"zundamon": {Credit: "VOICEVOX:ずんだもん"},
		},
		Casts: []model.RundownCast{
			{CharacterID: "zundamon"},
		},
	}
	got := manifest.CollectCredits(params)
	if len(got) != 1 || got[0] != "VOICEVOX:ずんだもん" {
		t.Errorf("CollectCredits() = %v, want [VOICEVOX:ずんだもん]", got)
	}
}

func TestCollectCredits_Deduplication(t *testing.T) {
	// 同じクレジットが複数ソースから現れても1件になること
	params := manifest.CreditParams{
		Assets: config.AssetsConfig{
			Jingle: map[string]config.JingleEntry{
				"opening": {Credit: "OtoLogic / CC BY 4.0"},
				"ending":  {Credit: "OtoLogic / CC BY 4.0"},
			},
		},
		Lines: &model.ScriptLines{
			Corners: []model.CornerLines{
				{
					StartAudio: &model.CornerAudio{Type: model.SegmentTypeJingle, AssetName: "opening"},
					EndAudio:   &model.CornerAudio{Type: model.SegmentTypeJingle, AssetName: "ending"},
				},
			},
		},
	}
	got := manifest.CollectCredits(params)
	if len(got) != 1 {
		t.Errorf("CollectCredits() len = %d, want 1 (dedup)", len(got))
	}
	if len(got) > 0 && got[0] != "OtoLogic / CC BY 4.0" {
		t.Errorf("CollectCredits()[0] = %q, want %q", got[0], "OtoLogic / CC BY 4.0")
	}
}

func TestCollectCredits_SkipEmptyCredit(t *testing.T) {
	// credit が空のアセットは無視されること
	params := manifest.CreditParams{
		Assets: config.AssetsConfig{
			Jingle: map[string]config.JingleEntry{
				"opening": {Credit: ""},
			},
		},
		Lines: &model.ScriptLines{
			Corners: []model.CornerLines{
				{
					StartAudio: &model.CornerAudio{Type: model.SegmentTypeJingle, AssetName: "opening"},
				},
			},
		},
	}
	got := manifest.CollectCredits(params)
	if len(got) != 0 {
		t.Errorf("CollectCredits() = %v, want [] (empty credit ignored)", got)
	}
}

func TestCollectCredits_NilLinksAndScript(t *testing.T) {
	// Lines も Script も nil のとき、Characters のみ収集されること
	params := manifest.CreditParams{
		Characters: map[string]config.CharacterConfig{
			"metan": {Credit: "VOICEVOX:四国めたん"},
		},
		Casts: []model.RundownCast{
			{CharacterID: "metan"},
		},
	}
	got := manifest.CollectCredits(params)
	if len(got) != 1 || got[0] != "VOICEVOX:四国めたん" {
		t.Errorf("CollectCredits() = %v, want [VOICEVOX:四国めたん]", got)
	}
}

func TestBuild_CreditsIncluded(t *testing.T) {
	p := newMinimalBuildParams()
	p.Assets = config.AssetsConfig{
		BGM: map[string]config.BGMEntry{"bgm1": {Credit: "OtoLogic / CC BY 4.0"}},
	}
	p.Lines = &model.ScriptLines{
		Corners: []model.CornerLines{{BGM: "bgm1", Lines: []model.Line{{Text: "テスト"}}}},
	}
	p.Rundown = model.Rundown{
		Casts: []model.RundownCast{{CharacterID: "zundamon"}},
	}
	p.Characters = map[string]config.CharacterConfig{
		"zundamon": {Credit: "VOICEVOX:ずんだもん"},
	}

	got := manifest.Build(p)
	if len(got.Credits) != 2 {
		t.Fatalf("Credits len = %d, want 2", len(got.Credits))
	}
	if got.Credits[0] != "OtoLogic / CC BY 4.0" {
		t.Errorf("Credits[0] = %q, want %q", got.Credits[0], "OtoLogic / CC BY 4.0")
	}
	if got.Credits[1] != "VOICEVOX:ずんだもん" {
		t.Errorf("Credits[1] = %q, want %q", got.Credits[1], "VOICEVOX:ずんだもん")
	}
}

func TestBuild_CreditsNeverNil(t *testing.T) {
	got := manifest.Build(newMinimalBuildParams())
	if got.Credits == nil {
		t.Error("Credits must be [] not nil")
	}
}
