package sheet

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

const defaultEditor = "vim"

func GetSheetsDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		die("cannot find home directory: %v", err)
	}
	return filepath.Join(home, ".sheets")
}

func GetSheetPath(name string) string {
	dir := GetSheetsDir()

	direct := filepath.Join(dir, name)
	if _, err := os.Stat(direct); err == nil {
		return direct
	}

	base := name
	if ext := filepath.Ext(name); ext != "" {
		base = strings.TrimSuffix(name, ext)
	}
	if _, err := os.Stat(filepath.Join(dir, base)); err == nil {
		return filepath.Join(dir, base)
	}

	return direct
}

func EnsureSheetsDir() error {
	dir := GetSheetsDir()
	return os.MkdirAll(dir, 0755)
}

func SheetExists(name string) bool {
	_, err := os.Stat(GetSheetPath(name))
	return err == nil
}

func CreateSheet(name string) (err error) {
	path := GetSheetPath(name)
	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("create sheet: %w", err)
	}
	defer func() {
		if cerr := f.Close(); cerr != nil {
			err = errors.Join(err, cerr)
		}
	}()

	template := "command > description\ncommand > description\n"
	_, err = f.WriteString(template)
	return err
}

func HandleEdit(name string) error {
	if err := EnsureSheetsDir(); err != nil {
		return fmt.Errorf("create sheets directory: %w", err)
	}

	path := GetSheetPath(name)

	exists := true
	if _, err := os.Stat(path); os.IsNotExist(err) {
		exists = false
		if err := CreateSheet(name); err != nil {
			return fmt.Errorf("create sheet: %w", err)
		}
	}

	if exists {
		fmt.Printf("Editing: %s\n", name)
	} else {
		fmt.Printf("Created: %s\n", name)
	}

	editor := getEditor()

	cmd := exec.Command(editor, path)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func HandleList() error {
	if err := EnsureSheetsDir(); err != nil {
		return fmt.Errorf("create sheets directory: %w", err)
	}

	dir := GetSheetsDir()

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

func HandleNew(name string) error {
	path := GetSheetPath(name)

	if _, err := os.Stat(path); err == nil {
		return fmt.Errorf("sheet %q already exists; use 'she edit %s' to edit", name, name)
	}

	if err := CreateSheet(name); err != nil {
		return fmt.Errorf("create sheet: %w", err)
	}

	fmt.Printf("Created: %s\n", name)

	editor := getEditor()

	cmd := exec.Command(editor, path)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func getEditor() string {
	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = os.Getenv("VISUAL")
	}
	if editor == "" {
		editor = defaultEditor
	}
	return editor
}

func die(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}