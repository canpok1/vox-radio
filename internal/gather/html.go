package gather

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

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
	published := findPublished(doc, c.loc)
	return &model.Article{
		DedupKey:  HTMLDedupKey(rawURL, title, body),
		URL:       rawURL,
		Title:     title,
		Body:      body,
		Published: published,
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

// findPublished extracts the article published date from HTML meta tags.
// Priority: OGP article:published_time → JSON-LD datePublished.
// Returns RFC3339 in loc, or empty string if not found or unparseable.
func findPublished(doc *html.Node, loc *time.Location) string {
	if raw := findMetaContent(doc, "article:published_time"); raw != "" {
		if t, err := time.Parse(time.RFC3339, raw); err == nil {
			return t.In(loc).Format(time.RFC3339)
		}
	}
	if raw := findJSONLDPublished(doc); raw != "" {
		if t, err := time.Parse(time.RFC3339, raw); err == nil {
			return t.In(loc).Format(time.RFC3339)
		}
	}
	return ""
}

// findMetaContent returns the content attribute of the first <meta property="prop"> element.
func findMetaContent(doc *html.Node, property string) string {
	var walk func(*html.Node) string
	walk = func(n *html.Node) string {
		if n.Type == html.ElementNode && n.Data == "meta" {
			var prop, content string
			for _, a := range n.Attr {
				switch a.Key {
				case "property":
					prop = a.Val
				case "content":
					content = a.Val
				}
			}
			if prop == property && content != "" {
				return content
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			if v := walk(c); v != "" {
				return v
			}
		}
		return ""
	}
	return walk(doc)
}

// findJSONLDPublished extracts datePublished from <script type="application/ld+json">.
// Supports both flat objects and @graph arrays.
func findJSONLDPublished(doc *html.Node) string {
	var walk func(*html.Node) string
	walk = func(n *html.Node) string {
		if n.Type == html.ElementNode && n.Data == "script" {
			isJSONLD := false
			for _, a := range n.Attr {
				if a.Key == "type" && a.Val == "application/ld+json" {
					isJSONLD = true
					break
				}
			}
			if isJSONLD && n.FirstChild != nil && n.FirstChild.Type == html.TextNode {
				if v := extractDatePublished(n.FirstChild.Data); v != "" {
					return v
				}
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			if v := walk(c); v != "" {
				return v
			}
		}
		return ""
	}
	return walk(doc)
}

func extractDatePublished(jsonText string) string {
	var obj map[string]json.RawMessage
	if err := json.Unmarshal([]byte(jsonText), &obj); err != nil {
		return ""
	}
	if raw, ok := obj["datePublished"]; ok {
		var s string
		if json.Unmarshal(raw, &s) == nil && s != "" {
			return s
		}
	}
	if raw, ok := obj["@graph"]; ok {
		var graph []map[string]json.RawMessage
		if json.Unmarshal(raw, &graph) == nil {
			for _, entry := range graph {
				if dp, ok := entry["datePublished"]; ok {
					var s string
					if json.Unmarshal(dp, &s) == nil && s != "" {
						return s
					}
				}
			}
		}
	}
	return ""
}

func extractTextFromHTML(s string) string {
	doc, err := html.Parse(strings.NewReader(s))
	if err != nil {
		return s
	}
	return findBody(doc)
}
