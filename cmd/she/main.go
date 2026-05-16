// Command she is a small cheat-sheet manager. It stores plain-text sheets
// under ~/.sheets and prints them with simple formatting.
package main

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/antonysastre/sheets/internal/sheet"
)

// version is overridden at build time via -ldflags "-X main.version=...".
var version = "dev"

func main() {
	os.Exit(run(os.Args, os.Stdout, os.Stderr))
}

// run dispatches a single she invocation and returns the process exit code:
// 0 on success, 1 for a runtime failure, 2 for a command-line usage error.
// Arguments and output streams are passed in explicitly so run can be
// exercised by tests without spawning a subprocess.
func run(args []string, stdout, stderr io.Writer) int {
	if len(args) < 2 {
		printUsage(stderr)
		return 2
	}

	if err := sheet.EnsureDir(); err != nil {
		return runtimeError(stderr, "failed to create sheets directory: %v", err)
	}

	switch args[1] {
	case "--edit", "-e":
		name, code := requireToolOperand(args, stderr, "she --edit, -e <tool>")
		if code != 0 {
			return code
		}
		if err := sheet.Edit(stderr, name); err != nil {
			return runtimeError(stderr, "failed to edit sheet: %v", err)
		}

	case "--new", "-n":
		name, code := requireToolOperand(args, stderr, "she --new, -n <tool>")
		if code != 0 {
			return code
		}
		if err := sheet.Initialize(stderr, name); err != nil {
			return runtimeError(stderr, "failed to create sheet: %v", err)
		}

	case "--list", "-l":
		if err := sheet.List(stdout); err != nil {
			return runtimeError(stderr, "failed to list sheets: %v", err)
		}

	case "--sync", "-s":
		repo, code := optionalOperand(args, stderr)
		if code != 0 {
			return code
		}
		if err := sheet.Sync(stderr, repo); err != nil {
			return runtimeError(stderr, "failed to sync sheets: %v", err)
		}

	case "--help", "-h":
		printUsage(stdout)

	case "--version", "-V":
		_, _ = fmt.Fprintf(stdout, "she %s\n", version)

	// "--" ends option parsing, so the next argument is taken literally —
	// the only way to view a sheet whose name begins with a dash.
	case "--":
		name, code := requireOperand(args, stderr, "she -- <tool>")
		if code != 0 {
			return code
		}
		return viewSheet(name, stdout, stderr)

	default:
		name := args[1]
		if strings.HasPrefix(name, "-") {
			return usageError(stderr, "unknown flag: %s", name)
		}
		return viewSheet(name, stdout, stderr)
	}

	return 0
}

// viewSheet renders the named sheet and returns the process exit code: 0 if
// the sheet was found and rendered, 1 if the sheet does not exist, 2 for a
// usage error. Rendered content goes to stdout; diagnostics go to stderr.
func viewSheet(name string, stdout, stderr io.Writer) int {
	if name == "" {
		return usageError(stderr, "usage: she <tool>")
	}

	if !sheet.Exists(name) {
		_, _ = fmt.Fprintf(stderr, "No cheat sheet found for %q.\n", name)
		_, _ = fmt.Fprintf(stderr, "Run 'she --edit %s' to create one.\n", name)
		return 1
	}

	path, err := sheet.Path(name)
	if err != nil {
		return runtimeError(stderr, "failed to resolve sheet path: %v", err)
	}
	if err := sheet.View(stdout, stderr, path); err != nil {
		return runtimeError(stderr, "failed to view sheet: %v", err)
	}
	return 0
}

// requireOperand returns args[2], the operand for a command that requires one.
// A missing operand is reported as a usage error and the returned exit code is
// non-zero for the caller to propagate. Used by "--" for raw operands, which
// may legitimately start with a dash.
func requireOperand(args []string, stderr io.Writer, usage string) (string, int) {
	if len(args) < 3 {
		return "", usageError(stderr, "usage: %s", usage)
	}
	return args[2], 0
}

// requireToolOperand returns args[2] as a non-empty tool name, rejecting an
// absent, empty, or flag-shaped operand as a usage error. Used by --edit and
// --new, whose operand must be a real sheet name (never a flag).
func requireToolOperand(args []string, stderr io.Writer, usage string) (string, int) {
	name, code := requireOperand(args, stderr, usage)
	if code != 0 {
		return "", code
	}
	if name == "" {
		return "", usageError(stderr, "usage: %s", usage)
	}
	if strings.HasPrefix(name, "-") {
		return "", usageError(stderr, "unknown flag: %s", name)
	}
	return name, 0
}

// optionalOperand returns args[2] for a command whose operand is optional,
// such as --sync. An absent operand yields "" with a zero exit code. An
// operand that looks like a flag is rejected as a usage error — it almost
// certainly means the command was fat-fingered, e.g. `she --sync --help`.
func optionalOperand(args []string, stderr io.Writer) (string, int) {
	if len(args) < 3 {
		return "", 0
	}
	operand := args[2]
	if strings.HasPrefix(operand, "-") {
		return "", usageError(stderr, "unknown flag: %s", operand)
	}
	return operand, 0
}

func printUsage(w io.Writer) {
	_, _ = fmt.Fprintln(w, `she - cheat sheet manager

Usage:
  she <tool>		View cheat sheet
  she --list, -l	List all cheat sheets
  she --edit, -e <tool>	Edit cheat sheet (creates if missing)
  she --new, -n <tool>	Create new cheat sheet
  she --sync, -s [repo]	Sync sheets to a git repo (pass repo to set up)
  she --help, -h	Show this help
  she --version, -V	Print version and exit

Examples:
  she docker		View docker sheet
  she --edit docker	Edit docker sheet
  she --list		Show all sheets
  she --sync git@github.com:me/sheets.git	Set up syncing
  she --sync		Sync sheets`)
}

// runtimeError reports a runtime failure on stderr and returns exit code 1.
func runtimeError(stderr io.Writer, format string, args ...any) int {
	_, _ = fmt.Fprintf(stderr, format+"\n", args...)
	return 1
}

// usageError reports a command-line usage error on stderr and returns exit
// code 2, the conventional code for incorrect invocation.
func usageError(stderr io.Writer, format string, args ...any) int {
	_, _ = fmt.Fprintf(stderr, format+"\n", args...)
	_, _ = fmt.Fprintln(stderr, "Run 'she --help' for more information.")
	return 2
}
