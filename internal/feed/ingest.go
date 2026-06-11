package feed

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/canpok1/vox-radio/internal/cache"
	"github.com/canpok1/vox-radio/internal/fileio"
)

// Options holds input parameters for feed generation.
type Options struct {
	CachePath string
	Spec      FeedSpec
}

// Run generates feed.xml from the given spec and cache, and writes it to the public directory.
// Returns the output path and the number of items written.
func Run(opts Options) (string, int, error) {
	cfg := opts.Spec

	mgr := cache.New(opts.CachePath)
	entries, err := mgr.Load()
	if err != nil {
		return "", 0, fmt.Errorf("load cache: %w", err)
	}

	// Validate: episode_number must be > 0
	for _, e := range entries {
		if e.EpisodeNumber <= 0 {
			return "", 0, fmt.Errorf("entry has invalid episode_number %d (must be > 0): datetime=%s", e.EpisodeNumber, e.Datetime)
		}
	}

	// Sort by datetime ascending (oldest first, BuildFeed reverses to newest-first)
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Datetime < entries[j].Datetime
	})

	xmlContent, err := BuildFeed(cfg, entries)
	if err != nil {
		return "", 0, fmt.Errorf("build feed: %w", err)
	}

	publicDir := cfg.EffectivePublicDir()
	if err := fileio.EnsureDir(publicDir); err != nil {
		return "", 0, fmt.Errorf("create public dir: %w", err)
	}

	feedPath := filepath.Join(publicDir, "feed.xml")
	if err := os.WriteFile(feedPath, []byte(xmlContent), 0o644); err != nil {
		return "", 0, fmt.Errorf("write feed.xml: %w", err)
	}

	return feedPath, len(entries), nil
}
