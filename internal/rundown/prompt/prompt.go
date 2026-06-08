// Package prompt は rundown の flow / select サブパッケージで共有する
// LLM プロンプト用のデータ型を提供する。
package prompt

import "github.com/canpok1/vox-radio/internal/config"

// CornerForPrompt はプロンプトに渡すコーナー情報（body 除外）。
// flow / select の両サブパッケージで共通利用する。
// AppearanceCount は今回を含む扱い回数（1 = 新コーナー）。LastEpisodeNumber は前回扱った回番号（0 = 過去になし）。
type CornerForPrompt struct {
	Title                 string `json:"title"`
	Content               string `json:"content"`
	TargetDurationSeconds int    `json:"target_duration_seconds"`
	AppearanceCount       int    `json:"appearance_count,omitempty"`
	LastEpisodeNumber     int    `json:"last_episode_number,omitempty"`
}

// NewCornerForPrompt は config.CornerConfig から CornerForPrompt を構築する。
func NewCornerForPrompt(corner config.CornerConfig) CornerForPrompt {
	return CornerForPrompt{
		Title:                 corner.Title,
		Content:               corner.Content,
		TargetDurationSeconds: corner.LengthSec,
	}
}
