package asc

import "strings"

// SanitizeTerminalText removes characters that a terminal, pager, or CI log
// viewer interprets instead of displaying. Structured output (JSON) keeps the
// original values; only human-facing table, Markdown, stdout, and stderr text
// is sanitized.
//
// Removed characters:
//   - C0 controls (U+0000-U+001F) and DEL (U+007F), which include the ESC
//     introducer used by CSI/OSC sequences as well as BEL, CR, LF, and TAB
//   - C1 controls (U+0080-U+009F), which include the single-byte CSI (U+009B)
//     and OSC (U+009D) forms
//   - Bidirectional marks, embeddings, overrides, and isolates (U+200E, U+200F,
//     U+202A-U+202E, U+2066-U+2069), which can visually reorder a rendered cell
//
// Line breaks and tabs are removed rather than preserved so a single value
// cannot forge extra rows or columns in table, Markdown, or log output.
func SanitizeTerminalText(input string) string {
	if input == "" || !HasInterpretedTerminalSequence(input) {
		return input
	}

	var b strings.Builder
	b.Grow(len(input))
	for _, r := range input {
		if isInterpretedTerminalRune(r) {
			continue
		}
		b.WriteRune(r)
	}
	return b.String()
}

// HasInterpretedTerminalSequence reports whether input contains any character
// that SanitizeTerminalText removes.
func HasInterpretedTerminalSequence(input string) bool {
	for _, r := range input {
		if isInterpretedTerminalRune(r) {
			return true
		}
	}
	return false
}

func isInterpretedTerminalRune(r rune) bool {
	switch {
	case r < 0x20, r == 0x7f:
		return true
	case r >= 0x80 && r <= 0x9f:
		return true
	case r == 0x200e, r == 0x200f:
		return true
	case r >= 0x202a && r <= 0x202e:
		return true
	case r >= 0x2066 && r <= 0x2069:
		return true
	default:
		return false
	}
}
