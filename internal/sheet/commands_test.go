package sheet

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// withHome redirects HOME to a per-test temp directory so each test gets a
// fresh ~/.sheets.
func withHome(t *testing.T) string {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	return home
}

func TestDir(t *testing.T) {
	home := withHome(t)
	got, err := Dir()
	if err != nil {
		t.Fatalf("Dir() error = %v", err)
	}
	want := filepath.Join(home, ".sheets")
	if got != want {
		t.Errorf("Dir() = %q, want %q", got, want)
	}
}

func TestEnsureDirIsIdempotent(t *testing.T) {
	withHome(t)
	for i := 0; i < 2; i++ {
		if err := EnsureDir(); err != nil {
			t.Fatalf("EnsureDir() call %d error = %v", i+1, err)
		}
	}
	dir, _ := Dir()
	info, err := os.Stat(dir)
	if err != nil {
		t.Fatalf("Stat(%q) error = %v", dir, err)
	}
	if !info.IsDir() {
		t.Errorf("expected %q to be a directory", dir)
	}
}

func TestExists(t *testing.T) {
	withHome(t)
	if err := EnsureDir(); err != nil {
		t.Fatalf("EnsureDir() error = %v", err)
	}
	if Exists("missing") {
		t.Errorf("Exists(\"missing\") = true, want false")
	}
	if err := Create("present"); err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if !Exists("present") {
		t.Errorf("Exists(\"present\") = false, want true")
	}
}

func TestCreateWritesTemplate(t *testing.T) {
	withHome(t)
	if err := EnsureDir(); err != nil {
		t.Fatalf("EnsureDir() error = %v", err)
	}
	if err := Create("docker"); err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	path, err := Path("docker")
	if err != nil {
		t.Fatalf("Path() error = %v", err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile(%q) error = %v", path, err)
	}
	if !strings.Contains(string(data), " > ") {
		t.Errorf("template missing ' > ' separator: %q", data)
	}
}

func TestPathExtensionFallback(t *testing.T) {
	withHome(t)
	if err := EnsureDir(); err != nil {
		t.Fatalf("EnsureDir() error = %v", err)
	}
	// Create a sheet at the bare name "ls"; looking up "ls.sh" should fall
	// back to "ls" because the exact-name file does not exist.
	if err := Create("ls"); err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	got, err := Path("ls.sh")
	if err != nil {
		t.Fatalf("Path(\"ls.sh\") error = %v", err)
	}
	dir, _ := Dir()
	want := filepath.Join(dir, "ls")
	if got != want {
		t.Errorf("Path(\"ls.sh\") = %q, want %q", got, want)
	}
}

func TestPathReturnsExactNameForCreationTarget(t *testing.T) {
	withHome(t)
	if err := EnsureDir(); err != nil {
		t.Fatalf("EnsureDir() error = %v", err)
	}
	// Neither "docker" nor "docker" with a stripped suffix exists. Path
	// should return the exact-name path so the caller can create it there.
	got, err := Path("docker")
	if err != nil {
		t.Fatalf("Path() error = %v", err)
	}
	dir, _ := Dir()
	want := filepath.Join(dir, "docker")
	if got != want {
		t.Errorf("Path(\"docker\") = %q, want %q", got, want)
	}
}

func TestNewFailsWhenSheetExists(t *testing.T) {
	withHome(t)
	if err := EnsureDir(); err != nil {
		t.Fatalf("EnsureDir() error = %v", err)
	}
	if err := Create("git"); err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	err := New("git")
	if err == nil {
		t.Fatalf("New(\"git\") on existing sheet returned nil error")
	}
	if !strings.Contains(err.Error(), "already exists") {
		t.Errorf("New() error = %q, want substring %q", err.Error(), "already exists")
	}
	if !strings.Contains(err.Error(), "she --edit") {
		t.Errorf("New() error = %q, want substring %q", err.Error(), "she --edit")
	}
}

func TestEditorPrefersVISUALOverEDITOR(t *testing.T) {
	t.Setenv("VISUAL", "nano")
	t.Setenv("EDITOR", "vim")
	if got := editor(); got != "nano" {
		t.Errorf("editor() = %q, want %q", got, "nano")
	}
}

func TestEditorFallsBackToEDITOR(t *testing.T) {
	t.Setenv("VISUAL", "")
	t.Setenv("EDITOR", "vim")
	if got := editor(); got != "vim" {
		t.Errorf("editor() = %q, want %q", got, "vim")
	}
}

func TestEditorDefaultsWhenUnset(t *testing.T) {
	t.Setenv("VISUAL", "")
	t.Setenv("EDITOR", "")
	if got := editor(); got != defaultEditor {
		t.Errorf("editor() = %q, want %q", got, defaultEditor)
	}
}
