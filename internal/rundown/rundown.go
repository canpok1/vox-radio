package rundown

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/canpok1/vox-radio/internal/config"
	"github.com/canpok1/vox-radio/internal/model"
	sel "github.com/canpok1/vox-radio/internal/rundown/select"
	"github.com/canpok1/vox-radio/internal/script/summarize"
)

// Rundowner generates a Rundown from collected articles.
type Rundowner interface {
	Run(ctx context.Context, corners []config.CornerConfig, articles model.Articles) (model.Rundown, error)
}

// ArticleFetcher fetches the full body text of an article by URL.
type ArticleFetcher interface {
	FetchFullText(ctx context.Context, url string) (string, error)
}

// Option configures an LLMRundowner.
type Option func(*LLMRundowner)

// WithLogger sets the logger used for log output.
func WithLogger(l *slog.Logger) Option {
	return func(r *LLMRundowner) { r.logger = l }
}

// LLMRundowner uses Selector + Summarizer to produce a Rundown.
type LLMRundowner struct {
	selector     sel.Selector
	summarizer   summarize.Summarizer
	fetcher      ArticleFetcher
	excludedURLs map[string]struct{}
	logger       *slog.Logger
}

// NewLLMRundowner creates a LLMRundowner.
// fetcher may be nil (skips full-text fetch).
// excludedURLs is the set of article URLs to exclude before selection (nil = no exclusion).
func NewLLMRundowner(selector sel.Selector, summarizer summarize.Summarizer, fetcher ArticleFetcher, excludedURLs []string, opts ...Option) *LLMRundowner {
	excluded := make(map[string]struct{}, len(excludedURLs))
	for _, u := range excludedURLs {
		excluded[u] = struct{}{}
	}
	r := &LLMRundowner{
		selector:     selector,
		summarizer:   summarizer,
		fetcher:      fetcher,
		excludedURLs: excluded,
		logger:       slog.Default(),
	}
	for _, opt := range opts {
		opt(r)
	}
	r.logger = r.logger.With("step", "rundown")
	return r
}

func (r *LLMRundowner) Run(ctx context.Context, corners []config.CornerConfig, articles model.Articles) (model.Rundown, error) {
	articleMap := articles.CornerMap()
	rundownCorners := make([]model.RundownCorner, 0, len(corners))

	for _, corner := range corners {
		cornerArticles := articleMap[corner.Title]

		filtered := make([]model.Article, 0, len(cornerArticles))
		for _, a := range cornerArticles {
			if _, excluded := r.excludedURLs[a.URL]; !excluded {
				filtered = append(filtered, a)
			}
		}
		if n := len(cornerArticles) - len(filtered); n > 0 {
			r.logger.Info("excluded past articles", "corner", corner.Title, "count", n)
		}
		cornerArticles = filtered

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
			if r.fetcher != nil {
				if fullText, err := r.fetcher.FetchFullText(ctx, url); err != nil {
					r.logger.Warn("full text fetch failed, using feed body", "url", url, "err", err)
				} else if fullText == "" {
					r.logger.Warn("full text fetch returned empty body, using feed body", "url", url)
				} else {
					a.Body = fullText
				}
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
