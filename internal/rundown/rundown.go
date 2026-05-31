package rundown

import (
	"context"
	"fmt"

	"github.com/canpok1/vox-radio/internal/config"
	"github.com/canpok1/vox-radio/internal/model"
	sel "github.com/canpok1/vox-radio/internal/rundown/select"
	"github.com/canpok1/vox-radio/internal/script/summarize"
)

// Rundowner generates a Rundown from collected articles.
type Rundowner interface {
	Run(ctx context.Context, corners []config.CornerConfig, articles model.Articles) (model.Rundown, error)
}

// LLMRundowner uses Selector + Summarizer to produce a Rundown.
type LLMRundowner struct {
	selector   sel.Selector
	summarizer summarize.Summarizer
}

func NewLLMRundowner(selector sel.Selector, summarizer summarize.Summarizer) *LLMRundowner {
	return &LLMRundowner{selector: selector, summarizer: summarizer}
}

func (r *LLMRundowner) Run(ctx context.Context, corners []config.CornerConfig, articles model.Articles) (model.Rundown, error) {
	articleMap := articles.CornerMap()
	rundownCorners := make([]model.RundownCorner, 0, len(corners))

	for _, corner := range corners {
		cornerArticles := articleMap[corner.Title]
		if len(cornerArticles) == 0 {
			rundownCorners = append(rundownCorners, model.RundownCorner{
				Title:    corner.Title,
				Flow:     "",
				Articles: make([]model.RundownArticle, 0),
			})
			continue
		}

		selected, err := r.selector.Select(ctx, corner, cornerArticles)
		if err != nil {
			return model.Rundown{}, fmt.Errorf("select corner %q: %w", corner.Title, err)
		}

		// Build URL→Article index for fast lookup
		articleByURL := make(map[string]model.Article, len(cornerArticles))
		for _, a := range cornerArticles {
			articleByURL[a.URL] = a
		}

		rdArticles := make([]model.RundownArticle, 0, len(selected.SelectedURLs))
		for _, url := range selected.SelectedURLs {
			a, ok := articleByURL[url]
			if !ok {
				continue
			}
			sum, err := r.summarizer.Summarize(ctx, a)
			if err != nil {
				return model.Rundown{}, fmt.Errorf("summarize %q: %w", url, err)
			}
			points := sum.Points
			if points == nil {
				points = make([]string, 0)
			}
			rdArticles = append(rdArticles, model.RundownArticle{
				URL:     a.URL,
				Title:   a.Title,
				Summary: sum.Summary,
				Points:  points,
			})
		}

		rundownCorners = append(rundownCorners, model.RundownCorner{
			Title:    corner.Title,
			Flow:     selected.Flow,
			Articles: rdArticles,
		})
	}

	return model.Rundown{Corners: rundownCorners}, nil
}
