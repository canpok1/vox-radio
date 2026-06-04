package guest

import (
	"maps"
	"slices"

	"github.com/canpok1/vox-radio/internal/config"
	"github.com/canpok1/vox-radio/internal/model"
)

// Select は episodeNumber に出現条件が合致するゲストの確定リストを返す。
// 出力はキャラID昇順にソートして決定論的にする。
// episodeNumber <= 0（回番号不明）の場合は空スライスを返す。乱数なし。
func Select(guests map[string]config.GuestConfig, episodeNumber int) []model.RundownGuest {
	result := make([]model.RundownGuest, 0)
	if episodeNumber <= 0 || len(guests) == 0 {
		return result
	}

	ids := slices.Sorted(maps.Keys(guests))

	for _, id := range ids {
		g := guests[id]
		if g.Condition.Matches(episodeNumber) {
			result = append(result, model.RundownGuest{CharacterID: id, Role: g.Role})
		}
	}
	return result
}
