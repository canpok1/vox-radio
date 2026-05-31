package collect

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/canpok1/vox-radio/internal/config"
	"github.com/canpok1/vox-radio/internal/model"
)

// Collector fetches articles from RSS/Atom feeds and individual URLs.
type Collector struct {
	client *http.Client
	logger *slog.Logger
}

// Option configures a Collector.
type Option func(*Collector)

// WithLogger sets the logger used for WARN messages.
func WithLogger(l *slog.Logger) Option {
	return func(c *Collector) { c.logger = l }
}

// New creates a Collector. If client is nil, http.DefaultClient is used.
func New(client *http.Client, opts ...Option) *Collector {
	if client == nil {
		client = http.DefaultClient
	}
	c := &Collector{
		client: client,
		logger: slog.Default(),
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

// Run collects articles from all feeds and individual URLs in cfg.
func (c *Collector) Run(ctx context.Context, cfg config.FeedsConfig) ([]model.Article, error) {
	articles := make([]model.Article, 0)

	for _, feed := range cfg.Feeds {
		items, err := c.fetchFeed(ctx, feed.URL, feed.MaxItems)
		if err != nil {
			return nil, fmt.Errorf("fetch feed %s: %w", feed.URL, err)
		}
		articles = append(articles, items...)
	}

	for _, u := range cfg.Articles {
		article, err := c.fetchArticle(ctx, u)
		if err != nil {
			return nil, fmt.Errorf("fetch article %s: %w", u, err)
		}
		articles = append(articles, *article)
	}

	return articles, nil
}

// RunAll collects articles per corner, skipping corners with no source.
func (c *Collector) RunAll(ctx context.Context, corners []config.CornerConfig) (model.Articles, error) {
	result := make([]model.CornerArticles, 0, len(corners))

	for _, corner := range corners {
		if corner.Source == nil {
			continue
		}
		articles, err := c.Run(ctx, config.FeedsConfig{
			Feeds:    corner.Source.Feeds,
			Articles: corner.Source.Articles,
		})
		if err != nil {
			return model.Articles{}, fmt.Errorf("collect corner %q: %w", corner.Title, err)
		}
		result = append(result, model.CornerArticles{
			CornerTitle: corner.Title,
			Articles:    articles,
		})
	}

	return model.Articles{Corners: result}, nil
}
