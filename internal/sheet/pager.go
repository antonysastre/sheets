package sheet

import (
	"fmt"
	"os"
	"strings"
	"unsafe"

	"golang.org/x/sys/unix"
)

type winsize struct {
	row    uint16
	col    uint16
	xpixel uint16
	ypixel uint16
}

func terminalHeight() int {
	var ws winsize
	ret, _, _ := unix.Syscall(unix.SYS_IOCTL, uintptr(os.Stdout.Fd()), uintptr(unix.TIOCGWINSZ), uintptr(unsafe.Pointer(&ws)))
	if ret != 0 {
		return 24
	}
	if ws.row == 0 {
		return 24
	}
	return int(ws.row)
}

func readKey() (byte, error) {
	fd := int(os.Stdin.Fd())

	var oldTermios unix.Termios
	if err := unix.IoctlSetTermios(fd, unix.TCGETS, &oldTermios); err != nil {
		return 0, err
	}

	newTermios := oldTermios
	newTermios.Lflag &^= unix.ICANON | unix.ECHO
	if err := unix.IoctlSetTermios(fd, unix.TCSETS, &newTermios); err != nil {
		return 0, err
	}

	defer func() {
		_ = unix.IoctlSetTermios(fd, unix.TCSETS, &oldTermios)
	}()

	charBuf := make([]byte, 1)
	if _, err := os.Stdin.Read(charBuf); err != nil {
		return 0, err
	}

	return charBuf[0], nil
}

func shouldContinue() bool {
	fmt.Print("\033[90m[Press any key... (q to quit)]\033[0m")

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

	validationErrors := ValidateFormat(lines)
	if len(validationErrors) > 0 {
		fmt.Println("\033[33mFormat warnings:\033[0m")
		for _, e := range validationErrors {
			fmt.Println("  ", e)
		}
		fmt.Println()
	}

	termHeight := terminalHeight()
	pageSize := termHeight - 2
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