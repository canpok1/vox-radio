package slack

import (
	"strconv"
	"strings"

	slackgo "github.com/slack-go/slack"

	"github.com/canpok1/vox-radio/internal/model"
)

// BuildHeader builds the initial comment text for the parent mp3 upload message.
func BuildHeader(manifest model.Manifest, tmpl MessageTemplate) string {
	s := tmpl.Header
	s = replacePlaceholders(s, manifest)

	// remove empty episode title quotes
	s = strings.ReplaceAll(s, "「」", "")

	// remove 第0回 segment when episode number is 0
	if manifest.EpisodeNumber == 0 {
		s = removeEpisodeSegment(s)
	}

	return strings.TrimSpace(s)
}

// BuildFallback builds the fallback plain text for thread reply notifications.
func BuildFallback(manifest model.Manifest, tmpl MessageTemplate) string {
	s := tmpl.Fallback
	s = replacePlaceholders(s, manifest)
	return strings.TrimSpace(s)
}

// BuildAudioTitle returns the Slack file title for the audio upload.
// Uses episode_title when set; falls back to title.
func BuildAudioTitle(manifest model.Manifest) string {
	if manifest.EpisodeTitle != "" {
		return manifest.EpisodeTitle
	}
	return manifest.Title
}

// BuildThreadBlocks builds the Block Kit blocks and fallback text for the thread reply.
// Returns nil blocks when both summary and corners are empty (thread should be skipped).
func BuildThreadBlocks(manifest model.Manifest, tmpl MessageTemplate) ([]slackgo.Block, string) {
	var blocks []slackgo.Block

	if manifest.Summary != "" {
		summaryText := replacePlaceholders(tmpl.Summary, manifest)
		blocks = append(blocks, slackgo.NewSectionBlock(
			slackgo.NewTextBlockObject(slackgo.MarkdownType, summaryText, false, false),
			nil, nil,
		))
	}

	if len(manifest.Corners) > 0 {
		if manifest.Summary != "" {
			blocks = append(blocks, slackgo.NewDividerBlock())
		}
		for _, corner := range manifest.Corners {
			text := buildCornerText(corner, tmpl)
			blocks = append(blocks, slackgo.NewSectionBlock(
				slackgo.NewTextBlockObject(slackgo.MarkdownType, text, false, false),
				nil, nil,
			))
		}
	}

	if len(blocks) == 0 {
		return nil, ""
	}

	fallback := BuildFallback(manifest, tmpl)
	return blocks, fallback
}

func buildCornerText(corner model.ManifestCorner, tmpl MessageTemplate) string {
	articlesText := buildArticlesText(corner.Articles, tmpl.Article)

	s := tmpl.Corner
	s = strings.ReplaceAll(s, "{corner_title}", corner.Title)
	s = strings.ReplaceAll(s, "{corner_summary}", corner.Summary)
	s = strings.ReplaceAll(s, "{articles}", articlesText)

	// remove empty lines
	lines := strings.Split(s, "\n")
	var kept []string
	for _, line := range lines {
		if strings.TrimSpace(line) != "" {
			kept = append(kept, line)
		}
	}
	return strings.Join(kept, "\n")
}

func buildArticlesText(articles []model.ArticleRef, articleTmpl string) string {
	if len(articles) == 0 {
		return ""
	}
	var parts []string
	for _, a := range articles {
		line := articleTmpl
		if a.URL == "" {
			// Remove Slack link wrappers "<{url}|...>" → inner text only
			const open = "<{url}|"
			for {
				before, after, found := strings.Cut(line, open)
				if !found {
					break
				}
				inner, tail, ok := strings.Cut(after, ">")
				if !ok {
					break
				}
				line = before + inner + tail
			}
			line = strings.ReplaceAll(line, "{url}", "")
		} else {
			line = strings.ReplaceAll(line, "{url}", a.URL)
		}
		line = strings.ReplaceAll(line, "{title}", a.Title)
		parts = append(parts, line)
	}
	return strings.Join(parts, "\n")
}

// replacePlaceholders replaces {placeholder} tokens with manifest field values.
func replacePlaceholders(s string, manifest model.Manifest) string {
	s = strings.ReplaceAll(s, "{title}", manifest.Title)
	s = strings.ReplaceAll(s, "{episode_number}", strconv.Itoa(manifest.EpisodeNumber))
	s = strings.ReplaceAll(s, "{episode_title}", manifest.EpisodeTitle)
	s = strings.ReplaceAll(s, "{description}", manifest.Description)
	s = strings.ReplaceAll(s, "{summary}", manifest.Summary)
	s = strings.ReplaceAll(s, "{datetime}", manifest.Datetime)
	s = strings.ReplaceAll(s, "{audio_file}", manifest.AudioFile)
	s = strings.ReplaceAll(s, "{credit}", strings.Join(manifest.Credits, "\n"))
	return s
}

// removeEpisodeSegment removes the 第0回 portion from a header string.
// TrimSpace is applied by the caller (BuildHeader).
func removeEpisodeSegment(s string) string {
	return strings.ReplaceAll(s, "第0回", "")
}
