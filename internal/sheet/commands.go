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

func Dir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		die("cannot find home directory: %v", err)
	}
	return filepath.Join(home, ".sheets")
}

func Path(name string) string {
	dir := Dir()

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

func EnsureDir() error {
	dir := Dir()
	return os.MkdirAll(dir, 0755)
}

func Exists(name string) bool {
	_, err := os.Stat(Path(name))
	return err == nil
}

func Create(name string) (err error) {
	path := Path(name)
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

func Edit(name string) error {
	if err := EnsureDir(); err != nil {
		return fmt.Errorf("create sheets directory: %w", err)
	}

	path := Path(name)

	exists := true
	if _, err := os.Stat(path); os.IsNotExist(err) {
		exists = false
		if err := Create(name); err != nil {
			return fmt.Errorf("create sheet: %w", err)
		}
	}

	if exists {
		fmt.Printf("Editing: %s\n", name)
	} else {
		fmt.Printf("Created: %s\n", name)
	}

	cmd := exec.Command(editor(), path)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func List() error {
	if err := EnsureDir(); err != nil {
		return fmt.Errorf("create sheets directory: %w", err)
	}

	dir := Dir()

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

func New(name string) error {
	path := Path(name)

	if _, err := os.Stat(path); err == nil {
		return fmt.Errorf("sheet %q already exists; use 'she edit %s' to edit", name, name)
	}

	if err := Create(name); err != nil {
		return fmt.Errorf("create sheet: %w", err)
	}

	fmt.Printf("Created: %s\n", name)

	cmd := exec.Command(editor(), path)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func editor() string {
	if e := os.Getenv("EDITOR"); e != "" {
		return e
	}
	if e := os.Getenv("VISUAL"); e != "" {
		return e
	}
	return defaultEditor
}

func die(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}