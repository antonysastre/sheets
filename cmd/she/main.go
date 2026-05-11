// Command she is a small cheat-sheet manager. It stores plain-text sheets
// under ~/.sheets and prints them with simple formatting.
package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/antonysastre/sheets/internal/sheet"
)

var version = "dev"

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	if err := sheet.EnsureDir(); err != nil {
		die("failed to create sheets directory: %v", err)
	}

	arg := os.Args[1]
	flag, inlineVal := splitFlag(arg)

	switch flag {
	case "--edit", "-e":
		if inlineVal != "" && len(os.Args) > 2 {
			dieUsage("-e, --edit <tool>")
		}
		name := requireOperand(inlineVal, "-e, --edit <tool>")
		if err := sheet.Edit(name); err != nil {
			die("edit failed: %v", err)
		}

	case "--new", "-n":
		if inlineVal != "" && len(os.Args) > 2 {
			dieUsage("-n, --new <tool>")
		}
		name := requireOperand(inlineVal, "-n, --new <tool>")
		if err := sheet.New(name); err != nil {
			die("new failed: %v", err)
		}

	case "--list", "-l":
		if inlineVal != "" {
			dieUsage("-l, --list")
		}
		if err := sheet.List(); err != nil {
			die("list failed: %v", err)
		}

	case "--help", "-h":
		if inlineVal != "" {
			dieUsage("-h, --help")
		}
		printUsage()

	case "--version", "-V":
		if inlineVal != "" {
			dieUsage("-V, --version")
		}
		fmt.Printf("she %s\n", version)

	case "--sync", "-s":
		if inlineVal != "" {
			if len(os.Args) > 2 {
				dieUsage("-s, --sync [url]")
			}
			if err := sheet.SyncInit(inlineVal); err != nil {
				die("sync init failed: %v", err)
			}
		} else if len(os.Args) == 3 {
			if err := sheet.SyncInit(os.Args[2]); err != nil {
				die("sync init failed: %v", err)
			}
		} else if len(os.Args) > 3 {
			dieUsage("-s, --sync [url]")
		}
		if err := sheet.Sync(); err != nil {
			die("sync failed: %v", err)
		}

	case "--":
		if len(os.Args) < 3 {
			dieUsage("-- <tool>")
		}
		viewSheet(os.Args[2])

	default:
		if len(flag) > 0 && flag[0] == '-' {
			fmt.Fprintf(os.Stderr, "she: invalid option '%s'\n", arg)
			fmt.Fprintf(os.Stderr, "Try 'she --help' for more information.\n")
			os.Exit(2)
		}
		viewSheet(arg)
	}
}

func viewSheet(name string) {
	if sheet.Exists(name) {
		path, err := sheet.Path(name)
		if err != nil {
			die("failed to resolve sheet path: %v", err)
		}
		if err := sheet.View(path); err != nil {
			die("failed to view sheet: %v", err)
		}
		return
	}
	fmt.Fprintf(os.Stderr, "No cheat sheet found for '%s'.\n", name)
	fmt.Fprintf(os.Stderr, "Run 'she -s' to pull sheets from remote, or 'she -e %s' to create one.\n", name)
}

// splitFlag splits "--edit=docker" into ("--edit", "docker").
// Returns (arg, "") if no "=" is present.
func splitFlag(arg string) (string, string) {
	if i := strings.IndexByte(arg, '='); i >= 0 {
		return arg[:i], arg[i+1:]
	}
	return arg, ""
}

// requireOperand returns inlineVal if non-empty, otherwise os.Args[2].
// Calls dieUsage with the given usage string if neither is available.
func requireOperand(inlineVal, usage string) string {
	if inlineVal != "" {
		return inlineVal
	}
	if len(os.Args) < 3 {
		dieUsage(usage)
	}
	return os.Args[2]
}

func printUsage() {
	fmt.Println(`she - cheat sheet manager

Usage:
  she <tool>                    View cheat sheet
  she -e, --edit <tool>         Edit cheat sheet (creates if missing)
  she -n, --new <tool>          Create new cheat sheet
  she -l, --list                List all cheat sheets
  she -s, --sync                Sync sheets with remote
  she -s, --sync <url>          Initialize sync with a remote git repo
  she -V, --version             Print version
  she -h, --help                Show this help

Examples:
  she docker                    View docker sheet
  she -e docker                 Edit docker sheet
  she --edit=docker             Edit docker sheet (long form with =)
  she -l                        Show all sheets
  she -s git@github.com:user/sheets.git
                                Initialize sync with remote
  she -s                        Push/pull changes
  she -- -sheet-name            View sheet whose name starts with '-'`)
}

func die(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "she: "+format+"\n", args...)
	os.Exit(1)
}

func dieUsage(usage string) {
	fmt.Fprintf(os.Stderr, "she: usage: she %s\n", usage)
	fmt.Fprintf(os.Stderr, "Try 'she --help' for more information.\n")
	os.Exit(2)
}
