package collect

import (
	"context"
	"fmt"

	"github.com/mmcdole/gofeed"

	"github.com/canpok1/vox-radio/internal/model"
)

func (c *Collector) fetchFeed(ctx context.Context, url string, maxItems int, excluded map[string]struct{}) ([]model.Article, error) {
	fp := gofeed.NewParser()
	fp.Client = c.client

	feed, err := fp.ParseURLWithContext(url, ctx)
	if err != nil {
		return nil, fmt.Errorf("parse feed: %w", err)
	}

	articles := make([]model.Article, 0, len(feed.Items))
	for _, item := range feed.Items {
		if _, skip := excluded[item.Link]; skip {
			continue
		}
		body := item.Content
		if body == "" {
			body = item.Description
		}
		articles = append(articles, model.Article{
			URL:   item.Link,
			Title: item.Title,
			Body:  extractTextFromHTML(body),
		})
		if maxItems > 0 && len(articles) >= maxItems {
			break
		}
	}

	if len(articles) == 0 {
		c.logger.Warn("feed returned 0 items", "url", url)
		return make([]model.Article, 0), nil
	}

	if maxItems > 0 && len(articles) < maxItems {
		c.logger.Warn("フィードの未使用記事が不足", "url", url, "got", len(articles), "want", maxItems)
	}

	return articles, nil
}
