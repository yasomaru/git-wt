package git

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"
)

func run(dir string, args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	if dir != "" {
		cmd.Dir = dir
	}
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			msg = err.Error()
		}
		return "", fmt.Errorf("git %s: %s", strings.Join(args, " "), msg)
	}
	return strings.TrimSpace(stdout.String()), nil
}

func RepoRoot(dir string) (string, error) {
	return run(dir, "rev-parse", "--show-toplevel")
}

func DefaultBranch(dir string) (string, error) {
	// Try origin/HEAD first
	out, err := run(dir, "symbolic-ref", "refs/remotes/origin/HEAD")
	if err == nil {
		return strings.TrimPrefix(out, "refs/remotes/origin/"), nil
	}
	// Fallback: check for main or master
	for _, name := range []string{"main", "master"} {
		if _, err := run(dir, "rev-parse", "--verify", "refs/heads/"+name); err == nil {
			return name, nil
		}
	}
	return "main", nil
}

func BranchExists(dir, branch string) bool {
	_, err := run(dir, "rev-parse", "--verify", "refs/heads/"+branch)
	return err == nil
}

func IsInsideWorktree(dir string) bool {
	out, err := run(dir, "rev-parse", "--is-inside-work-tree")
	return err == nil && out == "true"
}
