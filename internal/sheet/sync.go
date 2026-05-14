package sheet

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// syncBranch is the branch used for the central sheets repository. The
// repository passed to 'she --sync <repo>' is expected to use this branch.
const syncBranch = "main"

const (
	// markerName is the file she writes at the root of a sheets repository to
	// identify it as one. It is a dotfile, so sheet.List ignores it.
	markerName = ".shetag"
	// markerSignature is the first line of the marker file.
	markerSignature = "# she sheets repository"
	// markerVersion is the current marker format version.
	markerVersion = 1
)

// Sync synchronises the local sheets with a central git repository.
//
// Called with a non-empty repo it performs first-time setup: it turns the
// sheets directory into a git repository, points it at repo, and performs an
// initial exchange so both ends share a common history. Called with an empty
// repo it runs the routine sync: commit local changes, fetch and rebase onto
// the remote, then push.
func Sync(repo string) error {
	if err := EnsureDir(); err != nil {
		return fmt.Errorf("create sheets directory: %w", err)
	}
	dir, err := Dir()
	if err != nil {
		return err
	}
	if _, err := exec.LookPath("git"); err != nil {
		return errors.New("git is not installed or not on PATH")
	}

	if repo != "" {
		return syncSetup(dir, repo)
	}
	return syncRun(dir)
}

// syncSetup wires the sheets directory up to repo and performs the first
// exchange so the local sheets and any already on the remote are merged into
// a single shared history.
//
// Before merging anything it guards against repo being an unrelated project:
// a remote that already has history must carry a she marker file, otherwise
// setup would graft that project into ~/.sheets and push the sheets onto its
// branch. On rejection the half-wired repository is rolled back.
func syncSetup(dir, repo string) error {
	createdGit := false
	previousURL := ""

	if isGitRepo(dir) {
		if url, err := runGit(dir, "remote", "get-url", "origin"); err == nil {
			previousURL = url
			if _, err := runGit(dir, "remote", "set-url", "origin", repo); err != nil {
				return err
			}
		} else if _, err := runGit(dir, "remote", "add", "origin", repo); err != nil {
			return err
		}
		fmt.Printf("Updated sync remote to %s\n", repo)
	} else {
		if _, err := runGit(dir, "init", "-b", syncBranch); err != nil {
			return err
		}
		createdGit = true
		if _, err := runGit(dir, "remote", "add", "origin", repo); err != nil {
			return err
		}
		fmt.Printf("Initialised sync repository in %s\n", dir)
	}

	// rollback undoes the repository changes made above, so a rejected setup
	// does not leave ~/.sheets half-wired to the wrong remote.
	rollback := func() {
		if createdGit {
			_ = os.RemoveAll(filepath.Join(dir, ".git"))
			return
		}
		if previousURL != "" {
			_, _ = runGit(dir, "remote", "set-url", "origin", previousURL)
		}
	}

	fmt.Println("Fetching remote sheets...")
	if err := streamGit(dir, "fetch", "origin"); err != nil {
		rollback()
		return err
	}

	if remoteBranchExists(dir, syncBranch) {
		// The remote already has history: it must be a she sheets repository,
		// not some unrelated project we would otherwise merge into ~/.sheets
		// and then push our sheets onto.
		_, present, err := remoteMarkerID(dir)
		if err != nil {
			rollback()
			return err
		}
		if !present {
			rollback()
			return fmt.Errorf("the remote %s is not a she sheets repository "+
				"(no %s marker) — point --sync at a dedicated, empty repository; "+
				"or, if it really is your sheets repo, add a %s marker to it and retry",
				repo, markerName, markerName)
		}
	} else {
		// The remote is empty: we are establishing it. Make sure ~/.sheets
		// carries a marker so the repository has an identity from the start.
		if err := ensureLocalMarker(dir); err != nil {
			rollback()
			return err
		}
	}

	if _, err := commitLocal(dir, "Add sheets from "+hostname()); err != nil {
		return err
	}

	switch {
	case remoteBranchExists(dir, syncBranch) && hasCommits(dir):
		// Both ends have history: stitch them together. An established sheets
		// repo brings its marker in through this merge.
		if _, err := runGit(dir, "merge", "origin/"+syncBranch,
			"--allow-unrelated-histories", "-m", "Merge remote sheets"); err != nil {
			return fmt.Errorf("the same sheet differs on both ends — "+
				"resolve the conflict in %s, then run 'she --sync': %w", dir, err)
		}
	case remoteBranchExists(dir, syncBranch):
		// No local history yet: adopt the remote branch as-is.
		if _, err := runGit(dir, "checkout", "-B", syncBranch, "origin/"+syncBranch); err != nil {
			return err
		}
	}

	if !hasCommits(dir) {
		fmt.Println("Sync configured. Add sheets with 'she --new <tool>', then run 'she --sync'.")
		return nil
	}

	fmt.Println("Pushing...")
	if err := streamGit(dir, "push", "-u", "origin", syncBranch); err != nil {
		return err
	}

	fmt.Println("Sync configured. Run 'she --sync' to sync from now on.")
	return nil
}

// syncRun performs the routine sync. The order is deliberate: commit local
// changes first so the working tree is clean, then rebase onto the freshly
// fetched remote so local work is replayed on top of everyone else's, then
// push the result. This keeps history linear and conflict-free as long as no
// two machines edit the same sheet between syncs.
func syncRun(dir string) error {
	if !isGitRepo(dir) {
		return errors.New("syncing is not set up — run 'she --sync <repo>' first")
	}
	if _, err := runGit(dir, "remote", "get-url", "origin"); err != nil {
		return errors.New("no sync remote configured — run 'she --sync <repo>' first")
	}

	committed, err := commitLocal(dir, "Sync from "+hostname())
	if err != nil {
		return err
	}
	if committed {
		fmt.Println("Committed local changes.")
	} else {
		fmt.Println("No local changes to commit.")
	}

	fmt.Println("Fetching remote sheets...")
	if err := streamGit(dir, "fetch", "origin"); err != nil {
		return err
	}

	if remoteBranchExists(dir, syncBranch) {
		// Guard against ~/.sheets having been re-pointed at a different
		// sheets repository: if both ends carry a marker and the ids differ,
		// refuse rather than entangle two unrelated histories.
		localID, localOK := localMarkerID(dir)
		remoteID, remoteOK, err := remoteMarkerID(dir)
		if err != nil {
			return err
		}
		if localOK && remoteOK && localID != remoteID {
			return fmt.Errorf("local %s is bound to sheets repository %s, but "+
				"origin now carries %s — refusing to sync mismatched repositories",
				dir, localID, remoteID)
		}

		if hasCommits(dir) {
			if err := rebaseOntoRemote(dir); err != nil {
				return err
			}
		} else if _, err := runGit(dir, "checkout", "-B", syncBranch, "origin/"+syncBranch); err != nil {
			return err
		}
	}

	if !hasCommits(dir) {
		fmt.Println("No sheets to sync yet.")
		return nil
	}

	fmt.Println("Pushing...")
	if err := streamGit(dir, "push", "-u", "origin", syncBranch); err != nil {
		return err
	}

	fmt.Println("Sheets synced.")
	return nil
}

// rebaseOntoRemote replays local commits on top of the fetched remote branch.
// On conflict it aborts the rebase — leaving the local commit intact and the
// working tree clean — and reports which sheets need manual resolution.
func rebaseOntoRemote(dir string) error {
	if _, err := runGit(dir, "rebase", "origin/"+syncBranch); err == nil {
		return nil
	}

	conflicts, _ := runGit(dir, "diff", "--name-only", "--diff-filter=U")
	if _, abortErr := runGit(dir, "rebase", "--abort"); abortErr != nil {
		return fmt.Errorf("rebase failed and could not be aborted; "+
			"resolve manually in %s: %w", dir, abortErr)
	}

	if conflicts == "" {
		return fmt.Errorf("could not rebase onto the remote; resolve manually in %s", dir)
	}
	list := "  " + strings.ReplaceAll(conflicts, "\n", "\n  ")
	return fmt.Errorf("the same sheet was edited on two machines:\n%s\n"+
		"your local changes are committed and safe — resolve with:\n"+
		"  cd %s && git pull --rebase", list, dir)
}

// commitLocal stages every change in dir and commits it with msg. It reports
// whether a commit was made; an unchanged working tree is not an error and
// produces no commit.
func commitLocal(dir, msg string) (bool, error) {
	if _, err := runGit(dir, "add", "-A"); err != nil {
		return false, err
	}
	// git diff --cached --quiet exits zero only when nothing is staged.
	if gitCmd(dir, "diff", "--cached", "--quiet").Run() == nil {
		return false, nil
	}
	if _, err := runGit(dir, "commit", "-m", msg); err != nil {
		return false, err
	}
	return true, nil
}

// newRepoID returns a fresh random repository identifier: 32 bytes of
// cryptographically random data, hex-encoded. It is an identity, not a hash
// of anything — git already provides content integrity.
func newRepoID() (string, error) {
	var buf [32]byte
	if _, err := rand.Read(buf[:]); err != nil {
		return "", fmt.Errorf("generate repository id: %w", err)
	}
	return hex.EncodeToString(buf[:]), nil
}

// markerBody returns the contents of a marker file for the given id.
func markerBody(id string) string {
	return fmt.Sprintf("%s\nversion: %d\nid: %s\n", markerSignature, markerVersion, id)
}

// writeMarker writes the marker file for id into dir.
func writeMarker(dir, id string) error {
	if err := os.WriteFile(filepath.Join(dir, markerName), []byte(markerBody(id)), 0o644); err != nil {
		return fmt.Errorf("write %s marker: %w", markerName, err)
	}
	return nil
}

// parseMarker extracts the repository id from marker file content. ok is true
// only when the content carries the she signature line and a well-formed id.
func parseMarker(content string) (id string, ok bool) {
	hasSignature := false
	for _, line := range strings.Split(content, "\n") {
		line = strings.TrimSpace(line)
		switch {
		case line == markerSignature:
			hasSignature = true
		case strings.HasPrefix(line, "id:"):
			id = strings.TrimSpace(strings.TrimPrefix(line, "id:"))
		}
	}
	if !hasSignature || !isRepoID(id) {
		return "", false
	}
	return id, true
}

// isRepoID reports whether s looks like a repository id: 64 hex characters.
func isRepoID(s string) bool {
	if len(s) != 64 {
		return false
	}
	_, err := hex.DecodeString(s)
	return err == nil
}

// ensureLocalMarker writes a marker file with a fresh id into dir if one is
// not already present. It is idempotent: an existing marker is left untouched
// so a repository keeps its identity across re-runs of setup.
func ensureLocalMarker(dir string) error {
	if _, present := localMarkerID(dir); present {
		return nil
	}
	id, err := newRepoID()
	if err != nil {
		return err
	}
	return writeMarker(dir, id)
}

// localMarkerID returns the repository id recorded in dir's working-tree
// marker file, and whether a valid marker was found.
func localMarkerID(dir string) (id string, present bool) {
	data, err := os.ReadFile(filepath.Join(dir, markerName))
	if err != nil {
		return "", false
	}
	return parseMarker(string(data))
}

// remoteMarkerID returns the repository id recorded in the marker file on the
// fetched remote branch. present is false (with a nil error) when the remote
// branch carries no marker file; a non-nil error indicates git itself failed.
func remoteMarkerID(dir string) (id string, present bool, err error) {
	out, gerr := runGit(dir, "show", "origin/"+syncBranch+":"+markerName)
	if gerr != nil {
		// git exits non-zero when the path does not exist on that branch;
		// treat that as "no marker" rather than a hard failure.
		return "", false, nil
	}
	id, ok := parseMarker(out)
	return id, ok, nil
}

// isGitRepo reports whether dir contains a git repository.
func isGitRepo(dir string) bool {
	info, err := os.Stat(filepath.Join(dir, ".git"))
	return err == nil && info.IsDir()
}

// hasCommits reports whether the repository in dir has at least one commit.
func hasCommits(dir string) bool {
	return gitCmd(dir, "rev-parse", "--verify", "--quiet", "HEAD").Run() == nil
}

// remoteBranchExists reports whether refs/remotes/origin/<branch> is present.
// It inspects the ref populated by the preceding fetch, so it makes no
// network call of its own.
func remoteBranchExists(dir, branch string) bool {
	return gitCmd(dir, "rev-parse", "--verify", "--quiet",
		"refs/remotes/origin/"+branch).Run() == nil
}

// hostname returns the machine's host name for use in sync commit messages,
// falling back to a placeholder when it cannot be determined.
func hostname() string {
	h, err := os.Hostname()
	if err != nil || h == "" {
		return "unknown host"
	}
	return h
}

// gitCmd builds a git command rooted in dir.
func gitCmd(dir string, args ...string) *exec.Cmd {
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	return cmd
}

// runGit runs a git command in dir and returns its trimmed standard output.
// On failure the returned error includes git's standard error.
func runGit(dir string, args ...string) (string, error) {
	var stdout, stderr bytes.Buffer
	cmd := gitCmd(dir, args...)
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		detail := strings.TrimSpace(stderr.String())
		if detail == "" {
			detail = strings.TrimSpace(stdout.String())
		}
		if detail != "" {
			return "", fmt.Errorf("git %s: %w: %s", strings.Join(args, " "), err, detail)
		}
		return "", fmt.Errorf("git %s: %w", strings.Join(args, " "), err)
	}
	return strings.TrimSpace(stdout.String()), nil
}

// streamGit runs a git command in dir with the standard streams attached, so
// progress is visible and credential prompts work. It is used for the network
// operations, fetch and push.
func streamGit(dir string, args ...string) error {
	cmd := gitCmd(dir, args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("git %s: %w", strings.Join(args, " "), err)
	}
	return nil
}
