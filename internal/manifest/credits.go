package manifest

import (
	"github.com/canpok1/vox-radio/internal/config"
	"github.com/canpok1/vox-radio/internal/model"
)

// CreditParams holds the data needed to collect credits from used assets and characters.
type CreditParams struct {
	Assets     config.AssetsConfig
	Characters map[string]config.CharacterConfig
	Lines      *model.ScriptLines // nil = skip
	Script     *model.Script      // nil = skip
	Casts      []model.RundownCast
}

// CollectCredits collects and deduplicates credits from used assets and characters.
// Returns a non-nil slice (empty when no credits found).
// Order is first-seen (stable).
func CollectCredits(p CreditParams) []string {
	seen := make(map[string]bool)
	credits := make([]string, 0)

	add := func(credit string) {
		if credit != "" && !seen[credit] {
			seen[credit] = true
			credits = append(credits, credit)
		}
	}

	if p.Lines != nil {
		for _, corner := range p.Lines.Corners {
			if corner.StartAudio != nil {
				add(assetCredit(p.Assets, corner.StartAudio.Type, corner.StartAudio.AssetName))
			}
			if corner.EndAudio != nil {
				add(assetCredit(p.Assets, corner.EndAudio.Type, corner.EndAudio.AssetName))
			}
			if corner.BGM != "" {
				if entry, ok := p.Assets.BGM[corner.BGM]; ok {
					add(entry.Credit)
				}
			}
		}
	}

	if p.Script != nil {
		for _, seg := range p.Script.Segments {
			if seg.Type == model.SegmentTypeSE && seg.AssetName != "" {
				if entry, ok := p.Assets.SE[seg.AssetName]; ok {
					add(entry.Credit)
				}
			}
		}
	}

	for _, cast := range p.Casts {
		if ch, ok := p.Characters[cast.CharacterID]; ok {
			add(ch.Credit)
		}
	}

	return credits
}

func assetCredit(assets config.AssetsConfig, segType model.SegmentType, name string) string {
	switch segType {
	case model.SegmentTypeJingle:
		if entry, ok := assets.Jingle[name]; ok {
			return entry.Credit
		}
	case model.SegmentTypeSE:
		if entry, ok := assets.SE[name]; ok {
			return entry.Credit
		}
	}
	return ""
}
