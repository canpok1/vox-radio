package collect

import (
	"context"
	"fmt"
	"strings"
	"time"

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

	source := feed.Title

	articles := make([]model.Article, 0, len(feed.Items))
	for _, item := range feed.Items {
		body := item.Content
		if body == "" {
			body = item.Description
		}

		// GUID が非空なら DedupKey に bodyText 不要 → 除外チェック後まで HTML パースを遅延
		var bodyText string
		if item.GUID == "" {
			bodyText = extractTextFromHTML(body)
		}
		key := FeedDedupKey(url, item.GUID, item.Title, bodyText)

		if _, skip := excluded[key]; skip {
			continue
		}

		// GUID 非空で遅延した場合はここで HTML パース
		if item.GUID != "" {
			bodyText = extractTextFromHTML(body)
		}
		articles = append(articles, model.Article{
			DedupKey:  key,
			URL:       item.Link,
			Title:     item.Title,
			Body:      bodyText,
			Source:    source,
			Author:    extractAuthor(item),
			Published: extractPublished(item, c.loc),
		})
		if maxItems > 0 && len(articles) >= maxItems {
			break
		}
	}

	if len(articles) == 0 {
		c.logger.Warn("feed returned 0 items", "url", url)
		return articles, nil
	}

	if maxItems > 0 && len(articles) < maxItems {
		c.logger.Warn("フィードの未使用記事が不足", "url", url, "got", len(articles), "want", maxItems)
	}

	return articles, nil
}

// extractAuthor extracts the author name from a feed item.
// Priority: item.Authors[0].Name → item.Author.Name.
// Returns empty string if the result contains "@" (email format) or is blank.
func extractAuthor(item *gofeed.Item) string {
	var name string
	if len(item.Authors) > 0 {
		name = strings.TrimSpace(item.Authors[0].Name)
	}
	if name == "" && item.Author != nil {
		name = strings.TrimSpace(item.Author.Name)
	}
	if strings.Contains(name, "@") {
		return ""
	}
	return name
}

// extractPublished returns the item's published time as RFC3339 in loc.
// Returns empty string if PublishedParsed is nil.
func extractPublished(item *gofeed.Item, loc *time.Location) string {
	if item.PublishedParsed == nil {
		return ""
	}
	return item.PublishedParsed.In(loc).Format(time.RFC3339)
}
