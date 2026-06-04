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

// ResolveCornersByTitles は titles に含まれるタイトルのコーナーだけを titles の順序で返す。
// rundown.Corners のタイトル順を渡し、script 系で採用コーナーを再構成するために使う（回番号不要）。
// titles にあるが spec に存在しないタイトルがあればエラー。
func ResolveCornersByTitles(corners []CornerConfig, titles []string) ([]CornerConfig, error) {
	cornerMap := make(map[string]CornerConfig, len(corners))
	for _, c := range corners {
		cornerMap[c.Title] = c
	}
	result := make([]CornerConfig, 0, len(titles))
	for _, title := range titles {
		c, ok := cornerMap[title]
		if !ok {
			return nil, fmt.Errorf("corner %q not found in spec", title)
		}
		result = append(result, c)
	}
	return result, nil
}
