package sheet

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
)

// syncBranch is the branch used for the central sheets repository.
const syncBranch = "main"

const (
	markerName = ".shetag"
	// markerVersion is written into new markers; parseMarker does not enforce it yet.
	markerVersion = 1
)

// credentialURL matches a URL's userinfo, which may carry a token or password.
var credentialURL = regexp.MustCompile(`([a-zA-Z][a-zA-Z0-9+.-]*://)[^/@\s]+@`)

// Sync synchronises ~/.sheets with a central git repository. Pass a non-empty
// repo for first-time setup; pass "" for a routine commit-fetch-rebase-push
// cycle. Status messages are written to stderr; child git processes still
// inherit the parent's stderr for interactive prompts and progress output.
func Sync(stderr io.Writer, repo string) error {
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
		return syncSetup(stderr, dir, repo)
	}
	return syncRun(stderr, dir)
}

// syncSetup wires ~/.sheets up to repo and performs the first exchange. A remote
// that already has history must carry a marker, else it may be an unrelated
// project; rejection rolls back the half-wired repo.
func syncSetup(stderr io.Writer, dir, repo string) error {
	createdGit := false
	previousURL := ""
	// displayRepo masks any URL credentials; use it for all user-facing output.
	displayRepo := redactURL(repo)

	if isGitRepo(dir) {
		if url, err := runGit(dir, "remote", "get-url", "origin"); err == nil {
			previousURL = url
			if _, err := runGit(dir, "remote", "set-url", "origin", repo); err != nil {
				return err
			}
		} else if _, err := runGit(dir, "remote", "add", "origin", repo); err != nil {
			return err
		}
		fmt.Fprintf(stderr, "Updated sync remote to %s\n", displayRepo)
	} else {
		if _, err := runGit(dir, "init", "-b", syncBranch); err != nil {
			return err
		}
		createdGit = true
		if _, err := runGit(dir, "remote", "add", "origin", repo); err != nil {
			return err
		}
		fmt.Fprintf(stderr, "Initialized sync repository in %s\n", dir)
	}

	// rollback undoes the changes above on rejection or failure.
	rollback := func() {
		if createdGit {
			_ = os.RemoveAll(filepath.Join(dir, ".git"))
			return
		}
		if previousURL != "" {
			_, _ = runGit(dir, "remote", "set-url", "origin", previousURL)
		}
	}

	fmt.Fprintln(stderr, "Fetching remote sheets...")
	if err := streamGit(dir, "fetch", "origin"); err != nil {
		rollback()
		return fmt.Errorf("could not reach the remote %s — check the "+
			"repository exists and you have access; no changes were made to %s: %w",
			displayRepo, dir, err)
	}

	if remoteBranchExists(dir, syncBranch) {
		// Remote has history: established sheets repo if marker present, else refuse.
		_, present, err := remoteMarkerID(dir)
		if err != nil {
			rollback()
			return err
		}
		if !present {
			rollback()
			fmt.Fprintln(stderr, "Hint: 'she --sync' only initializes an empty repository.")
			fmt.Fprintln(stderr, "      Create the repo with no README, license, or .gitignore, then retry.")
			return fmt.Errorf("the remote %s has commits but no %s marker; no changes were made to %s",
				displayRepo, markerName, dir)
		}
	} else {
		// Empty remote: establish it with a fresh marker.
		if err := ensureLocalMarker(dir); err != nil {
			rollback()
			return err
		}
	}

	if _, err := commitLocal(dir, "Add sheets from "+hostname()); err != nil {
		rollback()
		return err
	}

	switch {
	case remoteBranchExists(dir, syncBranch) && hasCommits(dir):
		// Both ends have history: stitch them together.
		if _, err := runGit(dir, "merge", "origin/"+syncBranch,
			"--allow-unrelated-histories", "-m", "Merge remote sheets"); err != nil {
			conflicts, _ := runGit(dir, "diff", "--name-only", "--diff-filter=U")
			if slices.Contains(strings.Split(conflicts, "\n"), markerName) {
				// Marker conflict: different repo identities — nothing to resolve by hand.
				_, _ = runGit(dir, "merge", "--abort")
				rollback()
				return fmt.Errorf("%s and %s are different sheets repositories "+
					"(marker id mismatch) — point --sync at the correct repository",
					dir, displayRepo)
			}
			// Sheet conflict: leave the merge for the user to resolve.
			return fmt.Errorf("the same sheet differs between %s and the remote — "+
				"resolve the conflict there, then run 'she --sync': %w", dir, err)
		}
	case remoteBranchExists(dir, syncBranch):
		// No local history yet: adopt the remote branch as-is.
		if _, err := runGit(dir, "checkout", "-B", syncBranch, "origin/"+syncBranch); err != nil {
			rollback()
			return err
		}
	}

	if !hasCommits(dir) {
		fmt.Fprintln(stderr, "Sync configured. Add sheets with 'she --new <tool>', then run 'she --sync'.")
		return nil
	}

	fmt.Fprintln(stderr, "Pushing...")
	if err := streamGit(dir, "push", "-u", "origin", syncBranch); err != nil {
		// Not rolled back: the local repo is valid; re-run 'she --sync' to retry the push.
		return err
	}

	fmt.Fprintln(stderr, "Sync configured. Run 'she --sync' to sync from now on.")
	return nil
}

// syncRun is the routine sync: commit-fetch-rebase-push, in that order, so local
// work replays cleanly on top of remote work.
func syncRun(stderr io.Writer, dir string) error {
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
		fmt.Fprintln(stderr, "Committed local changes.")
	} else {
		fmt.Fprintln(stderr, "No local changes to commit.")
	}

	fmt.Fprintln(stderr, "Fetching remote sheets...")
	if err := streamGit(dir, "fetch", "origin"); err != nil {
		return err
	}

	if remoteBranchExists(dir, syncBranch) {
		// Refuse if local and remote marker ids differ — re-pointed at a different repo.
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
		fmt.Fprintln(stderr, "No sheets to sync yet.")
		return nil
	}

	fmt.Fprintln(stderr, "Pushing...")
	if err := streamGit(dir, "push", "-u", "origin", syncBranch); err != nil {
		return err
	}

	fmt.Fprintln(stderr, "Sheets synced.")
	return nil
}

// rebaseOntoRemote replays local commits onto the fetched remote branch. On
// conflict it aborts cleanly and reports which sheets need manual resolution.
func rebaseOntoRemote(dir string) error {
	_, rebaseErr := runGit(dir, "rebase", "origin/"+syncBranch)
	if rebaseErr == nil {
		return nil
	}

	conflicts, _ := runGit(dir, "diff", "--name-only", "--diff-filter=U")
	if _, abortErr := runGit(dir, "rebase", "--abort"); abortErr != nil {
		return fmt.Errorf("rebase failed (%v) and could not be aborted; "+
			"resolve manually in %s: %w", rebaseErr, dir, abortErr)
	}

	if conflicts == "" {
		return fmt.Errorf("could not rebase onto the remote; resolve manually in %s: %w", dir, rebaseErr)
	}
	list := "  " + strings.ReplaceAll(conflicts, "\n", "\n  ")
	return fmt.Errorf("the same sheet was edited on two machines:\n%s\n"+
		"your local changes are committed and safe — resolve with:\n"+
		"  cd %s && git pull --rebase", list, dir)
}

// commitLocal stages every change in dir and commits it. Reports whether a
// commit was made; an unchanged tree is not an error.
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

// newRepoID returns a fresh 32-byte random repository identifier, hex-encoded.
// It is an identity, not a hash — git already provides content integrity.
func newRepoID() (string, error) {
	var buf [32]byte
	if _, err := rand.Read(buf[:]); err != nil {
		return "", fmt.Errorf("generate repository id: %w", err)
	}
	return hex.EncodeToString(buf[:]), nil
}

// markerBody returns the contents of a marker file for the given id.
func markerBody(id string) string {
	return fmt.Sprintf("version: %d\nid: %s\n", markerVersion, id)
}

// writeMarker writes the marker file for id into dir.
func writeMarker(dir, id string) error {
	if err := os.WriteFile(filepath.Join(dir, markerName), []byte(markerBody(id)), 0o644); err != nil {
		return fmt.Errorf("write %s marker: %w", markerName, err)
	}
	return nil
}

// parseMarker extracts the id from marker content; ok requires a well-formed
// id. The .shetag filename is the only "is this ours?" check — the content
// has no self-signature.
func parseMarker(content string) (id string, ok bool) {
	for _, line := range strings.Split(content, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "id:") {
			id = strings.TrimSpace(strings.TrimPrefix(line, "id:"))
		}
	}
	if !isRepoID(id) {
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

// ensureLocalMarker writes a fresh marker into dir if one is not already
// present. Idempotent: an existing marker keeps its identity.
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

// localMarkerID returns the repository id from dir's marker file, if valid.
func localMarkerID(dir string) (id string, present bool) {
	data, err := os.ReadFile(filepath.Join(dir, markerName))
	if err != nil {
		return "", false
	}
	return parseMarker(string(data))
}

// remoteMarkerID returns the marker id on origin/<syncBranch>. present is false
// when the marker is absent; err is non-nil only on a real git failure.
func remoteMarkerID(dir string) (id string, present bool, err error) {
	// git ls-tree exits zero with empty output when the path is absent; non-zero is a real failure.
	listed, err := runGit(dir, "ls-tree", "origin/"+syncBranch, "--", markerName)
	if err != nil {
		return "", false, err
	}
	if listed == "" {
		return "", false, nil
	}
	out, err := runGit(dir, "show", "origin/"+syncBranch+":"+markerName)
	if err != nil {
		return "", false, err
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

// remoteBranchExists checks the local refs/remotes/origin/<branch> populated
// by the preceding fetch — no network call.
func remoteBranchExists(dir, branch string) bool {
	return gitCmd(dir, "rev-parse", "--verify", "--quiet",
		"refs/remotes/origin/"+branch).Run() == nil
}

// hostname returns the machine's host name for commit messages, with a fallback.
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

// runGit runs git in dir and returns its trimmed stdout; on failure the error
// includes git's stderr.
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
			return "", fmt.Errorf("git %s: %w: %s", redactArgs(args), err, detail)
		}
		return "", fmt.Errorf("git %s: %w", redactArgs(args), err)
	}
	return strings.TrimSpace(stdout.String()), nil
}

// streamGit runs git for the network ops (fetch/push): output goes to stderr
// to keep stdout clean; stdin stays attached for credential prompts.
func streamGit(dir string, args ...string) error {
	cmd := gitCmd(dir, args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stderr
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("git %s: %w", redactArgs(args), err)
	}
	return nil
}

// redactURL masks credentials embedded in a URL.
func redactURL(url string) string {
	return credentialURL.ReplaceAllString(url, "${1}***@")
}

// redactArgs joins git args for display, with credentials masked.
func redactArgs(args []string) string {
	return redactURL(strings.Join(args, " "))
}
