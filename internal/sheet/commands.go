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

func Dir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("locate home directory: %w", err)
	}
	return filepath.Join(home, ".sheets"), nil
}

func Path(name string) (string, error) {
	dir, err := Dir()
	if err != nil {
		return "", err
	}

	direct := filepath.Join(dir, name)
	if _, err := os.Stat(direct); err == nil {
		return direct, nil
	}

	base := name
	if ext := filepath.Ext(name); ext != "" {
		base = strings.TrimSuffix(name, ext)
	}
	if _, err := os.Stat(filepath.Join(dir, base)); err == nil {
		return filepath.Join(dir, base), nil
	}

	return direct, nil
}

func EnsureDir() error {
	dir, err := Dir()
	if err != nil {
		return err
	}
	return os.MkdirAll(dir, 0755)
}

func Exists(name string) bool {
	path, err := Path(name)
	if err != nil {
		return false
	}
	_, err = os.Stat(path)
	return err == nil
}

func Create(name string) (err error) {
	path, err := Path(name)
	if err != nil {
		return err
	}
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

	path, err := Path(name)
	if err != nil {
		return err
	}

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

	dir, err := Dir()
	if err != nil {
		return err
	}

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
	path, err := Path(name)
	if err != nil {
		return err
	}

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
