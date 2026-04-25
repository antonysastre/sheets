package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

const defaultEditor = "vim"

func handleEdit(name string) error {
	if err := ensureSheetsDir(); err != nil {
		return fmt.Errorf("create sheets directory: %w", err)
	}

	path := getSheetPath(name)

	exists := true
	if _, err := os.Stat(path); os.IsNotExist(err) {
		exists = false
		if err := createSheet(name); err != nil {
			return fmt.Errorf("create sheet: %w", err)
		}
	}

	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = defaultEditor
	}

	if exists {
		fmt.Printf("Editing: %s\n", name)
	} else {
		fmt.Printf("Created: %s\n", name)
	}

	cmd := exec.Command(editor, path)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func handleList() error {
	if err := ensureSheetsDir(); err != nil {
		return fmt.Errorf("create sheets directory: %w", err)
	}

	dir := getSheetsDir()

	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			fmt.Println("No sheets found. Run 'she edit <tool>' to create one.")
			return nil
		}
		return fmt.Errorf("read directory: %w", err)
	}

	if len(entries) == 0 {
		fmt.Println("No sheets found. Run 'she edit <tool>' to create one.")
		return nil
	}

	fmt.Println("\033[36;1mAvailable sheets:\033[0m")
	for _, entry := range entries {
		if entry.IsDir() || strings.HasPrefix(entry.Name(), ".") {
			continue
		}
		fmt.Printf("  \033[32m%s\033[0m\n", entry.Name())
	}
	return nil
}

func handleNew(name string) error {
	path := getSheetPath(name)

	if _, err := os.Stat(path); err == nil {
		return fmt.Errorf("sheet '%s' already exists. Use 'she edit %s' to edit.", name, name)
	}

	if err := createSheet(name); err != nil {
		return fmt.Errorf("create sheet: %w", err)
	}

	fmt.Printf("Created: %s\n", name)

	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = defaultEditor
	}

	cmd := exec.Command(editor, path)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
