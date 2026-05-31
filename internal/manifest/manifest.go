package manifest

import (
	"time"

	"github.com/canpok1/vox-radio/internal/config"
	"github.com/canpok1/vox-radio/internal/model"
)

// Build constructs a Manifest from program config, corners, collected articles, and generation time.
func Build(program config.ProgramConfig, corners []config.CornerConfig, articles model.Articles, audioFile string, generatedAt time.Time) model.Manifest {
	cornerMap := articles.CornerMap()
	manifestCorners := make([]model.ManifestCorner, 0, len(corners))
	for _, c := range corners {
		cornerArticles := cornerMap[c.Title]
		refs := make([]model.ArticleRef, 0, len(cornerArticles))
		for _, a := range cornerArticles {
			refs = append(refs, model.ArticleRef{Title: a.Title, URL: a.URL})
		}
		manifestCorners = append(manifestCorners, model.ManifestCorner{
			Title:    c.Title,
			Articles: refs,
		})
	}
	return model.Manifest{
		Title:       program.Title,
		Description: program.Description,
		Datetime:    generatedAt.UTC().Format(time.RFC3339),
		AudioFile:   audioFile,
		Corners:     manifestCorners,
	}
}
