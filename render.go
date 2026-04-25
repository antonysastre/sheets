package main

import (
	"fmt"
	"strings"
)

const (
	ansiReset  = "\033[0m"
	ansiDim    = "\033[90m"
	ansiGreen  = "\033[32;1m"
	ansiCyan   = "\033[36;1m"
	ansiItalic = "\033[3m"
)

func renderLine(line string) string {
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

func validateFormat(lines []string) []string {
	var errors []string
	realLineNum := 0
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		realLineNum++
		if !strings.Contains(line, " > ") {
			errors = append(errors, fmt.Sprintf("line %d: missing ' > ' separator", realLineNum))
		}
	}
	return errors
}
