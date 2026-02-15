package git

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/git-wt/git-wt/testutil"
)

func TestRepoRoot(t *testing.T) {
	t.Parallel()

	t.Run("valid repo returns absolute path", func(t *testing.T) {
		t.Parallel()
		dir := testutil.InitTestRepo(t)

		root, err := RepoRoot(dir)
		if err != nil {
			t.Fatalf("RepoRoot(%q) returned unexpected error: %v", dir, err)
		}

		// Resolve symlinks so the comparison works on macOS where
		// t.TempDir() may live under /var which symlinks to /private/var.
		wantResolved, err := filepath.EvalSymlinks(dir)
		if err != nil {
			t.Fatalf("EvalSymlinks(%q): %v", dir, err)
		}
		gotResolved, err := filepath.EvalSymlinks(root)
		if err != nil {
			t.Fatalf("EvalSymlinks(%q): %v", root, err)
		}

		if gotResolved != wantResolved {
			t.Errorf("RepoRoot(%q) = %q, want %q", dir, gotResolved, wantResolved)
		}
	})

	t.Run("subdirectory returns repo root", func(t *testing.T) {
		t.Parallel()
		dir := testutil.InitTestRepo(t)

		subdir := filepath.Join(dir, "sub", "nested")
		if err := os.MkdirAll(subdir, 0o755); err != nil {
			t.Fatalf("MkdirAll(%q): %v", subdir, err)
		}

		root, err := RepoRoot(subdir)
		if err != nil {
			t.Fatalf("RepoRoot(%q) returned unexpected error: %v", subdir, err)
		}

		wantResolved, _ := filepath.EvalSymlinks(dir)
		gotResolved, _ := filepath.EvalSymlinks(root)

		if gotResolved != wantResolved {
			t.Errorf("RepoRoot(%q) = %q, want %q", subdir, gotResolved, wantResolved)
		}
	})

	t.Run("non-repo directory returns error", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir() // plain directory, no git init

		_, err := RepoRoot(dir)
		if err == nil {
			t.Fatalf("RepoRoot(%q) expected error for non-repo directory, got nil", dir)
		}
	})
}

func TestDefaultBranch(t *testing.T) {
	t.Parallel()

	t.Run("detects existing master branch", func(t *testing.T) {
		t.Parallel()
		dir := testutil.InitTestRepo(t)

		// InitTestRepo uses the system default branch name.
		// Rename it to master explicitly so the test is deterministic.
		runGitHelper(t, dir, "branch", "-M", "master")

		got, err := DefaultBranch(dir)
		if err != nil {
			t.Fatalf("DefaultBranch(%q) returned unexpected error: %v", dir, err)
		}
		if got != "master" {
			t.Errorf("DefaultBranch(%q) = %q, want %q", dir, got, "master")
		}
	})

	t.Run("detects existing main branch", func(t *testing.T) {
		t.Parallel()
		dir := testutil.InitTestRepo(t)

		// Rename the default branch to main so the lookup finds it.
		runGitHelper(t, dir, "branch", "-M", "main")

		got, err := DefaultBranch(dir)
		if err != nil {
			t.Fatalf("DefaultBranch(%q) returned unexpected error: %v", dir, err)
		}
		if got != "main" {
			t.Errorf("DefaultBranch(%q) = %q, want %q", dir, got, "main")
		}
	})

	t.Run("falls back to main when neither main nor master exists", func(t *testing.T) {
		t.Parallel()
		dir := testutil.InitTestRepo(t)

		// Rename the default branch to something else entirely.
		runGitHelper(t, dir, "branch", "-M", "develop")

		got, err := DefaultBranch(dir)
		if err != nil {
			t.Fatalf("DefaultBranch(%q) returned unexpected error: %v", dir, err)
		}
		if got != "main" {
			t.Errorf("DefaultBranch(%q) = %q, want fallback %q", dir, got, "main")
		}
	})

	t.Run("prefers main over master when both exist", func(t *testing.T) {
		t.Parallel()
		dir := testutil.InitTestRepo(t)

		// Ensure the current branch is main, then create master as a second branch.
		runGitHelper(t, dir, "branch", "-M", "main")
		testutil.CreateBranch(t, dir, "master")

		got, err := DefaultBranch(dir)
		if err != nil {
			t.Fatalf("DefaultBranch(%q) returned unexpected error: %v", dir, err)
		}
		// The fallback loop checks "main" before "master", so main wins.
		if got != "main" {
			t.Errorf("DefaultBranch(%q) = %q, want %q (main preferred over master)", dir, got, "main")
		}
	})
}

func TestBranchExists(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		setupFunc  func(t *testing.T, dir string)
		branch     string
		wantExists bool
	}{
		{
			name:       "existing branch returns true",
			setupFunc:  func(t *testing.T, dir string) { testutil.CreateBranch(t, dir, "feature-x") },
			branch:     "feature-x",
			wantExists: true,
		},
		{
			name:       "non-existing branch returns false",
			setupFunc:  nil,
			branch:     "no-such-branch",
			wantExists: false,
		},
		{
			name:       "default branch exists",
			setupFunc:  nil,
			branch:     "master",
			wantExists: true, // InitTestRepo creates master on this system
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			dir := testutil.InitTestRepo(t)

			// Ensure the default branch is named master for deterministic results.
			runGitHelper(t, dir, "branch", "-M", "master")

			if tc.setupFunc != nil {
				tc.setupFunc(t, dir)
			}

			got := BranchExists(dir, tc.branch)
			if got != tc.wantExists {
				t.Errorf("BranchExists(%q, %q) = %v, want %v", dir, tc.branch, got, tc.wantExists)
			}
		})
	}
}

func TestIsInsideWorktree(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		dir  func(t *testing.T) string
		want bool
	}{
		{
			name: "inside git repo returns true",
			dir: func(t *testing.T) string {
				return testutil.InitTestRepo(t)
			},
			want: true,
		},
		{
			name: "inside linked worktree returns true",
			dir: func(t *testing.T) string {
				mainDir := testutil.InitTestRepo(t)
				return testutil.AddWorktree(t, mainDir, "wt-branch")
			},
			want: true,
		},
		{
			name: "plain directory returns false",
			dir: func(t *testing.T) string {
				return t.TempDir()
			},
			want: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			dir := tc.dir(t)

			got := IsInsideWorktree(dir)
			if got != tc.want {
				t.Errorf("IsInsideWorktree(%q) = %v, want %v", dir, got, tc.want)
			}
		})
	}
}

// runGitHelper executes a git command in the given directory, failing the test on error.
func runGitHelper(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v failed: %s\n%s", args, err, string(out))
	}
}
