package config

import "fmt"

// ResolveCornersForEpisode は condition を持つコーナーを回番号で絞り込んだコーナー列を返す。
// Condition == nil のコーナー（固定）は常に採用。順序は元の配列のまま保持する。
// episodeNumber <= 0（回番号不明）の場合は全コーナーを採用する（呼び出し側で警告）。乱数なし。
func ResolveCornersForEpisode(corners []CornerConfig, episodeNumber int) []CornerConfig {
	result := make([]CornerConfig, 0, len(corners))
	for _, c := range corners {
		if c.Condition == nil || episodeNumber <= 0 || c.Condition.Matches(episodeNumber) {
			result = append(result, c)
		}
	}
	return result
}

// ResolveCornersByIDs は ids に含まれる id のコーナーだけを ids の順序で返す。
// rundown.Corners の id 順を渡し、script 系で採用コーナーを再構成するために使う（回番号不要）。
// ids にあるが spec に存在しない id があればエラー。
func ResolveCornersByIDs(corners []CornerConfig, ids []string) ([]CornerConfig, error) {
	cornerMap := make(map[string]CornerConfig, len(corners))
	for _, c := range corners {
		cornerMap[c.ID] = c
	}
	result := make([]CornerConfig, 0, len(ids))
	for _, id := range ids {
		c, ok := cornerMap[id]
		if !ok {
			return nil, fmt.Errorf("corner id %q not found in spec", id)
		}
		result = append(result, c)
	}
	return result, nil
}
