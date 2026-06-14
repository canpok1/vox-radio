package model

import (
	"fmt"
	"time"
)

// EpisodeMeta holds per-episode metadata used to populate ID3 tags.
type EpisodeMeta struct {
	Number      int
	Title       string
	GeneratedAt time.Time
}

// EpisodeDisplayTitle builds the display title for an episode.
// Returns "第N回 subtitle" when number > 0 and subtitle is non-empty.
// Returns "第N回" when number > 0 and subtitle is empty.
// Returns fallbackTitle otherwise.
func EpisodeDisplayTitle(episodeNumber int, episodeTitle, fallbackTitle string) string {
	if episodeNumber > 0 && episodeTitle != "" {
		return fmt.Sprintf("第%d回 %s", episodeNumber, episodeTitle)
	}
	if episodeNumber > 0 {
		return fmt.Sprintf("第%d回", episodeNumber)
	}
	return fallbackTitle
}
