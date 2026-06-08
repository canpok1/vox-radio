package collect

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"time"
	"unicode/utf8"

	"github.com/canpok1/vox-radio/internal/config"
	"github.com/canpok1/vox-radio/internal/httpretry"
	"github.com/canpok1/vox-radio/internal/model"
)

// Collector fetches articles from RSS/Atom feeds and individual URLs.
type Collector struct {
	client *http.Client
	logger *slog.Logger
	loc    *time.Location
}

// Option configures a Collector.
type Option func(*Collector)

// WithLogger sets the logger used for WARN messages.
func WithLogger(l *slog.Logger) Option {
	return func(c *Collector) { c.logger = l }
}

// WithLocation sets the timezone used for converting article published times.
func WithLocation(loc *time.Location) Option {
	return func(c *Collector) { c.loc = loc }
}

// New creates a Collector. If client is nil, a client with retry-enabled
// transport (exponential backoff on 5xx/429) is used.
func New(client *http.Client, opts ...Option) *Collector {
	if client == nil {
		client = httpretry.NewClient(0)
	}
	c := &Collector{
		client: client,
		logger: slog.Default(),
		loc:    time.UTC,
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

// Run collects articles from all feeds and individual URLs in cfg.
// excluded is a set of URLs to skip when fetching from feeds (nil means no exclusion).
func (c *Collector) Run(ctx context.Context, cfg config.FeedsConfig, excluded map[string]struct{}) ([]model.Article, error) {
	articles := make([]model.Article, 0)

	for _, feed := range cfg.Feeds {
		items, err := c.fetchFeed(ctx, feed.URL, feed.MaxItems, excluded)
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

// FetchFullText fetches the full body text of the article at url.
func (c *Collector) FetchFullText(ctx context.Context, url string) (string, error) {
	article, err := c.fetchArticle(ctx, url)
	if err != nil {
		return "", err
	}
	return article.Body, nil
}

// RunAll collects articles per corner, skipping corners with no source.
// excludedURLs is a list of URLs to skip when fetching from feeds (nil means no exclusion).
func (c *Collector) RunAll(ctx context.Context, corners []config.CornerConfig, excludedURLs []string) (model.Articles, error) {
	logger := c.logger.With("step", "collect")
	start := time.Now()

	var excluded map[string]struct{}
	if len(excludedURLs) > 0 {
		excluded = make(map[string]struct{}, len(excludedURLs))
		for _, u := range excludedURLs {
			excluded[u] = struct{}{}
		}
	}

	filtered := make([]config.CornerConfig, 0, len(corners))
	for _, corner := range corners {
		if corner.Source != nil {
			filtered = append(filtered, corner)
		}
	}

	logger.Info("開始")

	result := make([]model.CornerArticles, 0, len(filtered))
	totalArticles := 0

	for i, corner := range filtered {
		logger.Info(fmt.Sprintf("コーナー「%s」を収集中 (%d/%d)", corner.Title, i+1, len(filtered)))

		articles, err := c.Run(ctx, config.FeedsConfig{
			Feeds:    corner.Source.Feeds,
			Articles: corner.Source.Articles,
		}, excluded)
		if err != nil {
			return model.Articles{}, fmt.Errorf("collect corner %q: %w", corner.Title, err)
		}
		for _, a := range articles {
			logger.Debug("記事取得", "title", a.Title, "url", a.URL, "chars", utf8.RuneCountInString(a.Body))
		}
		totalArticles += len(articles)
		result = append(result, model.CornerArticles{
			CornerTitle: corner.Title,
			Articles:    articles,
		})
	}

	logger.Info(fmt.Sprintf("完了 (%d記事 / %dコーナー, %.1fs)", totalArticles, len(result), time.Since(start).Seconds()))

	return model.Articles{Corners: result}, nil
}
