package collect

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/canpok1/vox-radio/internal/config"
	"github.com/canpok1/vox-radio/internal/model"
)

// injectionPatterns lists conservative regexps that match known prompt-injection phrases.
// Each pattern targets a specific injection technique; false positives are minimized
// by requiring explicit instruction-override vocabulary.
var injectionPatterns = []*regexp.Regexp{
	// Japanese: override previous/above instructions
	regexp.MustCompile(`(?i)(以前|これまで|上記)の指示を(無視|忘れ)`),
	// Japanese: "follow the next instruction"
	regexp.MustCompile(`次の指示に従って`),
	// Japanese: "you are now ..." identity override
	regexp.MustCompile(`あなたは(今|これ)から`),
	// Japanese: ignore output format
	regexp.MustCompile(`出力(形式)?を無視`),
	// English: ignore/disregard/forget instructions
	regexp.MustCompile(`(?i)(ignore|disregard|forget).{0,30}(above|previous|prior|all previous) instructions`),
	// English: "you are now" identity override
	regexp.MustCompile(`(?i)you are now `),
	// Role markers at the start of a word (system:, assistant:, <system>)
	regexp.MustCompile(`(?i)\bsystem:`),
	regexp.MustCompile(`(?i)\bassistant:`),
	regexp.MustCompile(`(?i)<system>`),
}

// invisibleCharRanges holds ranges of rune values to remove.
// Kept as sorted pairs [lo, hi] (inclusive on both ends).
var invisibleCharRanges = [][2]rune{
	{0x0001, 0x0008}, // C0 controls excluding \t (0x09)
	{0x000B, 0x000C}, // vertical tab, form feed
	{0x000E, 0x001F}, // remaining C0 controls (excluding \n=0x0A, \t=0x09)
	{0x007F, 0x009F}, // DEL + C1 controls
	{0x200B, 0x200D}, // zero-width space, ZWNJ, ZWJ
	{0x202A, 0x202E}, // bidi embedding/override controls
	{0x2066, 0x2069}, // bidi isolate controls
	{0xFEFF, 0xFEFF}, // BOM / zero-width no-break space
}

// removeInvisibleChars strips invisible and potentially-obfuscating characters.
// Newline (\n, 0x0A) and tab (\t, 0x09) are preserved.
func removeInvisibleChars(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	for _, r := range s {
		if !isInvisible(r) {
			b.WriteRune(r)
		}
	}
	return b.String()
}

func isInvisible(r rune) bool {
	for _, pair := range invisibleCharRanges {
		if r >= pair[0] && r <= pair[1] {
			return true
		}
	}
	return false
}

// truncateBody limits body to at most maxChars runes.
func truncateBody(body string, maxChars int) string {
	runes := []rune(body)
	if len(runes) <= maxChars {
		return body
	}
	return string(runes[:maxChars])
}

// containsInjectionPattern returns the matched text if s contains a
// known prompt-injection phrase, or empty string if none matched.
func containsInjectionPattern(s string) string {
	for _, re := range injectionPatterns {
		if match := re.FindString(s); match != "" {
			return match
		}
	}
	return ""
}

// sanitizeArticle cleans a in-place and returns (flagged, error).
// flagged is true when an injection pattern was detected in any field.
// On on_detect=error the function returns an error immediately on the first detection.
// On on_detect=exclude the caller is responsible for excluding the article entirely;
// this function does NOT modify any field on detection.
// Both Description (feed-derived) and Body (directly fetched) are sanitized.
// max_body_chars applies to each field independently.
func sanitizeArticle(a *model.Article, policy config.PromptInjectionConfig) (bool, error) {
	maxChars := policy.EffectiveMaxBodyChars()

	// Phase 1: remove invisible chars from all text fields
	a.Title = removeInvisibleChars(a.Title)
	a.Description = removeInvisibleChars(a.Description)
	a.Body = removeInvisibleChars(a.Body)
	a.Source = removeInvisibleChars(a.Source)
	a.Author = removeInvisibleChars(a.Author)

	// Phase 2: truncate description and body independently
	a.Description = truncateBody(a.Description, maxChars)
	a.Body = truncateBody(a.Body, maxChars)

	// Phase 3: detect injection patterns field by field
	fields := []struct {
		name  string
		value string
	}{
		{"Title", a.Title},
		{"Description", a.Description},
		{"Body", a.Body},
		{"Source", a.Source},
		{"Author", a.Author},
	}

	flagged := false
	onDetect := policy.EffectiveOnDetect()
	for _, f := range fields {
		if pat := containsInjectionPattern(f.value); pat != "" {
			flagged = true
			if onDetect == config.OnDetectError {
				return true, fmt.Errorf("prompt injection detected in article %s field %s (matched: %s)", a.URL, f.name, pat)
			}
		}
	}
	return flagged, nil
}
