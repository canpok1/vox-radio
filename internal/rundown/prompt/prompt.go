// Package prompt は rundown の flow / select サブパッケージで共有する
// LLM プロンプト用のデータ型を提供する。
package prompt

import "github.com/canpok1/vox-radio/internal/config"

// CornerForPrompt はプロンプトに渡すコーナー情報（body 除外）。
// flow / select の両サブパッケージで共通利用する。
type CornerForPrompt struct {
	Title                 string `json:"title"`
	Content               string `json:"content"`
	TargetDurationSeconds int    `json:"target_duration_seconds"`
}

// NewCornerForPrompt は config.CornerConfig から CornerForPrompt を構築する。
func NewCornerForPrompt(corner config.CornerConfig) CornerForPrompt {
	return CornerForPrompt{
		Title:                 corner.Title,
		Content:               corner.Content,
		TargetDurationSeconds: corner.LengthSec,
	}
}
