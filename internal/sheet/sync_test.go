package sheet

// Most tests here are integration tests: they drive Sync against a real git
// binary and a real (but local, hence hermetic) bare repository standing in
// for the central remote. No network is involved. They are skipped under `go
// test -short` and when git is not on PATH. TestParseMarker is the exception
// — a pure unit test that always runs.

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// requireGit skips the test unless a real git binary is available and we are
// not running in -short mode.
func requireGit(t *testing.T) {
	t.Helper()
	if testing.Short() {
		t.Skip("skipping sync integration test in -short mode")
	}
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not found on PATH")
	}
}

// gitEnv makes git hermetic and deterministic for the duration of the test:
// it ignores the developer's global and system config and supplies a fixed
// identity, so commits succeed without depending on the host's setup.
func gitEnv(t *testing.T) {
	t.Helper()
	t.Setenv("GIT_CONFIG_GLOBAL", os.DevNull)
	t.Setenv("GIT_CONFIG_SYSTEM", os.DevNull)
	t.Setenv("GIT_AUTHOR_NAME", "she test")
	t.Setenv("GIT_AUTHOR_EMAIL", "test@example.com")
	t.Setenv("GIT_COMMITTER_NAME", "she test")
	t.Setenv("GIT_COMMITTER_EMAIL", "test@example.com")
}

// silenceOutput redirects the process's standard streams to /dev/null for the
// duration of the test, so Sync's progress output (and git's) does not clutter
// test runs. It is restored on cleanup. Safe because these tests never run in
// parallel — t.Setenv already forbids it.
func silenceOutput(t *testing.T) {
	t.Helper()
	devnull, err := os.OpenFile(os.DevNull, os.O_WRONLY, 0)
	if err != nil {
		t.Fatalf("open %s: %v", os.DevNull, err)
	}
	origOut, origErr := os.Stdout, os.Stderr
	os.Stdout, os.Stderr = devnull, devnull
	t.Cleanup(func() {
		os.Stdout, os.Stderr = origOut, origErr
		_ = devnull.Close()
	})
}

// mustGit runs git in dir (the test's working directory if dir is empty) and
// fails the test if it errors.
func mustGit(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", args...)
	if dir != "" {
		cmd.Dir = dir
	}
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %s: %v\n%s", strings.Join(args, " "), err, out)
	}
	return strings.TrimSpace(string(out))
}

// newCentralRepo creates an empty bare repository to stand in for the shared
// sync remote.
func newCentralRepo(t *testing.T) string {
	t.Helper()
	dir := filepath.Join(t.TempDir(), "central.git")
	mustGit(t, "", "init", "--bare", "-b", syncBranch, dir)
	return dir
}

// seedRemote pushes a commit containing files (name -> content) to the bare
// repository at central, so a test can simulate a remote that already has
// history — be it an established sheets repo or an unrelated project.
func seedRemote(t *testing.T, central string, files map[string]string) {
	t.Helper()
	work := t.TempDir()
	mustGit(t, "", "init", "-b", syncBranch, work)
	mustGit(t, work, "remote", "add", "origin", central)
	for name, content := range files {
		if err := os.WriteFile(filepath.Join(work, name), []byte(content), 0o644); err != nil {
			t.Fatalf("seedRemote: write %s: %v", name, err)
		}
	}
	mustGit(t, work, "add", "-A")
	mustGit(t, work, "commit", "-m", "seed")
	mustGit(t, work, "push", "origin", syncBranch)
}

// laptop is a fake machine: a HOME directory whose ~/.sheets is synced.
type laptop struct {
	home   string
	sheets string
}

func newLaptop(t *testing.T, name string) *laptop {
	t.Helper()
	home := filepath.Join(t.TempDir(), name)
	sheets := filepath.Join(home, ".sheets")
	if err := os.MkdirAll(sheets, 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", sheets, err)
	}
	return &laptop{home: home, sheets: sheets}
}

// sync runs she's sync as this laptop by pointing HOME at it, then calling
// Sync. repo is passed straight through: non-empty for setup, empty for a
// routine sync.
func (l *laptop) sync(t *testing.T, repo string) error {
	t.Helper()
	t.Setenv("HOME", l.home)
	return Sync(repo)
}

func (l *laptop) writeSheet(t *testing.T, name, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(l.sheets, name), []byte(content), 0o644); err != nil {
		t.Fatalf("write sheet %s: %v", name, err)
	}
}

func (l *laptop) readSheet(t *testing.T, name string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(l.sheets, name))
	if err != nil {
		t.Fatalf("read sheet %s: %v", name, err)
	}
	return string(data)
}

func (l *laptop) sheetExists(name string) bool {
	_, err := os.Stat(filepath.Join(l.sheets, name))
	return err == nil
}

// TestSyncSetupAndPropagation covers the happy path: two machines set up
// syncing against the same remote and changes flow both ways, with a final
// no-op sync proving an idempotent run is not an error.
func TestSyncSetupAndPropagation(t *testing.T) {
	requireGit(t)
	gitEnv(t)
	silenceOutput(t)

	central := newCentralRepo(t)
	a := newLaptop(t, "laptopA")
	b := newLaptop(t, "laptopB")

	// Laptop A: create sheets, then set up syncing.
	a.writeSheet(t, "git", "git status > show tree\n")
	a.writeSheet(t, "docker", "docker ps > list containers\n")
	if err := a.sync(t, central); err != nil {
		t.Fatalf("laptop A setup: %v", err)
	}

	// Laptop B: create its own sheet, then set up syncing. It should come
	// away with A's sheets merged alongside its own.
	b.writeSheet(t, "k8s", "kubectl get pods > list pods\n")
	if err := b.sync(t, central); err != nil {
		t.Fatalf("laptop B setup: %v", err)
	}
	for _, name := range []string{"git", "docker", "k8s"} {
		if !b.sheetExists(name) {
			t.Errorf("after setup, laptop B is missing sheet %q", name)
		}
	}

	// Laptop A: a routine sync should pull B's new sheet down.
	if err := a.sync(t, ""); err != nil {
		t.Fatalf("laptop A routine sync: %v", err)
	}
	if !a.sheetExists("k8s") {
		t.Error("after sync, laptop A is missing sheet \"k8s\" from laptop B")
	}

	// An edit on A should reach B through the remote.
	const editedGit = "git status > show tree\ngit log > history\n"
	a.writeSheet(t, "git", editedGit)
	if err := a.sync(t, ""); err != nil {
		t.Fatalf("laptop A sync after edit: %v", err)
	}
	if err := b.sync(t, ""); err != nil {
		t.Fatalf("laptop B sync after A's edit: %v", err)
	}
	if got := b.readSheet(t, "git"); got != editedGit {
		t.Errorf("laptop B git sheet = %q, want %q", got, editedGit)
	}

	// A sync with nothing to do must succeed and change nothing.
	if err := b.sync(t, ""); err != nil {
		t.Fatalf("no-op sync: %v", err)
	}
	if got := b.readSheet(t, "git"); got != editedGit {
		t.Errorf("after no-op sync, laptop B git sheet = %q, want %q", got, editedGit)
	}
}

// TestSyncSetupOntoPopulatedRemote covers a fresh machine with no sheets of
// its own joining a remote that another machine already populated. Setup must
// adopt the remote's sheets wholesale rather than try to merge an empty local
// history.
func TestSyncSetupOntoPopulatedRemote(t *testing.T) {
	requireGit(t)
	gitEnv(t)
	silenceOutput(t)

	central := newCentralRepo(t)
	a := newLaptop(t, "laptopA")
	b := newLaptop(t, "laptopB")

	// Laptop A populates the remote.
	a.writeSheet(t, "git", "git status > show tree\n")
	a.writeSheet(t, "docker", "docker ps > list containers\n")
	if err := a.sync(t, central); err != nil {
		t.Fatalf("laptop A setup: %v", err)
	}

	// Laptop B has no sheets at all; setup should pull A's down.
	if err := b.sync(t, central); err != nil {
		t.Fatalf("laptop B setup onto populated remote: %v", err)
	}
	for _, name := range []string{"git", "docker"} {
		if !b.sheetExists(name) {
			t.Errorf("after setup, laptop B is missing sheet %q", name)
		}
	}

	// The branch must also be wired up for routine syncs afterwards.
	if err := b.sync(t, ""); err != nil {
		t.Fatalf("laptop B routine sync after setup: %v", err)
	}
}

// TestSyncSetupRerun covers running setup again on a directory that is already
// a sync repository — for instance to point it at a different remote. The
// origin URL must be updated in place and the existing sheets pushed to the
// new remote.
func TestSyncSetupRerun(t *testing.T) {
	requireGit(t)
	gitEnv(t)
	silenceOutput(t)

	first := newCentralRepo(t)
	second := newCentralRepo(t)
	a := newLaptop(t, "laptop")

	// Initial setup against the first remote.
	a.writeSheet(t, "git", "git status > show tree\n")
	if err := a.sync(t, first); err != nil {
		t.Fatalf("initial setup: %v", err)
	}

	// Re-run setup pointing at a different remote.
	if err := a.sync(t, second); err != nil {
		t.Fatalf("setup re-run: %v", err)
	}

	// origin must now point at the second remote...
	if got := mustGit(t, a.sheets, "remote", "get-url", "origin"); got != second {
		t.Errorf("origin = %q, want %q", got, second)
	}
	// ...and the existing sheet must have been pushed there.
	if tree := mustGit(t, second, "ls-tree", "-r", "--name-only", syncBranch); !strings.Contains(tree, "git") {
		t.Errorf("second remote does not contain the \"git\" sheet; tree:\n%s", tree)
	}
}

// TestSyncConflict covers the case the routine is built to handle safely: the
// same sheet edited on two machines between syncs. The second machine's sync
// must fail clearly, name the sheet, and leave that machine's commit and
// working tree intact — no half-finished rebase.
func TestSyncConflict(t *testing.T) {
	requireGit(t)
	gitEnv(t)
	silenceOutput(t)

	central := newCentralRepo(t)
	a := newLaptop(t, "laptopA")
	b := newLaptop(t, "laptopB")

	// Both laptops start from the same synced state.
	a.writeSheet(t, "git", "git status > show tree\n")
	if err := a.sync(t, central); err != nil {
		t.Fatalf("laptop A setup: %v", err)
	}
	if err := b.sync(t, central); err != nil {
		t.Fatalf("laptop B setup: %v", err)
	}

	// Both edit the same sheet differently; A syncs first and wins.
	a.writeSheet(t, "git", "git status > A's version\n")
	const bVersion = "git status > B's version\n"
	b.writeSheet(t, "git", bVersion)
	if err := a.sync(t, ""); err != nil {
		t.Fatalf("laptop A sync: %v", err)
	}

	// B's sync must fail with a conflict that names the sheet.
	err := b.sync(t, "")
	if err == nil {
		t.Fatal("laptop B sync: expected a conflict error, got nil")
	}
	if !strings.Contains(err.Error(), "the same sheet was edited") {
		t.Errorf("error %q is not the expected conflict error", err)
	}
	if !strings.Contains(err.Error(), "\n  git") {
		t.Errorf("conflict error %q does not list the conflicting sheet %q", err, "git")
	}

	// B's own edit must still be on disk, untouched.
	if got := b.readSheet(t, "git"); got != bVersion {
		t.Errorf("after aborted sync, laptop B git sheet = %q, want %q", got, bVersion)
	}

	// The rebase must have been aborted cleanly: no rebase state left behind
	// and a clean working tree.
	for _, leftover := range []string{"rebase-merge", "rebase-apply"} {
		if _, err := os.Stat(filepath.Join(b.sheets, ".git", leftover)); !os.IsNotExist(err) {
			t.Errorf("rebase state %q left behind in %s/.git", leftover, b.sheets)
		}
	}
	if status := mustGit(t, b.sheets, "status", "--porcelain"); status != "" {
		t.Errorf("laptop B working tree not clean after aborted sync:\n%s", status)
	}

	// B's local commit is preserved — still one commit ahead of the remote.
	if ahead := mustGit(t, b.sheets, "rev-list", "--count", "origin/"+syncBranch+"..HEAD"); ahead != "1" {
		t.Errorf("laptop B is %s commits ahead of the remote, want 1", ahead)
	}
}

// TestSyncWithoutSetup verifies that a routine sync before setup fails with a
// message pointing the user at the setup command, rather than doing anything
// surprising.
func TestSyncWithoutSetup(t *testing.T) {
	requireGit(t)
	gitEnv(t)
	silenceOutput(t)

	l := newLaptop(t, "laptop")
	err := l.sync(t, "")
	if err == nil {
		t.Fatal("sync without setup: expected an error, got nil")
	}
	if !strings.Contains(err.Error(), "she --sync <repo>") {
		t.Errorf("error %q should tell the user to run setup first", err)
	}
}

// TestSyncSetupWritesMarker verifies that establishing a repository writes a
// well-formed marker file, both locally and on the remote.
func TestSyncSetupWritesMarker(t *testing.T) {
	requireGit(t)
	gitEnv(t)
	silenceOutput(t)

	central := newCentralRepo(t)
	a := newLaptop(t, "laptop")
	a.writeSheet(t, "git", "git status > show tree\n")
	if err := a.sync(t, central); err != nil {
		t.Fatalf("setup: %v", err)
	}

	// Marker present locally and well-formed.
	data, err := os.ReadFile(filepath.Join(a.sheets, markerName))
	if err != nil {
		t.Fatalf("read local marker: %v", err)
	}
	id, ok := parseMarker(string(data))
	if !ok {
		t.Fatalf("local marker is not well-formed:\n%s", data)
	}
	if len(id) != 64 {
		t.Errorf("marker id = %q, want 64 hex chars", id)
	}

	// Marker present on the remote with the same id.
	remoteID, ok := parseMarker(mustGit(t, central, "show", syncBranch+":"+markerName))
	if !ok {
		t.Fatal("remote marker is not well-formed")
	}
	if remoteID != id {
		t.Errorf("remote marker id = %q, want %q", remoteID, id)
	}
}

// TestSyncSetupRejectsForeignRepo covers the footgun the marker is built to
// stop: pointing setup at an existing repo that is actually a different
// project. Setup must refuse before any merge or push, and roll the
// half-initialised ~/.sheets back so the user's files are untouched.
func TestSyncSetupRejectsForeignRepo(t *testing.T) {
	requireGit(t)
	gitEnv(t)
	silenceOutput(t)

	central := newCentralRepo(t)
	seedRemote(t, central, map[string]string{
		"main.go":   "package main\n",
		"README.md": "# some other project\n",
	})

	a := newLaptop(t, "laptop")
	a.writeSheet(t, "git", "git status > show tree\n")

	err := a.sync(t, central)
	if err == nil {
		t.Fatal("setup against a non-empty repo: expected an error, got nil")
	}
	if !strings.Contains(err.Error(), "only initialises an empty repository") {
		t.Errorf("error %q is not the expected non-empty-remote rejection", err)
	}
	// The half-initialised repository must have been rolled back...
	if _, statErr := os.Stat(filepath.Join(a.sheets, ".git")); !os.IsNotExist(statErr) {
		t.Errorf("expected %s/.git to be removed after rejection", a.sheets)
	}
	// ...and the user's own sheet left untouched.
	if !a.sheetExists("git") {
		t.Error("user's sheet was removed during rollback")
	}
}

// TestSyncMarkerSharedAcrossLaptops verifies that a second machine joining an
// established repository adopts the same repository id, so the two are bound
// to the same sheets repo.
func TestSyncMarkerSharedAcrossLaptops(t *testing.T) {
	requireGit(t)
	gitEnv(t)
	silenceOutput(t)

	central := newCentralRepo(t)
	a := newLaptop(t, "laptopA")
	b := newLaptop(t, "laptopB")

	a.writeSheet(t, "git", "git status > show tree\n")
	if err := a.sync(t, central); err != nil {
		t.Fatalf("laptop A setup: %v", err)
	}
	b.writeSheet(t, "docker", "docker ps > list containers\n")
	if err := b.sync(t, central); err != nil {
		t.Fatalf("laptop B setup: %v", err)
	}

	aID, ok := localMarkerID(a.sheets)
	if !ok {
		t.Fatal("laptop A has no valid marker")
	}
	bID, ok := localMarkerID(b.sheets)
	if !ok {
		t.Fatal("laptop B has no valid marker")
	}
	if aID != bID {
		t.Errorf("laptop A id %q != laptop B id %q — not bound to the same repo", aID, bID)
	}
}

// TestSyncRunRejectsIdentityMismatch verifies that a routine sync refuses when
// ~/.sheets has been re-pointed at a different sheets repository.
func TestSyncRunRejectsIdentityMismatch(t *testing.T) {
	requireGit(t)
	gitEnv(t)
	silenceOutput(t)

	repo1 := newCentralRepo(t)
	repo2 := newCentralRepo(t)

	// Set the laptop up against repo1.
	a := newLaptop(t, "laptop")
	a.writeSheet(t, "git", "git status > show tree\n")
	if err := a.sync(t, repo1); err != nil {
		t.Fatalf("setup against repo1: %v", err)
	}

	// Independently establish repo2 as a different sheets repo, with its own
	// marker id, via a separate laptop.
	other := newLaptop(t, "other")
	other.writeSheet(t, "docker", "docker ps > list containers\n")
	if err := other.sync(t, repo2); err != nil {
		t.Fatalf("setup repo2 via other laptop: %v", err)
	}

	// Re-point the first laptop's origin at repo2 and run a routine sync.
	mustGit(t, a.sheets, "remote", "set-url", "origin", repo2)
	err := a.sync(t, "")
	if err == nil {
		t.Fatal("routine sync against a mismatched repo: expected an error, got nil")
	}
	if !strings.Contains(err.Error(), "mismatched repositories") {
		t.Errorf("error %q is not the expected identity-mismatch error", err)
	}
}

// TestSyncSetupMergeConflict covers a genuine sheet conflict during setup: two
// machines independently created the same sheet with different content. Setup
// must fail clearly and leave the merge in place for the user to resolve.
func TestSyncSetupMergeConflict(t *testing.T) {
	requireGit(t)
	gitEnv(t)
	silenceOutput(t)

	central := newCentralRepo(t)
	a := newLaptop(t, "laptopA")
	b := newLaptop(t, "laptopB")

	// Laptop A establishes the repo with a "git" sheet.
	a.writeSheet(t, "git", "git status > A's version\n")
	if err := a.sync(t, central); err != nil {
		t.Fatalf("laptop A setup: %v", err)
	}

	// Laptop B has its own "git" sheet with different content.
	b.writeSheet(t, "git", "git status > B's version\n")
	err := b.sync(t, central)
	if err == nil {
		t.Fatal("setup with a conflicting sheet: expected an error, got nil")
	}
	if !strings.Contains(err.Error(), "the same sheet differs") {
		t.Errorf("error %q is not the expected setup-conflict error", err)
	}
	// The conflicted merge must be left in place for the user to resolve.
	if _, statErr := os.Stat(filepath.Join(b.sheets, ".git", "MERGE_HEAD")); statErr != nil {
		t.Errorf("expected an unresolved merge in %s/.git for the user to resolve", b.sheets)
	}
}

// TestSyncSetupRejectsDifferentRepoOnRerun covers re-running setup against a
// different sheets repository than the one ~/.sheets is already bound to. The
// marker ids differ, so setup must refuse, abort the merge, and restore the
// original remote.
func TestSyncSetupRejectsDifferentRepoOnRerun(t *testing.T) {
	requireGit(t)
	gitEnv(t)
	silenceOutput(t)

	repoA := newCentralRepo(t)
	repoB := newCentralRepo(t)

	// One laptop establishes repoA.
	a := newLaptop(t, "laptopA")
	a.writeSheet(t, "git", "git status > show tree\n")
	if err := a.sync(t, repoA); err != nil {
		t.Fatalf("laptop A setup: %v", err)
	}

	// Another laptop establishes repoB — a distinct sheets repo with its own
	// marker id.
	b := newLaptop(t, "laptopB")
	b.writeSheet(t, "docker", "docker ps > list containers\n")
	if err := b.sync(t, repoB); err != nil {
		t.Fatalf("laptop B setup: %v", err)
	}

	// Re-running B's setup against repoA must be refused: different identity.
	err := b.sync(t, repoA)
	if err == nil {
		t.Fatal("setup re-pointed at a different sheets repo: expected an error, got nil")
	}
	if !strings.Contains(err.Error(), "different sheets repositories") {
		t.Errorf("error %q is not the expected marker-mismatch error", err)
	}
	// The original remote must have been restored by the rollback.
	if got := mustGit(t, b.sheets, "remote", "get-url", "origin"); got != repoB {
		t.Errorf("origin = %q after rejection, want it restored to %q", got, repoB)
	}
	// No half-finished merge left behind.
	if _, statErr := os.Stat(filepath.Join(b.sheets, ".git", "MERGE_HEAD")); !os.IsNotExist(statErr) {
		t.Errorf("expected the merge to be aborted, but MERGE_HEAD remains in %s/.git", b.sheets)
	}
}

// TestParseMarker checks marker parsing in isolation, without any git.
func TestParseMarker(t *testing.T) {
	const goodID = "ab0a7fdface20d8549d02fe4c90f16a7dd33b7f3586ea152229c1b53dc1f29ac"
	tests := []struct {
		name    string
		content string
		wantID  string
		wantOK  bool
	}{
		{"well-formed", markerBody(goodID), goodID, true},
		{"id only", "id: " + goodID + "\n", goodID, true},
		{"extra whitespace tolerated", "  id:  " + goodID + "  \n", goodID, true},
		{"last id wins", "id: " + strings.Repeat("a", 64) + "\nid: " + goodID + "\n", goodID, true},
		{"missing id", "version: 1\n", "", false},
		{"id too short", "id: abc123\n", "", false},
		{"id not hex", "id: " + strings.Repeat("z", 64) + "\n", "", false},
		{"empty", "", "", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			id, ok := parseMarker(tt.content)
			if id != tt.wantID || ok != tt.wantOK {
				t.Errorf("parseMarker(%q) = (%q, %v), want (%q, %v)",
					tt.content, id, ok, tt.wantID, tt.wantOK)
			}
		})
	}
}
