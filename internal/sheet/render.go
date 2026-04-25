package sheet

import (
	"fmt"
	"strings"
)

const (
	ansiReset  = "\033[0m"
	ansiDim    = "\033[90m"
	ansiGreen  = "\033[32;1m"
	ansiItalic = "\033[3m"
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
	var errors []string
	realLineNum := 0
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "//") {
			continue
		}
		realLineNum++
		if !strings.Contains(line, " > ") {
			errors = append(errors, fmt.Sprintf("line %d: missing ' > ' separator", realLineNum))
		}
	}
	return errors
}