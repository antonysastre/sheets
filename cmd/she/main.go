// Command she is a small cheat-sheet manager. It stores plain-text sheets
// under ~/.sheets and prints them with simple formatting.
package main

import (
	"fmt"
	"os"

	"sheets/internal/sheet"
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
	case "edit", "e":
		if len(os.Args) < 3 {
			fmt.Println("Usage: she edit <tool>")
			os.Exit(1)
		}
		if err := sheet.Edit(os.Args[2]); err != nil {
			die("edit failed: %v", err)
		}

	case "new", "n":
		if len(os.Args) < 3 {
			fmt.Println("Usage: she new <tool>")
			os.Exit(1)
		}
		if err := sheet.New(os.Args[2]); err != nil {
			die("new failed: %v", err)
		}

	case "list", "ls", "l":
		if err := sheet.List(); err != nil {
			die("list failed: %v", err)
		}

	case "help", "h", "-h", "--help":
		printUsage()

	default:
		name := os.Args[1]
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

		fmt.Printf("No cheat sheet found for '%s'.\n", name)
		fmt.Println("Run 'she edit " + name + "' to create one.")
	}
}

func printUsage() {
	fmt.Println(`she - cheat sheet manager

Usage:
  she list		List all cheat sheets
  she <tool>		View cheat sheet
  she edit <tool>	Edit cheat sheet (creates if missing)
  she new <tool>	Create new cheat sheet
  she help		Show this help

Examples:
  she docker		View docker sheet
  she edit docker	Edit docker sheet
  she list		Show all sheets`)
}

func die(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}