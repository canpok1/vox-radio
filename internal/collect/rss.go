package collect

import (
	"context"
	"encoding/xml"
	"fmt"
	"net/http"

	"github.com/canpok1/vox-radio/internal/model"
)

type rssItem struct {
	Title       string `xml:"title"`
	Link        string `xml:"link"`
	Description string `xml:"description"`
}

type rssChannel struct {
	Items []rssItem `xml:"item"`
}

type rssFeed struct {
	Channel rssChannel `xml:"channel"`
}

func (c *Collector) fetchFeed(ctx context.Context, url string, maxItems int) ([]model.Article, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("fetch returned %d", resp.StatusCode)
	}

	var feed rssFeed
	if err := xml.NewDecoder(resp.Body).Decode(&feed); err != nil {
		return nil, fmt.Errorf("parse RSS: %w", err)
	}

	items := feed.Channel.Items
	if maxItems > 0 && len(items) > maxItems {
		items = items[:maxItems]
	}

	articles := make([]model.Article, 0, len(items))
	for _, item := range items {
		articles = append(articles, model.Article{
			URL:   item.Link,
			Title: item.Title,
			Body:  extractTextFromHTML(item.Description),
		})
	}
	return articles, nil
}
