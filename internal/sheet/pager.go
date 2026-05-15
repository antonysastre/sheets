package sheet

import (
	"fmt"
	"io"
	"os"
	"os/signal"
	"strings"
	"syscall"

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

// isTerminal reports whether w refers to an interactive terminal. A non-file
// writer (e.g. bytes.Buffer in tests) or a pipe both yield false.
func isTerminal(w io.Writer) bool {
	f, ok := w.(*os.File)
	if !ok {
		return false
	}
	return term.IsTerminal(int(f.Fd()))
}

// readKey reads one byte from stdin in raw mode. A SIGINT/SIGTERM/SIGHUP
// during the read restores the terminal before exiting — the deferred Restore
// does not run when a signal kills the process, so we exit explicitly with
// the conventional 128+signal code.
func readKey() (byte, error) {
	fd := int(os.Stdin.Fd())
	oldState, err := term.MakeRaw(fd)
	if err != nil {
		return 0, err
	}
	defer func() { _ = term.Restore(fd, oldState) }()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM, syscall.SIGHUP)
	done := make(chan struct{})
	defer func() {
		signal.Stop(sigCh)
		close(done)
	}()
	go func() {
		select {
		case sig := <-sigCh:
			_ = term.Restore(fd, oldState)
			code := 128
			if s, ok := sig.(syscall.Signal); ok {
				code += int(s)
			}
			os.Exit(code)
		case <-done:
		}
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
// format-validation warnings are written to stderr. Output is paginated only
// when stdout is an interactive terminal — piped or redirected output dumps
// everything without prompting.
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

	if !isTerminal(stdout) {
		for _, line := range lines {
			if line != "" {
				fmt.Fprintln(stdout, RenderLine(line))
			}
		}
		return nil
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
