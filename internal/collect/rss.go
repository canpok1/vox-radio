package collect

import (
	"context"
	"fmt"

	"github.com/mmcdole/gofeed"

	"github.com/canpok1/vox-radio/internal/model"
)

func (c *Collector) fetchFeed(ctx context.Context, url string, maxItems int) ([]model.Article, error) {
	fp := gofeed.NewParser()
	fp.Client = c.client

	feed, err := fp.ParseURLWithContext(url, ctx)
	if err != nil {
		return nil, fmt.Errorf("parse feed: %w", err)
	}

	items := feed.Items
	if maxItems > 0 && len(items) > maxItems {
		items = items[:maxItems]
	}

	if len(items) == 0 {
		c.logger.Warn("feed returned 0 items", "url", url)
		return make([]model.Article, 0), nil
	}

	articles := make([]model.Article, 0, len(items))
	for _, item := range items {
		body := item.Content
		if body == "" {
			body = item.Description
		}
		articles = append(articles, model.Article{
			URL:   item.Link,
			Title: item.Title,
			Body:  extractTextFromHTML(body),
		})
	}
	return articles, nil
}
