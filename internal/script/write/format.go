package write

import (
	"fmt"
	"strings"

	"github.com/canpok1/vox-radio/internal/cache"
)

// episodeHeader returns the heading string for a single episode entry.
func episodeHeader(e cache.Entry) string {
	if e.EpisodeNumber <= 0 {
		return e.Datetime
	}
	if e.EpisodeTitle != "" {
		return fmt.Sprintf("第%d回（%s）%s", e.EpisodeNumber, e.EpisodeTitle, e.Datetime)
	}
	return fmt.Sprintf("第%d回 %s", e.EpisodeNumber, e.Datetime)
}

// formatPastEpisodes formats past episodes as a concise text block for LLM injection.
// Episodes are ordered newest first. Returns "（なし）" if eps is empty.
func formatPastEpisodes(eps []cache.Entry) string {
	if len(eps) == 0 {
		return "（なし）"
	}

	var sb strings.Builder
	fmt.Fprintln(&sb, "## 過去のエピソード（新しい順）")

	for i := len(eps) - 1; i >= 0; i-- {
		e := eps[i]
		fmt.Fprintf(&sb, "\n### %s\n", episodeHeader(e))

		if e.Summary != "" {
			fmt.Fprintf(&sb, "概要: %s\n", e.Summary)
		}

		for _, c := range e.Corners {
			fmt.Fprintf(&sb, "- %s: %s\n", c.Title, c.Summary)
			for _, p := range c.Points {
				fmt.Fprintf(&sb, "  ・%s\n", p)
			}
		}

		if len(e.ConversationNotes) > 0 {
			fmt.Fprintln(&sb, "会話メモ:")
			for _, n := range e.ConversationNotes {
				charIDs := ""
				if len(n.CharacterIDs) > 0 {
					charIDs = "/" + strings.Join(n.CharacterIDs, ",")
				}
				fmt.Fprintf(&sb, "- (%s%s) %s\n", n.Category, charIDs, n.Note)
			}
		}
	}

	return sb.String()
}

// RetroTryItem is one retro problem/action pair to inject into the write prompt as
// "{{retro_try}}". A write-package-local type (rather than internal/retro.Problem) so this
// domain does not need to depend on the retro domain; the cli layer converts.
type RetroTryItem struct {
	Problem string
	Action  string
}

// FormatRetroTry formats retro's in-progress problems as a concise text block for LLM injection.
// Returns "" if items is empty; LLMWriter.SetRetroTry then renders "（なし）" via stringOrNone.
func FormatRetroTry(items []RetroTryItem) string {
	if len(items) == 0 {
		return ""
	}

	var sb strings.Builder
	for _, it := range items {
		fmt.Fprintf(&sb, "- 問題: %s\n  施策: %s\n", it.Problem, it.Action)
	}
	return sb.String()
}
