package sheet

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

const gitTimeout = 30 * time.Second

// git runs a single git command scoped to the sheets directory, with its own
// 30-second deadline so a slow network call on one step doesn't eat into the
// next.
func git(args ...string) ([]byte, error) {
	dir, err := Dir()
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), gitTimeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, "git", append([]string{"-C", dir}, args...)...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return out, fmt.Errorf("git %s: %w\n%s", strings.Join(args, " "), err, strings.TrimSpace(string(out)))
	}
	return out, nil
}

func isGitRepo() bool {
	_, err := git("rev-parse", "--git-dir")
	return err == nil
}

func Sync() error {
	if !isGitRepo() {
		return nil
	}

	dir, err := Dir()
	if err != nil {
		return err
	}

	if _, err := git("pull", "--rebase"); err != nil {
		return fmt.Errorf("pull failed, resolve conflicts with: git -C %s rebase --continue\n%w", dir, err)
	}

	status, err := git("status", "--porcelain")
	if err != nil {
		return err
	}
	if len(bytes.TrimSpace(status)) == 0 {
		return nil
	}

	if _, err := git("add", "-A"); err != nil {
		return err
	}
	if _, err := git("commit", "-m", "auto sync"); err != nil {
		return err
	}
	if _, err := git("push"); err != nil {
		return fmt.Errorf("push failed, your changes are committed locally. Run 'she --sync' later to retry\n%w", err)
	}

	return nil
}

func SyncInit(remoteURL string) error {
	if isGitRepo() {
		return fmt.Errorf("~/.sheets is already a git repo")
	}

	if _, err := git("init"); err != nil {
		return err
	}
	if _, err := git("add", "-A"); err != nil {
		return err
	}
	// --allow-empty ensures the initial commit succeeds even when ~/.sheets
	// has no files yet (e.g. she --sync <url> before any she --edit calls).
	if _, err := git("commit", "--allow-empty", "-m", "initial"); err != nil {
		return err
	}
	if _, err := git("remote", "add", "origin", remoteURL); err != nil {
		return err
	}

	branch, err := git("rev-parse", "--abbrev-ref", "HEAD")
	if err != nil {
		return err
	}
	branchName := strings.TrimSpace(string(branch))

	if _, err := git("push", "-u", "origin", branchName); err != nil {
		return fmt.Errorf("push to %s failed: %w", remoteURL, err)
	}

	fmt.Printf("Sync initialized for remote: %s\n", remoteURL)
	return nil
}
