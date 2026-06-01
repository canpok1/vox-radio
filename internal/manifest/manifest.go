package manifest

import (
	"time"

	"github.com/canpok1/vox-radio/internal/config"
	"github.com/canpok1/vox-radio/internal/model"
)

// Build constructs a Manifest from program config, corners, rundown, generation time, episode summary, optional corner summaries, and conversation notes.
// cornerSummaries maps corner title to its LLM-generated summary; nil means no corner summaries.
// conversationNotes contains non-rundown conversation information extracted from the episode.
func Build(program config.ProgramConfig, corners []config.CornerConfig, rundown model.Rundown, audioFile string, generatedAt time.Time, summary string, cornerSummaries map[string]model.CornerSummary, conversationNotes []model.ConversationNote) model.Manifest {
	cornerMap := rundown.CornerMap()
	manifestCorners := make([]model.ManifestCorner, 0, len(corners))
	for _, c := range corners {
		rc := cornerMap[c.Title]
		refs := make([]model.ArticleRef, 0, len(rc.Articles))
		for _, a := range rc.Articles {
			refs = append(refs, model.ArticleRef{Title: a.Title, URL: a.URL})
		}
		cs := cornerSummaries[c.Title]
		points := cs.Points
		if points == nil {
			points = make([]string, 0)
		}
		manifestCorners = append(manifestCorners, model.ManifestCorner{
			Title:    c.Title,
			Summary:  cs.Summary,
			Points:   points,
			Articles: refs,
		})
	}
	notes := conversationNotes
	if notes == nil {
		notes = make([]model.ConversationNote, 0)
	}
	return model.Manifest{
		Title:             program.Title,
		Description:       program.Description,
		Summary:           summary,
		Datetime:          generatedAt.UTC().Format(time.RFC3339),
		AudioFile:         audioFile,
		Corners:           manifestCorners,
		ConversationNotes: notes,
	}
}
