package main

import (
	"bytes"
	"strings"
	"testing"
)

// TestRun exercises run's argument dispatch: the exit code each invocation
// yields and which stream it writes to. It covers the paths that do not
// delegate into the sheet package's rendering — that dispatch logic is where
// the bugs live.
func TestRun(t *testing.T) {
	tests := []struct {
		name       string
		args       []string
		wantCode   int
		wantStdout string // substring; "" means stdout must be empty
		wantStderr string // substring; "" means stderr must be empty
	}{
		{"no args", []string{"she"}, 2, "", "Usage:"},
		{"help to stdout", []string{"she", "--help"}, 0, "Usage:", ""},
		{"help short flag", []string{"she", "-h"}, 0, "Usage:", ""},
		{"unknown flag", []string{"she", "--bogus"}, 2, "", "unknown flag: --bogus"},
		{"edit without operand", []string{"she", "--edit"}, 2, "", "usage: she --edit"},
		{"edit with flag operand", []string{"she", "--edit", "--help"}, 2, "", "unknown flag: --help"},
		{"edit with empty operand", []string{"she", "--edit", ""}, 2, "", "usage: she --edit"},
		{"new without operand", []string{"she", "--new"}, 2, "", "usage: she --new"},
		{"new with flag operand", []string{"she", "--new", "-x"}, 2, "", "unknown flag: -x"},
		{"new with empty operand", []string{"she", "--new", ""}, 2, "", "usage: she --new"},
		{"sync with flag operand", []string{"she", "--sync", "--help"}, 2, "", "unknown flag: --help"},
		{"sync with dash operand", []string{"she", "--sync", "-x"}, 2, "", "unknown flag: -x"},
		{"empty tool name", []string{"she", ""}, 2, "", "usage: she <tool>"},
		{"sheet not found", []string{"she", "nonexistent"}, 1, "", "No cheat sheet found"},
		{"list with no sheets", []string{"she", "--list"}, 0, "No sheets found", ""},
		{"list short flag", []string{"she", "-l"}, 0, "No sheets found", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Sandbox the sheets directory so run's EnsureDir call cannot
			// touch the real ~/.sheets.
			t.Setenv("HOME", t.TempDir())

			var stdout, stderr bytes.Buffer
			code := run(tt.args, &stdout, &stderr)

			if code != tt.wantCode {
				t.Errorf("run(%v) = %d, want %d", tt.args[1:], code, tt.wantCode)
			}
			checkStream(t, "stdout", stdout.String(), tt.wantStdout)
			checkStream(t, "stderr", stderr.String(), tt.wantStderr)
		})
	}
}

// checkStream asserts that got contains want, or — when want is empty — that
// got is empty too, so each outcome is verified to use exactly one stream.
func checkStream(t *testing.T, name, got, want string) {
	t.Helper()
	switch {
	case want == "" && got != "":
		t.Errorf("%s = %q, want it to be empty", name, got)
	case want != "" && !strings.Contains(got, want):
		t.Errorf("%s = %q, want it to contain %q", name, got, want)
	}
}
