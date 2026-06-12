package manifest

import (
	"time"

	"github.com/canpok1/vox-radio/internal/config"
	"github.com/canpok1/vox-radio/internal/model"
)

// BuildParams holds all parameters required to build a Manifest.
type BuildParams struct {
	Program           config.ProgramConfig
	Corners           []config.CornerConfig
	Rundown           model.Rundown
	AudioFile         string
	GeneratedAt       time.Time
	Summary           string
	CornerSummaries   map[string]model.CornerSummary
	ConversationNotes []model.ConversationNote
	EpisodeNumber     int
	EpisodeTitle      string
}

// Build constructs a Manifest from BuildParams.
// BuildParams.CornerSummaries maps corner title to its LLM-generated summary; nil means no corner summaries.
// BuildParams.ConversationNotes contains non-rundown conversation information extracted from the episode.
// BuildParams.EpisodeNumber 0 means unknown (omitted from manifest). BuildParams.EpisodeTitle "" means unknown (omitted).
func Build(p BuildParams) model.Manifest {
	cornerMap := p.Rundown.CornerMap()
	manifestCorners := make([]model.ManifestCorner, 0, len(p.Corners))
	for _, c := range p.Corners {
		rc := cornerMap[c.ID]
		refs := make([]model.ArticleRef, 0, len(rc.Articles))
		for _, a := range rc.Articles {
			refs = append(refs, model.ArticleRef{Title: a.Title, URL: a.URL})
		}
		cs := p.CornerSummaries[c.Title]
		manifestCorners = append(manifestCorners, model.ManifestCorner{
			ID:       c.ID,
			Title:    c.Title,
			Summary:  cs.Summary,
			Points:   model.NonNil(cs.Points),
			Articles: refs,
		})
	}
	notes := model.NonNil(p.ConversationNotes)
	casts := model.NonNil(p.Rundown.Casts)

	return model.Manifest{
		Title:             p.Program.Title,
		EpisodeNumber:     p.EpisodeNumber,
		EpisodeTitle:      p.EpisodeTitle,
		Description:       p.Program.Description,
		Summary:           p.Summary,
		Datetime:          p.GeneratedAt.UTC().Format(time.RFC3339),
		AudioFile:         p.AudioFile,
		Corners:           manifestCorners,
		ConversationNotes: notes,
		Casts:             casts,
	}
}
