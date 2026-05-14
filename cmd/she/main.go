// Command she is a small cheat-sheet manager. It stores plain-text sheets
// under ~/.sheets and prints them with simple formatting.
package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/antonysastre/sheets/internal/sheet"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	if err := sheet.EnsureDir(); err != nil {
		die("failed to create sheets directory: %v", err)
	}

	switch os.Args[1] {
	case "--edit", "-e":
		if err := sheet.Edit(requireOperand("she --edit, -e <tool>")); err != nil {
			die("edit failed: %v", err)
		}

	case "--new", "-n":
		if err := sheet.New(requireOperand("she --new, -n <tool>")); err != nil {
			die("new failed: %v", err)
		}

	case "--list", "-l":
		if err := sheet.List(); err != nil {
			die("list failed: %v", err)
		}

	case "--sync", "-s":
		if err := sheet.Sync(optionalOperand()); err != nil {
			die("sync failed: %v", err)
		}

	case "--help", "-h":
		printUsage()

	// "--" ends option parsing, so the next argument is taken literally —
	// the only way to view a sheet whose name begins with a dash.
	case "--":
		viewSheet(requireOperand("she -- <tool>"))

	default:
		name := os.Args[1]
		if strings.HasPrefix(name, "-") {
			dieUsage("unknown flag: %s", name)
		}
		viewSheet(name)
	}
}

// viewSheet renders the named sheet to standard output, or reports that no
// such sheet exists. An empty name is treated as a usage error.
func viewSheet(name string) {
	if name == "" {
		dieUsage("usage: she <tool>")
	}

	if !sheet.Exists(name) {
		fmt.Printf("No cheat sheet found for '%s'.\n", name)
		fmt.Println("Run 'she --edit " + name + "' to create one.")
		return
	}

	path, err := sheet.Path(name)
	if err != nil {
		die("failed to resolve sheet path: %v", err)
	}
	if err := sheet.View(path); err != nil {
		die("failed to view sheet: %v", err)
	}
}

// requireOperand returns the command operand (os.Args[2]), exiting with a
// usage error if it is missing. usage is the one-line invocation summary
// shown to the user.
func requireOperand(usage string) string {
	if len(os.Args) < 3 {
		dieUsage("usage: %s", usage)
	}
	return os.Args[2]
}

// optionalOperand returns the command operand (os.Args[2]) if present, or
// the empty string otherwise. It is for commands whose operand is optional,
// such as --sync.
func optionalOperand() string {
	if len(os.Args) < 3 {
		return ""
	}
	return os.Args[2]
}

func printUsage() {
	fmt.Println(`she - cheat sheet manager

Usage:
  she <tool>		View cheat sheet
  she --list, -l	List all cheat sheets
  she --edit, -e <tool>	Edit cheat sheet (creates if missing)
  she --new, -n <tool>	Create new cheat sheet
  she --sync, -s [repo]	Sync sheets to a git repo (pass repo to set up)
  she --help, -h	Show this help

Examples:
  she docker		View docker sheet
  she --edit docker	Edit docker sheet
  she --list		Show all sheets
  she --sync git@github.com:me/sheets.git	Set up syncing
  she --sync		Sync sheets`)
}

// die reports a runtime failure on standard error and exits with status 1.
func die(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}

// dieUsage reports a command-line usage error on standard error and exits
// with status 2, the conventional exit code for incorrect invocation.
func dieUsage(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	fmt.Fprintln(os.Stderr, "Run 'she --help' for more information.")
	os.Exit(2)
}
