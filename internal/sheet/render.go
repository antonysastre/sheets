package sheet

import "strings"

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
func RenderLine(line string) string {
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
		return "  " + ansiGreen + cmd + ansiReset
	}
	desc = strings.TrimLeft(desc, " \t")
	return "  " + ansiGreen + cmd + ansiReset + ansiDim + " · " + ansiReset + ansiItalic + desc + ansiReset
}
