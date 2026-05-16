package sheet

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Under `go test` stdout is not a TTY, so term.GetSize returns an error and
// terminalHeight must fall back to its hard-coded default.
func TestTerminalHeightFallback(t *testing.T) {
	const fallback = 24
	if got := terminalHeight(); got != fallback {
		t.Errorf("terminalHeight() = %d, want %d (fallback)", got, fallback)
	}
}

// When stdout is not a terminal (e.g. piped to another command), View must
// dump every line at once instead of paginating — otherwise `she foo | grep`
// silently truncates after one page worth of output.
func TestViewDumpsAllLinesWhenStdoutIsNotTerminal(t *testing.T) {
	withHome(t)
	if err := EnsureDir(); err != nil {
		t.Fatalf("EnsureDir() error = %v", err)
	}
	dir, _ := Dir()
	path := filepath.Join(dir, "many")

	const n = 200
	var b strings.Builder
	for i := 0; i < n; i++ {
		fmt.Fprintf(&b, "cmd%03d # desc%03d\n", i, i)
	}
	if err := os.WriteFile(path, []byte(b.String()), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	var stdout, stderr bytes.Buffer
	if err := View(&stdout, &stderr, path); err != nil {
		t.Fatalf("View() error = %v", err)
	}

	for _, marker := range []string{"cmd000", "cmd099", "cmd199"} {
		if !strings.Contains(stdout.String(), marker) {
			t.Errorf("stdout missing %q — output truncated by pager", marker)
		}
	}
	if strings.Contains(stdout.String(), "Press any key") {
		t.Errorf("stdout contains pager prompt; non-TTY should not paginate")
	}
}

// A *bytes.Buffer cannot be a terminal — guards against a regression where
// isTerminal accidentally treats non-files as TTYs.
func TestIsTerminalRejectsNonFiles(t *testing.T) {
	if isTerminal(&bytes.Buffer{}) {
		t.Errorf("isTerminal(*bytes.Buffer) = true, want false")
	}
}

// A typed-nil *os.File satisfies io.Writer with a non-nil interface header
// but a nil concrete value. Without the explicit nil check, isTerminal would
// dereference it via Fd() and panic.
func TestIsTerminalRejectsTypedNilFile(t *testing.T) {
	var f *os.File
	if isTerminal(f) {
		t.Errorf("isTerminal(typed-nil *os.File) = true, want false")
	}
}
