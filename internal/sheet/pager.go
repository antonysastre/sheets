package sheet

import (
	"fmt"
	"os"
	"strings"

	"golang.org/x/sys/unix"
)

func terminalHeight() int {
	const fallback = 24
	ws, err := unix.IoctlGetWinsize(int(os.Stdout.Fd()), unix.TIOCGWINSZ)
	if err != nil || ws.Row == 0 {
		return fallback
	}
	return int(ws.Row)
}

func readKey() (byte, error) {
	fd := int(os.Stdin.Fd())

	oldTermios, err := unix.IoctlGetTermios(fd, unix.TCGETS)
	if err != nil {
		return 0, err
	}

	newTermios := *oldTermios
	newTermios.Lflag &^= unix.ICANON | unix.ECHO
	if err := unix.IoctlSetTermios(fd, unix.TCSETS, &newTermios); err != nil {
		return 0, err
	}
	defer func() {
		_ = unix.IoctlSetTermios(fd, unix.TCSETS, oldTermios)
	}()

	buf := make([]byte, 1)
	if _, err := os.Stdin.Read(buf); err != nil {
		return 0, err
	}
	return buf[0], nil
}

func shouldContinue() bool {
	fmt.Print("\x1b[90m[Press any key... (q to quit)]\x1b[0m")

	key, err := readKey()
	if err != nil {
		fmt.Println()
		return false
	}

	fmt.Println("\r" + strings.Repeat(" ", 40) + "\r")

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
		fmt.Println("\x1b[33mFormat warnings:\x1b[0m")
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
