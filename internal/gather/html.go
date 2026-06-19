package gather

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"golang.org/x/net/html"

	"github.com/canpok1/vox-radio/internal/model"
)

func (c *Gatherer) fetchArticle(ctx context.Context, rawURL string) (*model.Article, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
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

	doc, err := html.Parse(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("parse HTML: %w", err)
	}

	title := findTitle(doc)
	body := findBody(doc)
	return &model.Article{
		DedupKey: HTMLDedupKey(rawURL, title, body),
		URL:      rawURL,
		Title:    title,
		Body:     body,
	}, nil
}

func findTitle(doc *html.Node) string {
	var walk func(*html.Node) string
	walk = func(n *html.Node) string {
		if n.Type == html.ElementNode && n.Data == "title" {
			if n.FirstChild != nil && n.FirstChild.Type == html.TextNode {
				return strings.TrimSpace(n.FirstChild.Data)
			}
			return ""
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			if t := walk(c); t != "" {
				return t
			}
		}
		return ""
	}
	return walk(doc)
}

// findBody extracts body text from the first <article>, <main>, or <body> element.
func findBody(doc *html.Node) string {
	for _, tag := range []string{"article", "main", "body"} {
		if n := findElement(doc, tag); n != nil {
			return extractText(n)
		}
	}
	return ""
}

func findElement(n *html.Node, tag string) *html.Node {
	if n.Type == html.ElementNode && n.Data == tag {
		return n
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if found := findElement(c, tag); found != nil {
			return found
		}
	}
	return nil
}

func extractText(n *html.Node) string {
	var sb strings.Builder
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.TextNode {
			text := strings.TrimSpace(n.Data)
			if text != "" {
				if sb.Len() > 0 {
					sb.WriteString("\n")
				}
				sb.WriteString(text)
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(n)
	return sb.String()
}

func extractTextFromHTML(s string) string {
	doc, err := html.Parse(strings.NewReader(s))
	if err != nil {
		return s
	}
	return findBody(doc)
}
