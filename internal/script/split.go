package script

import (
	"strings"

	"github.com/canpok1/vox-radio/internal/model"
)

// terminalPunctuation are the sentence-ending characters that trigger a split.
// 、（読点）・…・― は対象外（ADR-0104）。
var terminalPunctuation = map[rune]bool{
	'。': true, '！': true, '？': true, '!': true, '?': true,
}

// closingFollowers are closing brackets/quotes absorbed into the preceding
// sentence when they directly follow a terminal punctuation mark.
var closingFollowers = map[rune]bool{
	'」': true, '』': true, '）': true, ')': true, '"': true, '\'': true,
}

// bracketPairs maps an opening bracket to its closing counterpart. Terminal
// punctuation inside an open bracket (nest depth > 0) does not trigger a split.
var bracketPairs = map[rune]rune{
	'「': '」',
	'『': '』',
	'（': '）',
	'(': ')',
}

// SplitLinesIntoSentences splits each Line's text into one Line per sentence.
// write.md previously asked the LLM to split lines at emotion boundaries, but
// the LLM did not reliably follow it, so splitting is done mechanically here
// instead (ADR-0104). Each resulting Line inherits SpeakerRole and Style from
// its parent; a Line without any sentence terminator is returned unsplit.
func SplitLinesIntoSentences(lines []model.Line) []model.Line {
	result := make([]model.Line, 0, len(lines))
	for _, line := range lines {
		for _, text := range splitSentences(line.Text) {
			result = append(result, model.Line{
				SpeakerRole: line.SpeakerRole,
				Style:       line.Style,
				Text:        text,
			})
		}
	}
	return result
}

// splitSentences splits text at terminal punctuation, honoring bracket/quote
// nesting. It returns []string{text} unchanged if text has no terminator.
func splitSentences(text string) []string {
	var sentences []string
	var stack []rune
	quoteOpen := map[rune]bool{'"': false, '\'': false}
	var current strings.Builder
	split := false

	runes := []rune(text)
	for i := 0; i < len(runes); i++ {
		r := runes[i]
		current.WriteRune(r)

		if closer, isOpen := bracketPairs[r]; isOpen {
			stack = append(stack, closer)
			continue
		}
		if len(stack) > 0 && r == stack[len(stack)-1] {
			stack = stack[:len(stack)-1]
			continue
		}
		if r == '"' || r == '\'' {
			quoteOpen[r] = !quoteOpen[r]
			continue
		}
		if len(stack) > 0 || quoteOpen['"'] || quoteOpen['\''] {
			continue
		}
		if !terminalPunctuation[r] {
			continue
		}

		for i+1 < len(runes) && terminalPunctuation[runes[i+1]] {
			i++
			current.WriteRune(runes[i])
		}
		for i+1 < len(runes) && closingFollowers[runes[i+1]] {
			i++
			current.WriteRune(runes[i])
		}
		split = true
		sentences = append(sentences, current.String())
		current.Reset()
	}

	if !split {
		return []string{text}
	}
	if rest := current.String(); strings.TrimSpace(rest) != "" {
		sentences = append(sentences, rest)
	}
	return sentences
}
