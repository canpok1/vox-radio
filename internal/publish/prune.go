package publish

import (
	"context"
	"fmt"
	"net/url"
	"path"

	"github.com/canpok1/vox-radio/internal/publish/hosting"
)

// DefaultKeep is the number of episodes retained when no explicit limit is configured.
const DefaultKeep = 7

// Pruner trims episodes.json to at most Keep entries and deletes evicted audio files.
type Pruner struct {
	Hosting hosting.Hosting
	Keep    int
}

// NewPruner creates a Pruner that retains at most keep episodes.
func NewPruner(h hosting.Hosting, keep int) *Pruner {
	return &Pruner{Hosting: h, Keep: keep}
}

// Run loads episodes, removes the oldest beyond Keep, deletes their audio, and saves.
func (p *Pruner) Run(ctx context.Context) error {
	eps, err := p.Hosting.LoadEpisodes(ctx)
	if err != nil {
		return fmt.Errorf("load episodes: %w", err)
	}

	if len(eps.Episodes) <= p.Keep {
		return nil
	}

	for _, ep := range eps.Episodes[p.Keep:] {
		name, err := audioBaseName(ep.AudioURL)
		if err != nil {
			return fmt.Errorf("parse audio url %q: %w", ep.AudioURL, err)
		}
		if err := p.Hosting.DeleteAudio(ctx, name); err != nil {
			return fmt.Errorf("delete audio %q: %w", name, err)
		}
	}

	eps.Episodes = eps.Episodes[:p.Keep]
	if err := p.Hosting.SaveEpisodes(ctx, eps); err != nil {
		return fmt.Errorf("save episodes: %w", err)
	}

	return nil
}

func audioBaseName(audioURL string) (string, error) {
	u, err := url.Parse(audioURL)
	if err != nil {
		return "", err
	}
	return path.Base(u.Path), nil
}
