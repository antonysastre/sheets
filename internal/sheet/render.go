package sheet

import (
	"fmt"
	"strings"
)

const (
	ansiReset  = "\x1b[0m"
	ansiDim    = "\x1b[90m"
	ansiGreen  = "\x1b[32;1m"
	ansiItalic = "\x1b[3m"
)

// RenderLine formats a single sheet line for terminal display. Lines starting
// with "//" are treated as comments and rendered as empty strings; lines
// containing the " > " separator are split into a colored command/description
// pair; all other lines are returned unchanged (modulo trailing newlines).
func RenderLine(line string) string {
	line = strings.TrimRight(line, "\r\n")
	if strings.HasPrefix(line, "//") {
		return ""
	}

	if idx := strings.Index(line, " > "); idx != -1 {
		cmd := line[:idx]
		desc := line[idx+3:]
		indent := "  "
		sep := ansiDim + " · " + ansiReset
		return indent + ansiGreen + cmd + ansiReset + sep + ansiItalic + desc + ansiReset
	}

	return line
}

// ValidateFormat scans lines and returns human-readable messages for any
// non-empty, non-comment line that does not contain the " > " separator.
// The returned slice is empty when the input is well-formed.
func ValidateFormat(lines []string) []string {
	var issues []string
	n := 0
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "//") {
			continue
		}
		n++
		if !strings.Contains(line, " > ") {
			issues = append(issues, fmt.Sprintf("line %d: missing ' > ' separator", n))
		}
	}
	return issues
}
