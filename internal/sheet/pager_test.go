package sheet

import (
	"bytes"
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

// Format warnings are diagnostic output and must go to stderr so they do not
// corrupt the data stream when stdout is piped.
func TestViewWarningsGoToStderr(t *testing.T) {
	withHome(t)
	if err := EnsureDir(); err != nil {
		t.Fatalf("EnsureDir() error = %v", err)
	}
	dir, _ := Dir()
	path := filepath.Join(dir, "bad")
	if err := os.WriteFile(path, []byte("missing separator\n"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	var stdout, stderr bytes.Buffer
	if err := View(&stdout, &stderr, path); err != nil {
		t.Fatalf("View() error = %v", err)
	}

	if !strings.Contains(stderr.String(), "Format warnings") {
		t.Errorf("stderr = %q, want it to contain 'Format warnings'", stderr.String())
	}
	if strings.Contains(stdout.String(), "Format warnings") {
		t.Errorf("stdout = %q, want it to NOT contain 'Format warnings'", stdout.String())
	}
}
