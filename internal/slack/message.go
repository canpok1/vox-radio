package slack

import (
	"strings"
	"unicode/utf8"

	slackgo "github.com/slack-go/slack"

	"github.com/canpok1/vox-radio/internal/model"
	"github.com/canpok1/vox-radio/internal/render"
)

const maxSectionRunes = 3000

func renderTemplate(manifest model.Manifest, tmplText string) (string, error) {
	result, err := render.Render(tmplText, manifest)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(result), nil
}

// BuildParent renders the parent template (mp3 upload initial comment).
func BuildParent(manifest model.Manifest, tmplText string) (string, error) {
	return renderTemplate(manifest, tmplText)
}

// BuildFallback renders the fallback template (notification plain text).
func BuildFallback(manifest model.Manifest, tmplText string) (string, error) {
	return renderTemplate(manifest, tmplText)
}

// BuildThread renders the thread body template to a single string.
func BuildThread(manifest model.Manifest, tmplText string) (string, error) {
	return renderTemplate(manifest, tmplText)
}

// BuildAudioTitle returns the Slack file title for the audio upload.
// Uses episode_title when set; falls back to title.
func BuildAudioTitle(manifest model.Manifest) string {
	if manifest.EpisodeTitle != "" {
		return manifest.EpisodeTitle
	}
	return manifest.Title
}

// SplitIntoSectionBlocks splits text into Slack Section blocks of at most
// maxSectionRunes runes each. Splits occur at newline boundaries.
// Returns nil when text is empty.
func SplitIntoSectionBlocks(text string) []slackgo.Block {
	if text == "" {
		return nil
	}

	lines := strings.Split(text, "\n")
	var blocks []slackgo.Block
	var current strings.Builder
	var currentRunes int

	for _, line := range lines {
		lineWithNL := line + "\n"
		lineRunes := utf8.RuneCountInString(lineWithNL)
		if currentRunes+lineRunes > maxSectionRunes && current.Len() > 0 {
			blocks = append(blocks, newSectionBlock(strings.TrimRight(current.String(), "\n")))
			current.Reset()
			currentRunes = 0
		}
		current.WriteString(lineWithNL)
		currentRunes += lineRunes
	}

	if s := strings.TrimRight(current.String(), "\n"); s != "" {
		blocks = append(blocks, newSectionBlock(s))
	}

	return blocks
}

func newSectionBlock(text string) *slackgo.SectionBlock {
	return slackgo.NewSectionBlock(
		slackgo.NewTextBlockObject(slackgo.MarkdownType, text, false, false),
		nil, nil,
	)
}
