package direct

import "strings"

// annotationBrackets maps an opening round bracket to its closing counterpart.
// Quotation brackets (「」『』) are deliberately absent: they carry article titles and
// quoted speech that are meant to be read aloud.
var annotationBrackets = map[rune]rune{
	'（': '）',
	'(': ')',
}

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
// line. A line that would become empty is returned unchanged, since an empty clip has no
// text to synthesize.
func sanitizeSpeechText(text string) string {
	runes := []rune(text)
	var b strings.Builder
	stripped := false

	for i := 0; i < len(runes); i++ {
		closer, isOpen := annotationBrackets[runes[i]]
		if !isOpen {
			b.WriteRune(runes[i])
			continue
		}
		end := matchingBracket(runes, i, runes[i], closer)
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
	if out == "" {
		return text
	}
	return out
}

// matchingBracket returns the index of the closer that matches the bracket opened at start,
// honoring nesting of that same bracket kind, or -1 when it is never closed.
func matchingBracket(runes []rune, start int, opener, closer rune) int {
	depth := 0
	for i := start; i < len(runes); i++ {
		switch runes[i] {
		case opener:
			depth++
		case closer:
			depth--
			if depth == 0 {
				return i
			}
		}
	}
	return -1
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
