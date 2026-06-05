package cast

import (
	"maps"
	"slices"

	"github.com/canpok1/vox-radio/internal/config"
	"github.com/canpok1/vox-radio/internal/model"
)

// Select は episodeNumber に出演条件が合致するキャストの確定リストを返す。
// 出力はキャラID昇順にソートして決定論的にする。乱数なし。
//
// - regular & condition == nil → 常に採用（episodeNumber 不問）
// - regular & condition != nil → condition.Matches(episodeNumber) で採用
// - guest → episodeNumber > 0 かつ condition.Matches(episodeNumber) で採用
func Select(casts map[string]config.CastConfig, episodeNumber int) []model.RundownCast {
	result := make([]model.RundownCast, 0)
	if len(casts) == 0 {
		return result
	}

	ids := slices.Sorted(maps.Keys(casts))

	for _, id := range ids {
		c := casts[id]
		var selected bool
		switch c.Type {
		case config.CastTypeRegular:
			if c.Condition == nil {
				selected = true
			} else {
				selected = c.Condition.Matches(episodeNumber)
			}
		case config.CastTypeGuest:
			selected = episodeNumber > 0 && c.Condition != nil && c.Condition.Matches(episodeNumber)
		}
		if selected {
			result = append(result, model.RundownCast{CharacterID: id, Role: c.Role, Type: c.Type})
		}
	}
	return result
}
