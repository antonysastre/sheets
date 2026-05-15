package sheet

import "testing"

// Under `go test` stdout is not a TTY, so term.GetSize returns an error and
// terminalHeight must fall back to its hard-coded default.
func TestTerminalHeightFallback(t *testing.T) {
	const fallback = 24
	if got := terminalHeight(); got != fallback {
		t.Errorf("terminalHeight() = %d, want %d (fallback)", got, fallback)
	}
}
