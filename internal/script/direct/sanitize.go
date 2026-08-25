package direct

import (
	"strings"
	"unicode"
)

// Round brackets mark eye-only annotations. Full-width and half-width forms are pooled
// rather than paired, so a mixed pair (「キータ(Qiita）」) is removed too — the LLM writes
// the text, and nothing guarantees it keeps the widths consistent.
//
// Quotation brackets (「」『』) are deliberately absent: they carry article titles and
// quoted speech that are meant to be read aloud.
var (
	annotationOpeners = map[rune]bool{'（': true, '(': true}
	annotationClosers = map[rune]bool{'）': true, ')': true}
)

// sanitizeSpeechText removes eye-only annotations from a line that is about to be
// synthesized.
//
// The text handed to VOICEVOX is spoken verbatim — OpenJTalk pronounces what is inside
// brackets too (「キータ（Qiita）」→ キータ キューアイアイティーエー), so a line has no place to
// carry a note that is not heard. The reading-conversion LLM nevertheless emits the
// screen-oriented "reading（original spelling）" form, which the listener hears as the same
// word twice. Dropping bracketed runs enforces the invariant that synthesized text contains
// only what should be heard; supplementary information has to be written as dialogue.
//
// Only balanced pairs are removed, so an unmatched bracket cannot swallow the rest of the
// line. A line left with nothing pronounceable is returned unchanged, since a clip of pure
// punctuation is near-silent audio that still costs a synthesis and an inter-clip pause.
func sanitizeSpeechText(text string) string {
	runes := []rune(text)
	var b strings.Builder
	stripped := false

	for i := 0; i < len(runes); i++ {
		if !annotationOpeners[runes[i]] {
			b.WriteRune(runes[i])
			continue
		}
		end := matchingCloser(runes, i)
		if end < 0 {
			b.WriteRune(runes[i])
			continue
		}
		i = end
		stripped = true
	}

	if !stripped {
		return text
	}
	out := strings.TrimSpace(collapseSpaces(b.String()))
	if !hasPronounceable(out) {
		return text
	}
	return out
}

// matchingCloser returns the index of the closer that matches the bracket opened at start,
// honoring nested round brackets, or -1 when it is never closed.
func matchingCloser(runes []rune, start int) int {
	depth := 0
	for i := start; i < len(runes); i++ {
		switch {
		case annotationOpeners[runes[i]]:
			depth++
		case annotationClosers[runes[i]]:
			depth--
			if depth == 0 {
				return i
			}
		}
	}
	return -1
}

// hasPronounceable reports whether s holds anything that becomes speech. Punctuation and
// spaces alone do not.
func hasPronounceable(s string) bool {
	for _, r := range s {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			return true
		}
	}
	return false
}

// collapseSpaces squeezes runs of spaces left behind by a removal into a single space.
func collapseSpaces(s string) string {
	var b strings.Builder
	prevSpace := false
	for _, r := range s {
		isSpace := r == ' ' || r == '　'
		if isSpace && prevSpace {
			continue
		}
		prevSpace = isSpace
		b.WriteRune(r)
	}
	return b.String()
}
