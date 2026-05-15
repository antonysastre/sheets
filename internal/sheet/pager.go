package sheet

import (
	"fmt"
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

func shouldContinue() bool {
	fmt.Print(ansiDim + "[Press any key... (q to quit)]" + ansiReset)

	key, err := readKey()
	if err != nil {
		fmt.Println()
		return false
	}

	fmt.Print(eraseLine)

	return key != 'q' && key != 'Q'
}

// View reads the sheet file at path, validates its format, and prints the
// rendered content to standard output, paginating to terminal height.
func View(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read sheet: %w", err)
	}

	lines := strings.Split(string(data), "\n")

	if issues := ValidateFormat(lines); len(issues) > 0 {
		fmt.Println(ansiYellow + "Format warnings:" + ansiReset)
		for _, msg := range issues {
			fmt.Println("  ", msg)
		}
		fmt.Println()
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
				fmt.Println(RenderLine(line))
			}
		}

		if end >= len(lines) {
			break
		}

		if !shouldContinue() {
			break
		}
		start = end
	}

	return nil
}
