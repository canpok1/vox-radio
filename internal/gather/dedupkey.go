package gather

import (
	"crypto/sha256"
	"encoding/hex"
	"regexp"
	"strings"
)

var whitespaceRe = regexp.MustCompile(`\s+`)

// dedupKey computes a content-based deduplication key for an article.
// The key is "sha256:<hex>" where the hash is sha256(namespace + "\x00" + material).
// namespace isolates the key domain (feed URL or page URL) to avoid cross-feed collisions.
// material is the identifying content (guid or normalized text).
func dedupKey(namespace, material string) string {
	h := sha256.Sum256([]byte(namespace + "\x00" + material))
	return "sha256:" + hex.EncodeToString(h[:])
}

// FeedDedupKey computes the DedupKey for a feed item.
// material priority: guid → link → normalizeContent(title, body).
// feedURL is the namespace (feed's source URL).
func FeedDedupKey(feedURL, guid, link, title, body string) string {
	material := guid
	if material == "" {
		material = link
	}
	if material == "" {
		material = normalizeContent(title, body)
	}
	return dedupKey(feedURL, material)
}

// LinksDedupKey computes the DedupKey for an entry in a links-type source file.
// namespace=filePath, material=lineURL (stable regardless of page content).
func LinksDedupKey(filePath, lineURL string) string {
	return dedupKey(filePath, lineURL)
}

// TextDedupKey computes the DedupKey for a text-type source file.
// namespace=filePath, material=normalizeContent(title, body) (content-based; changes when content changes).
func TextDedupKey(filePath, title, body string) string {
	return dedupKey(filePath, normalizeContent(title, body))
}

// HTMLDedupKey computes the DedupKey for an HTML article page.
// pageURL is both the namespace and the stable identifier.
func HTMLDedupKey(pageURL, title, body string) string {
	return dedupKey(pageURL, normalizeContent(title, body))
}

// normalizeContent returns a canonical form of title+body for use as dedup material.
// Leading/trailing whitespace is trimmed and runs of whitespace are collapsed to a single space.
func normalizeContent(title, body string) string {
	normTitle := whitespaceRe.ReplaceAllString(strings.TrimSpace(title), " ")
	normBody := whitespaceRe.ReplaceAllString(strings.TrimSpace(body), " ")
	return normTitle + "\n" + normBody
}
