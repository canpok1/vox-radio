package publish

import (
	"context"
	"fmt"
	"io"
	"testing"

	"github.com/canpok1/vox-radio/internal/model"
)

type pruneHosting struct {
	episodes     model.Episodes
	savedEps     model.Episodes
	deletedAudio []string
}

func (m *pruneHosting) PutAudio(_ context.Context, name string, _ io.Reader) (string, error) {
	return "https://example.com/audio/" + name, nil
}

func (m *pruneHosting) PutFeed(_ context.Context, _ []byte) (string, error) {
	return "https://example.com/feed.xml", nil
}

func (m *pruneHosting) LoadEpisodes(_ context.Context) (model.Episodes, error) {
	return m.episodes, nil
}

func (m *pruneHosting) SaveEpisodes(_ context.Context, e model.Episodes) error {
	m.savedEps = e
	return nil
}

func (m *pruneHosting) DeleteAudio(_ context.Context, name string) error {
	m.deletedAudio = append(m.deletedAudio, name)
	return nil
}

// makeTestEpisodes creates count episodes in newest-first order.
// Episodes have dates 2026-05-{count} through 2026-05-01.
func makeTestEpisodes(count int) model.Episodes {
	eps := model.Episodes{Episodes: make([]model.Episode, count)}
	for i := 0; i < count; i++ {
		day := count - i
		date := fmt.Sprintf("2026-05-%02d", day)
		eps.Episodes[i] = model.Episode{
			GUID:     "episode-" + date,
			PubDate:  date + "T00:00:00Z",
			AudioURL: "https://example.com/audio/episode_" + date + ".mp3",
		}
	}
	return eps
}

func TestPruner_Run_NoPruneWhenAtLimit(t *testing.T) {
	h := &pruneHosting{episodes: makeTestEpisodes(7)}
	p := NewPruner(h, 7)

	if err := p.Run(context.Background()); err != nil {
		t.Fatalf("Run: %v", err)
	}

	if len(h.deletedAudio) != 0 {
		t.Errorf("expected no deletions, got %v", h.deletedAudio)
	}
	if len(h.savedEps.Episodes) != 0 {
		t.Errorf("expected no save when not pruning, got %d episodes", len(h.savedEps.Episodes))
	}
}

func TestPruner_Run_DeletesOldestEpisode(t *testing.T) {
	h := &pruneHosting{episodes: makeTestEpisodes(8)}
	p := NewPruner(h, 7)

	if err := p.Run(context.Background()); err != nil {
		t.Fatalf("Run: %v", err)
	}

	if len(h.deletedAudio) != 1 {
		t.Fatalf("expected 1 deletion, got %d: %v", len(h.deletedAudio), h.deletedAudio)
	}
	want := "episode_2026-05-01.mp3"
	if h.deletedAudio[0] != want {
		t.Errorf("deleted audio = %q, want %q", h.deletedAudio[0], want)
	}
}

func TestPruner_Run_SavesTrimmedEpisodes(t *testing.T) {
	h := &pruneHosting{episodes: makeTestEpisodes(8)}
	p := NewPruner(h, 7)

	if err := p.Run(context.Background()); err != nil {
		t.Fatalf("Run: %v", err)
	}

	if len(h.savedEps.Episodes) != 7 {
		t.Fatalf("expected 7 saved episodes, got %d", len(h.savedEps.Episodes))
	}
	if h.savedEps.Episodes[0].GUID != "episode-2026-05-08" {
		t.Errorf("newest episode should be first, got %q", h.savedEps.Episodes[0].GUID)
	}
}

func TestPruner_Run_EmptyEpisodes(t *testing.T) {
	h := &pruneHosting{episodes: model.Episodes{Episodes: make([]model.Episode, 0)}}
	p := NewPruner(h, 7)

	if err := p.Run(context.Background()); err != nil {
		t.Fatalf("Run: %v", err)
	}

	if len(h.deletedAudio) != 0 {
		t.Errorf("expected no deletions, got %v", h.deletedAudio)
	}
}
