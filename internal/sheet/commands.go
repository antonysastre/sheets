// Package sheet implements the storage and command operations for she's
// cheat sheets: locating the sheets directory, creating sheets from a
// template, and invoking the user's editor.
package sheet

import (
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

const defaultEditor = "vim"

// Dir returns the absolute path to the directory where sheets are stored
// (~/.sheets), or an error if the user's home directory cannot be determined.
func Dir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("locate home directory: %w", err)
	}
	return filepath.Join(home, ".sheets"), nil
}

// Path returns the on-disk path for the sheet identified by name. If a file
// with the exact name exists it is returned; otherwise the extension is
// stripped and the bare-name path is tried. When neither exists, the
// exact-name path is returned so callers can use it as a creation target.
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

// EnsureDir creates the sheets directory if it does not already exist.
func EnsureDir() error {
	dir, err := Dir()
	if err != nil {
		return err
	}
	return os.MkdirAll(dir, 0o755)
}

// Exists reports whether a sheet with the given name is stored on disk.
func Exists(name string) bool {
	path, err := Path(name)
	if err != nil {
		return false
	}
	_, err = os.Stat(path)
	return err == nil
}

// Create writes a new sheet file for name, populated with a small template.
// It is an error to call Create for a sheet that already exists.
func Create(name string) (err error) {
	path, err := Path(name)
	if err != nil {
		return err
	}
	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("create sheet: %w", err)
	}
	// Named return + errors.Join so a Close failure isn't silently dropped
	// when WriteString already succeeded.
	defer func() {
		if cerr := f.Close(); cerr != nil {
			err = errors.Join(err, cerr)
		}
	}()

	const template = "Section\ncommand # description\ncommand # description\n"
	_, err = f.WriteString(template)
	return err
}

// Edit opens the sheet for name in the user's editor, creating it from the
// template first if it does not yet exist. Status messages are written to
// stderr.
func Edit(stderr io.Writer, name string) error {
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
		_, _ = fmt.Fprintf(stderr, "Editing: %s\n", name)
	} else {
		_, _ = fmt.Fprintf(stderr, "Created: %s\n", name)
	}

	cmd := exec.Command(editor(), path)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// List prints the names of all stored sheets to w.
func List(w io.Writer) error {
	if err := EnsureDir(); err != nil {
		return fmt.Errorf("create sheets directory: %w", err)
	}

	dir, err := Dir()
	if err != nil {
		return err
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		return fmt.Errorf("read directory: %w", err)
	}

	if len(entries) == 0 {
		_, _ = fmt.Fprintln(w, "No sheets found. Run 'she --edit <tool>' to create one.")
		return nil
	}

	_, _ = fmt.Fprintln(w, ansiCyanBold+"Available sheets:"+ansiReset)
	for _, entry := range entries {
		if entry.IsDir() || strings.HasPrefix(entry.Name(), ".") {
			continue
		}
		_, _ = fmt.Fprintf(w, "  %s%s%s\n", ansiGreen, entry.Name(), ansiReset)
	}
	return nil
}

// Initialize creates a fresh sheet for name and opens it in the editor.
// Status messages are written to stderr. Returns an error if a sheet with
// that name already exists.
func Initialize(stderr io.Writer, name string) error {
	path, err := Path(name)
	if err != nil {
		return err
	}

	if _, err := os.Stat(path); err == nil {
		return fmt.Errorf("sheet %q already exists; use 'she --edit %s' to edit", name, name)
	}

	if err := Create(name); err != nil {
		return fmt.Errorf("create sheet: %w", err)
	}

	_, _ = fmt.Fprintf(stderr, "Created: %s\n", name)

	cmd := exec.Command(editor(), path)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// editor resolves the user's preferred editor, following the long-standing
// Unix convention of consulting VISUAL before EDITOR.
func editor() string {
	if e := os.Getenv("VISUAL"); e != "" {
		return e
	}
	if e := os.Getenv("EDITOR"); e != "" {
		return e
	}
	return defaultEditor
}
