package gather

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/canpok1/vox-radio/internal/model"
)

// fetchLinks reads a links file (one URL per line), fetches each URL as an article,
// and overwrites the DedupKey with LinksDedupKey(filePath, lineURL) for content-independent deduplication.
func (c *Gatherer) fetchLinks(ctx context.Context, filePath string) ([]model.Article, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("open links file %q: %w", filePath, err)
	}
	defer func() { _ = f.Close() }()

	articles := make([]model.Article, 0)
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		article, err := c.fetchArticle(ctx, line)
		if err != nil {
			return nil, fmt.Errorf("fetch links entry %q: %w", line, err)
		}
		article.DedupKey = LinksDedupKey(filePath, line)
		articles = append(articles, *article)
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("read links file %q: %w", filePath, err)
	}
	return articles, nil
}
