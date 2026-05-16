package sheet

import (
	"strings"
	"unicode/utf8"
)

const (
	ansiReset    = "\x1b[0m"
	ansiDim      = "\x1b[90m"
	ansiGreen    = "\x1b[32;1m"
	ansiItalic   = "\x1b[3m"
	ansiCyanBold = "\x1b[36;1m"
)

// RenderLine formats a single sheet line for terminal display, following the
// markdown-flavored format: blank lines stay blank; lines beginning with '#'
// are free-standing comments; lines beginning with an uppercase ASCII letter
// are section headers; everything else is a command line, optionally followed
// by a '#'-introduced description.
//
// When width > 0, command-with-description lines are right-padded so that
// all descriptions align in one vertical column — typically the caller
// passes the result of MaxCommandWidth on the whole sheet.
func RenderLine(line string, width int) string {
	line = strings.TrimRight(line, "\r\n")
	trimmed := strings.TrimLeft(line, " \t")

	if trimmed == "" {
		return ""
	}

	first := trimmed[0]
	switch {
	case first == '#':
		return ansiDim + ansiItalic + trimmed + ansiReset
	case first >= 'A' && first <= 'Z':
		return ansiCyanBold + trimmed + ansiReset
	}

	cmd, desc, hasDesc := strings.Cut(trimmed, "#")
	cmd = strings.TrimRight(cmd, " \t")
	if !hasDesc {
		return ansiGreen + cmd + ansiReset
	}
	desc = strings.TrimLeft(desc, " \t")

	pad := ""
	if w := utf8.RuneCountInString(cmd); width > w {
		pad = strings.Repeat(" ", width-w)
	}
	return ansiGreen + cmd + ansiReset + pad + ansiDim + " · " + ansiReset + ansiItalic + desc + ansiReset
}

// MaxCommandWidth returns the rune length of the longest command on any
// line that has both a command and a description. Lines without a
// description, blank lines, headers, and free-standing comments are
// ignored — they don't participate in column alignment.
//
// Width is measured in runes, not terminal cells, so wide characters
// (CJK, emoji) and tabs in commands will not align perfectly. Sheets
// are ASCII in practice, so this is rarely visible.
func MaxCommandWidth(lines []string) int {
	n := 0
	for _, line := range lines {
		trimmed := strings.TrimLeft(strings.TrimRight(line, "\r\n"), " \t")
		if trimmed == "" {
			continue
		}
		first := trimmed[0]
		if first == '#' || (first >= 'A' && first <= 'Z') {
			continue
		}
		cmd, _, hasDesc := strings.Cut(trimmed, "#")
		if !hasDesc {
			continue
		}
		cmd = strings.TrimRight(cmd, " \t")
		if w := utf8.RuneCountInString(cmd); w > n {
			n = w
		}
	}
	return n
}
