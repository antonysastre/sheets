package sheet

import (
	"fmt"
	"io"
	"os"
	"strings"

	"golang.org/x/term"
)

// eraseLine returns the cursor to column 0 and erases the current line.
const eraseLine = "\r\x1b[2K"

func terminalHeight() int {
	const fallback = 24
	_, h, err := term.GetSize(int(os.Stdout.Fd()))
	if err != nil || h == 0 {
		return fallback
	}
	return h
}

func readKey() (byte, error) {
	fd := int(os.Stdin.Fd())
	oldState, err := term.MakeRaw(fd)
	if err != nil {
		return 0, err
	}
	defer func() {
		// Best-effort: there is nothing useful to do if Restore fails.
		_ = term.Restore(fd, oldState)
	}()

	buf := make([]byte, 1)
	if _, err := os.Stdin.Read(buf); err != nil {
		return 0, err
	}
	return buf[0], nil
}

func shouldContinue(w io.Writer) bool {
	fmt.Fprint(w, ansiDim+"[Press any key... (q to quit)]"+ansiReset)

	key, err := readKey()
	if err != nil {
		fmt.Fprintln(w)
		return false
	}

	fmt.Fprint(w, eraseLine)

	return key != 'q' && key != 'Q'
}

// View renders the sheet at path. Rendered content is written to stdout;
// format-validation warnings are written to stderr.
func View(stdout, stderr io.Writer, path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read sheet: %w", err)
	}

	lines := strings.Split(string(data), "\n")

	if issues := ValidateFormat(lines); len(issues) > 0 {
		fmt.Fprintln(stderr, ansiYellow+"Format warnings:"+ansiReset)
		for _, msg := range issues {
			fmt.Fprintln(stderr, "  ", msg)
		}
		fmt.Fprintln(stderr)
	}

	pageSize := terminalHeight() - 2
	if pageSize < 5 {
		pageSize = 10
	}

	start := 0
	for {
		end := start + pageSize
		if end > len(lines) {
			end = len(lines)
		}

		for _, line := range lines[start:end] {
			if line != "" {
				fmt.Fprintln(stdout, RenderLine(line))
			}
		}

		if end >= len(lines) {
			break
		}

		if !shouldContinue(stdout) {
			break
		}
		start = end
	}

	return nil
}
