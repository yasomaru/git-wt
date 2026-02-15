package testutil

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// InitTestRepo creates a temporary git repository with an initial commit.
// Returns the path to the repo directory.
func InitTestRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()

	runGit(t, dir, "init")
	runGit(t, dir, "config", "user.email", "test@example.com")
	runGit(t, dir, "config", "user.name", "Test")

	WriteFile(t, dir, "README.md", "# test\n")
	runGit(t, dir, "add", ".")
	runGit(t, dir, "commit", "-m", "initial commit")

	return dir
}

// CreateBranch creates a new branch in the given repo directory.
func CreateBranch(t *testing.T, dir, branch string) {
	t.Helper()
	runGit(t, dir, "branch", branch)
}

// AddWorktree creates a new worktree for the given branch and returns its path.
func AddWorktree(t *testing.T, dir, branch string) string {
	t.Helper()
	wtPath := filepath.Join(filepath.Dir(dir), filepath.Base(dir)+"-"+branch)
	runGit(t, dir, "worktree", "add", "-b", branch, wtPath)
	t.Cleanup(func() {
		// Best-effort cleanup
		_ = exec.Command("git", "-C", dir, "worktree", "remove", "--force", wtPath).Run()
	})
	return wtPath
}

// MakeCommit creates a new commit with a dummy file change.
func MakeCommit(t *testing.T, dir, message string) {
	t.Helper()
	// Create a unique file to avoid conflicts
	name := "commit-" + message + ".txt"
	WriteFile(t, dir, name, message+"\n")
	runGit(t, dir, "add", ".")
	runGit(t, dir, "commit", "-m", message)
}

// WriteFile creates or overwrites a file in the given directory.
func WriteFile(t *testing.T, dir, name, content string) {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func runGit(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v failed: %s\n%s", args, err, string(out))
	}
	return string(out)
}
