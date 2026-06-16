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
	"github.com/canpok1/vox-radio/internal/logging"
	"github.com/canpok1/vox-radio/internal/model"
)

// Collector fetches articles from RSS/Atom feeds and individual URLs.
type Collector struct {
	client *http.Client
	logger *slog.Logger
	loc    *time.Location
	policy config.PromptInjectionConfig
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

// WithSanitizePolicy sets the prompt-injection sanitize policy applied to each fetched article.
func WithSanitizePolicy(p config.PromptInjectionConfig) Option {
	return func(c *Collector) { c.policy = p }
}

// New creates a Collector. If client is nil, a client with retry-enabled
// transport (exponential backoff on 5xx/429) is used. The default client
// also supports file:// URLs for loading locally saved feed/article files.
func New(client *http.Client, opts ...Option) *Collector {
	if client == nil {
		base := http.DefaultTransport.(*http.Transport).Clone()
		base.RegisterProtocol("file", http.NewFileTransport(http.Dir("/")))
		client = &http.Client{Transport: httpretry.NewTransport(base)}
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
		for i := range items {
			flagged, err := c.applySanitize(&items[i])
			if err != nil {
				return nil, err
			}
			if !flagged {
				articles = append(articles, items[i])
			}
		}
	}

	for _, u := range cfg.Articles {
		article, err := c.fetchArticle(ctx, u)
		if err != nil {
			return nil, fmt.Errorf("fetch article %s: %w", u, err)
		}
		flagged, err := c.applySanitize(article)
		if err != nil {
			return nil, err
		}
		if !flagged {
			articles = append(articles, *article)
		}
	}

	return articles, nil
}

// RunAll collects articles per corner, skipping corners with no source.
// excludedDedupKeys is a list of DedupKeys to skip when fetching from feeds (nil means no exclusion).
func (c *Collector) RunAll(ctx context.Context, corners []config.CornerConfig, excludedDedupKeys []string) (model.Articles, error) {
	logger := c.logger.With("step", "collect")

	var excluded map[string]struct{}
	if len(excludedDedupKeys) > 0 {
		excluded = make(map[string]struct{}, len(excludedDedupKeys))
		for _, k := range excludedDedupKeys {
			excluded[k] = struct{}{}
		}
	}

	filtered := make([]config.CornerConfig, 0, len(corners))
	for _, corner := range corners {
		if corner.Source != nil {
			filtered = append(filtered, corner)
		}
	}

	done := logging.StartStep(logger, "開始")

	result := make([]model.CornerArticles, 0, len(filtered))
	totalArticles := 0

	for i, corner := range filtered {
		logger.Info(fmt.Sprintf("コーナー「%s」を収集中 (%d/%d)", corner.Title, i+1, len(filtered)))

		articles, err := c.Run(ctx, *corner.Source, excluded)
		if err != nil {
			return model.Articles{}, fmt.Errorf("collect corner %q: %w", corner.Title, err)
		}
		for _, a := range articles {
			logger.Debug("記事取得", "title", a.Title, "url", a.URL, "chars", utf8.RuneCountInString(a.Text()))
		}
		totalArticles += len(articles)
		result = append(result, model.CornerArticles{
			CornerTitle: corner.Title,
			Articles:    articles,
		})
	}

	done(fmt.Sprintf("%d記事 / %dコーナー", totalArticles, len(result)))

	return model.Articles{Corners: result}, nil
}

// applySanitize applies prompt-injection sanitization to a.
// Returns (true, nil) when an injection pattern is detected under on_detect=exclude (caller must exclude the article).
// Returns (true, err) when on_detect=error and injection is detected.
func (c *Collector) applySanitize(a *model.Article) (bool, error) {
	flagged, err := sanitizeArticle(a, c.policy)
	if err != nil {
		return true, err
	}
	if flagged {
		c.logger.Warn("prompt injection pattern detected; article excluded", "url", a.URL)
	}
	return flagged, nil
}
