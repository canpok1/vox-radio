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
	Assets            config.AssetsConfig
	Characters        map[string]config.CharacterConfig
	Lines             *model.ScriptLines
	Script            *model.Script
}

// Build constructs a Manifest from BuildParams.
// BuildParams.CornerSummaries maps corner title to its LLM-generated summary; nil means no corner summaries.
// BuildParams.ConversationNotes contains non-rundown conversation information extracted from the episode.
// BuildParams.EpisodeNumber 0 means unknown (omitted from manifest). BuildParams.EpisodeTitle "" means unknown (omitted).
// Credits are collected internally from Assets/Characters/Lines/Script/Rundown.Casts.
func Build(p BuildParams) model.Manifest {
	credits := CollectCredits(CreditParams{
		Assets:     p.Assets,
		Characters: p.Characters,
		Lines:      p.Lines,
		Script:     p.Script,
		Casts:      p.Rundown.Casts,
	})
	cornerMap := p.Rundown.CornerMap()
	manifestCorners := make([]model.ManifestCorner, 0, len(p.Corners))
	for _, c := range p.Corners {
		rc := cornerMap[c.ID]
		refs := make([]model.ArticleRef, 0, len(rc.Articles))
		for _, a := range rc.Articles {
			refs = append(refs, model.ArticleRef{DedupKey: a.DedupKey, Title: a.Title, URL: a.URL})
		}
		cs := p.CornerSummaries[c.Title]
		manifestCorners = append(manifestCorners, model.NewManifestCorner(
			c.ID, c.Title, cs.Summary, cs.Points, refs,
		))
	}
	return model.Manifest{
		Title:             p.Program.Title,
		EpisodeNumber:     p.EpisodeNumber,
		EpisodeTitle:      p.EpisodeTitle,
		Description:       p.Program.Description,
		Summary:           p.Summary,
		Datetime:          p.GeneratedAt.UTC().Format(time.RFC3339),
		AudioFile:         p.AudioFile,
		Corners:           manifestCorners,
		ConversationNotes: model.NonNil(p.ConversationNotes),
		Casts:             model.NonNil(p.Rundown.Casts),
		Credits:           model.NonNil(credits),
	}
}
