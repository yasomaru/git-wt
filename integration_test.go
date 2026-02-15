//go:build integration

package main

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/yasomaru/git-wt/testutil"
)

var binPath string

func TestMain(m *testing.M) {
	tmp, err := os.MkdirTemp("", "git-wt-integration-*")
	if err != nil {
		panic("failed to create temp dir: " + err.Error())
	}
	defer os.RemoveAll(tmp)

	binName := "git-wt"
	if runtime.GOOS == "windows" {
		binName = "git-wt.exe"
	}
	binPath = filepath.Join(tmp, binName)
	cmd := exec.Command("go", "build", "-o", binPath, ".")
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		panic("failed to build binary: " + err.Error())
	}

	os.Exit(m.Run())
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func runBinary(t *testing.T, bin, dir string, args ...string) (stdout, stderr string, err error) {
	t.Helper()
	return runBinaryInput(t, bin, dir, "", args...)
}

func runBinaryInput(t *testing.T, bin, dir, input string, args ...string) (stdout, stderr string, err error) {
	t.Helper()
	cmd := exec.Command(bin, args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(),
		"GIT_CONFIG_GLOBAL="+devNull(),
		"GIT_CONFIG_SYSTEM="+devNull(),
		"NO_COLOR=1",
	)
	if input != "" {
		cmd.Stdin = strings.NewReader(input)
	}
	var outBuf, errBuf bytes.Buffer
	cmd.Stdout = &outBuf
	cmd.Stderr = &errBuf
	err = cmd.Run()
	return outBuf.String(), errBuf.String(), err
}

// devNull returns the null device path for the current OS.
func devNull() string {
	if runtime.GOOS == "windows" {
		return "NUL"
	}
	return "/dev/null"
}

// evalDir resolves macOS /tmp symlinks so paths can be compared reliably.
func evalDir(t *testing.T, dir string) string {
	t.Helper()
	resolved, err := filepath.EvalSymlinks(dir)
	if err != nil {
		t.Fatalf("EvalSymlinks(%s): %v", dir, err)
	}
	return resolved
}

// gitRun is a small helper that runs a git command in a directory.
func gitRun(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(),
		"GIT_CONFIG_GLOBAL="+devNull(),
		"GIT_CONFIG_SYSTEM="+devNull(),
	)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v failed: %s\n%s", args, err, string(out))
	}
	return string(out)
}

// writeLocalConfig writes a .git-wt.toml in the repo directory.
func writeLocalConfig(t *testing.T, dir, content string) {
	t.Helper()
	testutil.WriteFile(t, dir, ".git-wt.toml", content)
}

// branchExists checks if a branch exists in the repo.
func branchExists(t *testing.T, dir, branch string) bool {
	t.Helper()
	cmd := exec.Command("git", "rev-parse", "--verify", "refs/heads/"+branch)
	cmd.Dir = dir
	return cmd.Run() == nil
}

// ===========================================================================
// ADD COMMAND TESTS
// ===========================================================================

func TestAdd_NewBranch(t *testing.T) {
	repo := evalDir(t, testutil.InitTestRepo(t))

	stdout, stderr, err := runBinary(t, binPath, repo, "add", "feature-new")
	if err != nil {
		t.Fatalf("add failed: %v\nstdout: %s\nstderr: %s", err, stdout, stderr)
	}

	// Verify the branch was created in git.
	if !branchExists(t, repo, "feature-new") {
		t.Error("expected branch 'feature-new' to exist after add")
	}

	// Verify the worktree directory was created (adjacent layout by default).
	repoName := filepath.Base(repo)
	expectedPath := filepath.Join(filepath.Dir(repo), repoName+"-feature-new")
	if _, err := os.Stat(expectedPath); os.IsNotExist(err) {
		t.Errorf("expected worktree directory to exist at %s", expectedPath)
	}

	// Output should mention the branch.
	if !strings.Contains(stdout, "feature-new") {
		t.Errorf("expected stdout to mention branch name, got: %s", stdout)
	}
}

func TestAdd_ExistingBranch(t *testing.T) {
	repo := evalDir(t, testutil.InitTestRepo(t))

	// Create a branch first via git (without a worktree).
	testutil.CreateBranch(t, repo, "existing-branch")

	stdout, stderr, err := runBinary(t, binPath, repo, "add", "existing-branch")
	if err != nil {
		t.Fatalf("add existing branch failed: %v\nstdout: %s\nstderr: %s", err, stdout, stderr)
	}

	// Verify worktree directory exists.
	repoName := filepath.Base(repo)
	expectedPath := filepath.Join(filepath.Dir(repo), repoName+"-existing-branch")
	if _, err := os.Stat(expectedPath); os.IsNotExist(err) {
		t.Errorf("expected worktree directory at %s", expectedPath)
	}
}

func TestAdd_BaseFlag(t *testing.T) {
	repo := evalDir(t, testutil.InitTestRepo(t))

	// Create a "develop" branch with an extra commit.
	testutil.CreateBranch(t, repo, "develop")
	gitRun(t, repo, "checkout", "develop")
	testutil.MakeCommit(t, repo, "develop-commit")
	developHead := strings.TrimSpace(gitRun(t, repo, "rev-parse", "HEAD"))
	gitRun(t, repo, "checkout", "master")

	stdout, stderr, err := runBinary(t, binPath, repo, "add", "feature-from-develop", "-b", "develop")
	if err != nil {
		t.Fatalf("add --base failed: %v\nstdout: %s\nstderr: %s", err, stdout, stderr)
	}

	// The new branch should be based on develop, so its HEAD should match.
	repoName := filepath.Base(repo)
	wtPath := filepath.Join(filepath.Dir(repo), repoName+"-feature-from-develop")
	wtHead := strings.TrimSpace(gitRun(t, wtPath, "rev-parse", "HEAD"))

	if wtHead != developHead {
		t.Errorf("expected worktree HEAD to match develop (%s), got %s", developHead, wtHead)
	}
}

func TestAdd_PathAlreadyExists(t *testing.T) {
	repo := evalDir(t, testutil.InitTestRepo(t))

	// Pre-create the target directory so it already exists.
	repoName := filepath.Base(repo)
	targetPath := filepath.Join(filepath.Dir(repo), repoName+"-conflict")
	if err := os.MkdirAll(targetPath, 0o755); err != nil {
		t.Fatalf("failed to create conflict dir: %v", err)
	}

	_, stderr, err := runBinary(t, binPath, repo, "add", "conflict")
	if err == nil {
		t.Fatal("expected error when path already exists, got nil")
	}
	if !strings.Contains(stderr, "path already exists") {
		t.Errorf("expected 'path already exists' in stderr, got: %s", stderr)
	}
}

func TestAdd_AdjacentLayout(t *testing.T) {
	repo := evalDir(t, testutil.InitTestRepo(t))

	stdout, stderr, err := runBinary(t, binPath, repo, "add", "adjacent-test")
	if err != nil {
		t.Fatalf("add failed: %v\nstdout: %s\nstderr: %s", err, stdout, stderr)
	}

	// Default layout is "adjacent": ../repo-branch/
	repoName := filepath.Base(repo)
	expectedPath := filepath.Join(filepath.Dir(repo), repoName+"-adjacent-test")
	if _, err := os.Stat(expectedPath); os.IsNotExist(err) {
		t.Errorf("expected adjacent worktree at %s", expectedPath)
	}

	// Verify it is NOT inside the repo.
	if strings.HasPrefix(expectedPath, repo+string(os.PathSeparator)) {
		t.Errorf("adjacent layout should NOT be inside repo dir")
	}
}

func TestAdd_SubdirectoryLayout(t *testing.T) {
	repo := evalDir(t, testutil.InitTestRepo(t))

	// Write local config to use subdirectory layout.
	writeLocalConfig(t, repo, `
[layout]
strategy = "subdirectory"
`)

	stdout, stderr, err := runBinary(t, binPath, repo, "add", "subdir-test")
	if err != nil {
		t.Fatalf("add with subdirectory layout failed: %v\nstdout: %s\nstderr: %s", err, stdout, stderr)
	}

	// Subdirectory layout: .worktrees/branch/
	expectedPath := filepath.Join(repo, ".worktrees", "subdir-test")
	if _, err := os.Stat(expectedPath); os.IsNotExist(err) {
		t.Errorf("expected subdirectory worktree at %s", expectedPath)
	}
}

func TestAdd_PostAddHook(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("hook test uses sh -c which is not available on Windows")
	}

	repo := evalDir(t, testutil.InitTestRepo(t))

	// Configure a post_add hook that creates a marker file.
	writeLocalConfig(t, repo, `
[hooks]
post_add = "touch hook-ran.txt"
`)

	stdout, stderr, err := runBinary(t, binPath, repo, "add", "hook-test")
	if err != nil {
		t.Fatalf("add with hook failed: %v\nstdout: %s\nstderr: %s", err, stdout, stderr)
	}

	// The hook runs inside the new worktree directory.
	repoName := filepath.Base(repo)
	wtPath := filepath.Join(filepath.Dir(repo), repoName+"-hook-test")
	markerPath := filepath.Join(wtPath, "hook-ran.txt")
	if _, err := os.Stat(markerPath); os.IsNotExist(err) {
		t.Errorf("expected hook marker file at %s", markerPath)
	}

	// Output should indicate the hook was run.
	if !strings.Contains(stdout, "Running") || !strings.Contains(stdout, "touch hook-ran.txt") {
		t.Errorf("expected stdout to mention hook execution, got: %s", stdout)
	}
}

func TestAdd_OutsideGitRepo(t *testing.T) {
	// Use a plain temp dir that is not a git repo.
	dir := t.TempDir()

	_, stderr, err := runBinary(t, binPath, dir, "add", "should-fail")
	if err == nil {
		t.Fatal("expected error when running add outside git repo, got nil")
	}
	if !strings.Contains(stderr, "not a git repository") {
		t.Errorf("expected 'not a git repository' in stderr, got: %s", stderr)
	}
}

// ===========================================================================
// LS COMMAND TESTS
// ===========================================================================

func TestLs_SingleWorktree(t *testing.T) {
	repo := evalDir(t, testutil.InitTestRepo(t))

	stdout, stderr, err := runBinary(t, binPath, repo, "ls")
	if err != nil {
		t.Fatalf("ls failed: %v\nstdout: %s\nstderr: %s", err, stdout, stderr)
	}

	// Should list the main worktree (master branch).
	if !strings.Contains(stdout, "master") {
		t.Errorf("expected 'master' in ls output, got: %s", stdout)
	}
}

func TestLs_MultipleWorktrees(t *testing.T) {
	repo := evalDir(t, testutil.InitTestRepo(t))

	// Add two worktrees.
	testutil.AddWorktree(t, repo, "feature-a")
	testutil.AddWorktree(t, repo, "feature-b")

	stdout, stderr, err := runBinary(t, binPath, repo, "ls")
	if err != nil {
		t.Fatalf("ls failed: %v\nstdout: %s\nstderr: %s", err, stdout, stderr)
	}

	if !strings.Contains(stdout, "master") {
		t.Errorf("expected 'master' in output, got: %s", stdout)
	}
	if !strings.Contains(stdout, "feature-a") {
		t.Errorf("expected 'feature-a' in output, got: %s", stdout)
	}
	if !strings.Contains(stdout, "feature-b") {
		t.Errorf("expected 'feature-b' in output, got: %s", stdout)
	}
}

func TestLs_DirtyWorktree(t *testing.T) {
	repo := evalDir(t, testutil.InitTestRepo(t))

	wtPath := testutil.AddWorktree(t, repo, "dirty-branch")

	// Create an untracked file to make the worktree dirty.
	testutil.WriteFile(t, wtPath, "untracked.txt", "dirty content\n")

	stdout, stderr, err := runBinary(t, binPath, repo, "ls")
	if err != nil {
		t.Fatalf("ls failed: %v\nstdout: %s\nstderr: %s", err, stdout, stderr)
	}

	// The output should show the dirty branch with some modification indicator.
	if !strings.Contains(stdout, "dirty-branch") {
		t.Errorf("expected 'dirty-branch' in output, got: %s", stdout)
	}
	// Should show untracked file count or modified status, not "clean" for that row.
	if !strings.Contains(stdout, "untracked") {
		t.Errorf("expected 'untracked' status indicator in output, got: %s", stdout)
	}
}

func TestLs_MergedBranch(t *testing.T) {
	repo := evalDir(t, testutil.InitTestRepo(t))

	// Create a branch, add commit, then merge into master.
	wtPath := testutil.AddWorktree(t, repo, "merged-branch")
	testutil.MakeCommit(t, wtPath, "feature work")

	gitRun(t, repo, "merge", "merged-branch")

	stdout, stderr, err := runBinary(t, binPath, repo, "ls")
	if err != nil {
		t.Fatalf("ls failed: %v\nstdout: %s\nstderr: %s", err, stdout, stderr)
	}

	if !strings.Contains(stdout, "merged-branch") {
		t.Errorf("expected 'merged-branch' in output, got: %s", stdout)
	}
	if !strings.Contains(stdout, "merged") {
		t.Errorf("expected 'merged' indicator in output, got: %s", stdout)
	}
}

func TestLs_DetachedHead(t *testing.T) {
	repo := evalDir(t, testutil.InitTestRepo(t))

	// Get a commit hash to create a detached worktree.
	head := strings.TrimSpace(gitRun(t, repo, "rev-parse", "HEAD"))

	// Create a worktree at a detached HEAD.
	wtPath := filepath.Join(filepath.Dir(repo), filepath.Base(repo)+"-detached")
	gitRun(t, repo, "worktree", "add", "--detach", wtPath, head)
	t.Cleanup(func() {
		exec.Command("git", "-C", repo, "worktree", "remove", "--force", wtPath).Run()
	})

	stdout, stderr, err := runBinary(t, binPath, repo, "ls")
	if err != nil {
		t.Fatalf("ls failed: %v\nstdout: %s\nstderr: %s", err, stdout, stderr)
	}

	// Should show detached indicator.
	if !strings.Contains(stdout, "detached") {
		t.Errorf("expected 'detached' in output, got: %s", stdout)
	}
}

// ===========================================================================
// CLEAN COMMAND TESTS
// ===========================================================================

func TestClean_MergedFlag(t *testing.T) {
	repo := evalDir(t, testutil.InitTestRepo(t))

	// Create and merge a branch.
	wtPath := testutil.AddWorktree(t, repo, "to-clean")
	testutil.MakeCommit(t, wtPath, "feature to merge")
	gitRun(t, repo, "merge", "to-clean")

	stdout, stderr, err := runBinary(t, binPath, repo, "clean", "--merged", "--force")
	if err != nil {
		t.Fatalf("clean --merged --force failed: %v\nstdout: %s\nstderr: %s", err, stdout, stderr)
	}

	// The merged worktree should be removed.
	if _, statErr := os.Stat(wtPath); !os.IsNotExist(statErr) {
		t.Errorf("expected merged worktree to be removed at %s", wtPath)
	}

	if !strings.Contains(stdout, "Removed") || !strings.Contains(stdout, "to-clean") {
		t.Errorf("expected removal confirmation in output, got: %s", stdout)
	}
}

func TestClean_StaleFlagSkipsFresh(t *testing.T) {
	repo := evalDir(t, testutil.InitTestRepo(t))

	// Create a fresh worktree (just created, so 0 days inactive).
	wtPath := testutil.AddWorktree(t, repo, "fresh-branch")
	testutil.MakeCommit(t, wtPath, "recent work")

	stdout, _, err := runBinary(t, binPath, repo, "clean", "--stale", "30", "--force")
	if err != nil {
		t.Fatalf("clean --stale failed: %v", err)
	}

	// The fresh worktree should NOT be removed.
	if _, statErr := os.Stat(wtPath); os.IsNotExist(statErr) {
		t.Error("expected fresh worktree to be kept, but it was removed")
	}

	// Output should indicate nothing to clean.
	if !strings.Contains(stdout, "No worktrees to clean up") {
		t.Errorf("expected 'No worktrees to clean up' in output, got: %s", stdout)
	}
}

func TestClean_DryRun(t *testing.T) {
	repo := evalDir(t, testutil.InitTestRepo(t))

	// Create and merge a branch.
	wtPath := testutil.AddWorktree(t, repo, "dry-run-branch")
	testutil.MakeCommit(t, wtPath, "feature")
	gitRun(t, repo, "merge", "dry-run-branch")

	stdout, _, err := runBinary(t, binPath, repo, "clean", "--merged", "--dry-run")
	if err != nil {
		t.Fatalf("clean --dry-run failed: %v", err)
	}

	// The worktree should still exist.
	if _, statErr := os.Stat(wtPath); os.IsNotExist(statErr) {
		t.Error("expected dry-run to preserve worktree, but it was removed")
	}

	// Output should indicate dry run.
	if !strings.Contains(stdout, "Dry run") {
		t.Errorf("expected 'Dry run' in output, got: %s", stdout)
	}

	// Output should list the candidate.
	if !strings.Contains(stdout, "dry-run-branch") {
		t.Errorf("expected candidate 'dry-run-branch' in dry-run output, got: %s", stdout)
	}
}

func TestClean_ForceSkipsConfirmation(t *testing.T) {
	repo := evalDir(t, testutil.InitTestRepo(t))

	// Create and merge a branch.
	wtPath := testutil.AddWorktree(t, repo, "force-clean")
	testutil.MakeCommit(t, wtPath, "feature")
	gitRun(t, repo, "merge", "force-clean")

	// With --force, no stdin input should be required.
	stdout, _, err := runBinary(t, binPath, repo, "clean", "--merged", "--force")
	if err != nil {
		t.Fatalf("clean --force failed: %v", err)
	}

	if _, statErr := os.Stat(wtPath); !os.IsNotExist(statErr) {
		t.Errorf("expected force-clean worktree to be removed")
	}

	if !strings.Contains(stdout, "Cleaned up") {
		t.Errorf("expected 'Cleaned up' in output, got: %s", stdout)
	}
}

func TestClean_SkipsCurrentWorktree(t *testing.T) {
	repo := evalDir(t, testutil.InitTestRepo(t))

	// Create a worktree and merge it, then run clean FROM that worktree.
	wtPath := evalDir(t, testutil.AddWorktree(t, repo, "current-wt"))
	testutil.MakeCommit(t, wtPath, "feature")
	gitRun(t, repo, "merge", "current-wt")

	// Run clean from inside the worktree itself.
	stdout, _, err := runBinary(t, binPath, wtPath, "clean", "--merged", "--force")
	if err != nil {
		t.Fatalf("clean from worktree failed: %v", err)
	}

	// The worktree we are inside should be skipped (IsCurrent = true).
	if _, statErr := os.Stat(wtPath); os.IsNotExist(statErr) {
		t.Error("expected current worktree to be skipped, but it was removed")
	}

	// Output should indicate nothing to clean (only candidate was current).
	if !strings.Contains(stdout, "No worktrees to clean up") {
		// It could also just not list this worktree - either way it should not be removed.
		t.Logf("stdout: %s", stdout)
	}
}

func TestClean_SkipsDefaultBranch(t *testing.T) {
	repo := evalDir(t, testutil.InitTestRepo(t))

	// Add a non-default worktree and merge it.
	wtPath := testutil.AddWorktree(t, repo, "to-remove")
	testutil.MakeCommit(t, wtPath, "feature")
	gitRun(t, repo, "merge", "to-remove")

	stdout, _, err := runBinary(t, binPath, repo, "clean", "--merged", "--force")
	if err != nil {
		t.Fatalf("clean failed: %v", err)
	}

	// Master (default branch) worktree should still exist.
	if _, statErr := os.Stat(repo); os.IsNotExist(statErr) {
		t.Error("expected default branch (master) worktree to be preserved")
	}

	// The non-default merged branch should be removed.
	if _, statErr := os.Stat(wtPath); !os.IsNotExist(statErr) {
		t.Errorf("expected merged non-default worktree to be removed at %s", wtPath)
	}

	// Verify master is not mentioned in removal output.
	if strings.Contains(stdout, "Removed") && strings.Contains(stdout, "master") {
		t.Errorf("clean should not have removed master")
	}
}

func TestClean_NoCandidates(t *testing.T) {
	repo := evalDir(t, testutil.InitTestRepo(t))

	// No extra worktrees, so nothing to clean.
	stdout, _, err := runBinary(t, binPath, repo, "clean", "--merged", "--force")
	if err != nil {
		t.Fatalf("clean with no candidates failed: %v", err)
	}

	if !strings.Contains(stdout, "No worktrees to clean up") {
		t.Errorf("expected 'No worktrees to clean up', got: %s", stdout)
	}
}

func TestClean_MergedForceCombination(t *testing.T) {
	repo := evalDir(t, testutil.InitTestRepo(t))

	// Create and merge multiple branches.
	wt1 := testutil.AddWorktree(t, repo, "merged-a")
	testutil.MakeCommit(t, wt1, "feature-a")
	gitRun(t, repo, "merge", "merged-a")

	wt2 := testutil.AddWorktree(t, repo, "merged-b")
	testutil.MakeCommit(t, wt2, "feature-b")
	gitRun(t, repo, "merge", "merged-b")

	// Also create an unmerged branch that should NOT be cleaned.
	wt3 := testutil.AddWorktree(t, repo, "unmerged-c")
	testutil.MakeCommit(t, wt3, "wip")

	stdout, _, err := runBinary(t, binPath, repo, "clean", "--merged", "--force")
	if err != nil {
		t.Fatalf("clean --merged --force failed: %v", err)
	}

	// Merged branches should be removed.
	if _, statErr := os.Stat(wt1); !os.IsNotExist(statErr) {
		t.Errorf("expected merged-a to be removed")
	}
	if _, statErr := os.Stat(wt2); !os.IsNotExist(statErr) {
		t.Errorf("expected merged-b to be removed")
	}

	// Unmerged branch should be kept.
	if _, statErr := os.Stat(wt3); os.IsNotExist(statErr) {
		t.Errorf("expected unmerged-c to be preserved")
	}

	if !strings.Contains(stdout, "Cleaned up 2") {
		t.Errorf("expected 'Cleaned up 2' in output, got: %s", stdout)
	}
}

// ===========================================================================
// INIT COMMAND TESTS
// ===========================================================================

func TestInit_GlobalConfig(t *testing.T) {
	repo := evalDir(t, testutil.InitTestRepo(t))

	// Use a fake HOME directory to avoid polluting the real home.
	fakeHome := t.TempDir()

	stdout, stderr, err := runBinaryWithHome(t, binPath, repo, fakeHome, "init")
	if err != nil {
		t.Fatalf("init failed: %v\nstdout: %s\nstderr: %s", err, stdout, stderr)
	}

	expectedPath := filepath.Join(fakeHome, ".config", "git-wt", "config.toml")
	if _, statErr := os.Stat(expectedPath); os.IsNotExist(statErr) {
		t.Errorf("expected global config at %s", expectedPath)
	}

	if !strings.Contains(stdout, "Created config") {
		t.Errorf("expected 'Created config' in output, got: %s", stdout)
	}
}

func TestInit_LocalConfig(t *testing.T) {
	repo := evalDir(t, testutil.InitTestRepo(t))

	stdout, stderr, err := runBinary(t, binPath, repo, "init", "--local")
	if err != nil {
		t.Fatalf("init --local failed: %v\nstdout: %s\nstderr: %s", err, stdout, stderr)
	}

	expectedPath := filepath.Join(repo, ".git-wt.toml")
	if _, statErr := os.Stat(expectedPath); os.IsNotExist(statErr) {
		t.Errorf("expected local config at %s", expectedPath)
	}
}

func TestInit_AlreadyExists(t *testing.T) {
	repo := evalDir(t, testutil.InitTestRepo(t))

	// Create the local config first.
	writeLocalConfig(t, repo, "# existing config\n")

	_, stderr, err := runBinary(t, binPath, repo, "init", "--local")
	if err == nil {
		t.Fatal("expected error when config already exists, got nil")
	}
	if !strings.Contains(stderr, "config already exists") {
		t.Errorf("expected 'config already exists' in stderr, got: %s", stderr)
	}
}

func TestInit_ValidTOML(t *testing.T) {
	repo := evalDir(t, testutil.InitTestRepo(t))

	_, _, err := runBinary(t, binPath, repo, "init", "--local")
	if err != nil {
		t.Fatalf("init --local failed: %v", err)
	}

	configPath := filepath.Join(repo, ".git-wt.toml")
	content, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("failed to read config: %v", err)
	}

	configStr := string(content)

	// Verify key TOML sections are present.
	if !strings.Contains(configStr, "[layout]") {
		t.Error("expected [layout] section in config")
	}
	if !strings.Contains(configStr, "[cleanup]") {
		t.Error("expected [cleanup] section in config")
	}
	if !strings.Contains(configStr, "[hooks]") {
		t.Error("expected [hooks] section in config")
	}
	if !strings.Contains(configStr, `strategy = "adjacent"`) {
		t.Error("expected strategy = \"adjacent\" in config")
	}
	if !strings.Contains(configStr, `stale_days = 30`) {
		t.Error("expected stale_days = 30 in config")
	}
}

// ===========================================================================
// WORKFLOW / LIFECYCLE TESTS
// ===========================================================================

func TestWorkflow_FullLifecycle(t *testing.T) {
	repo := evalDir(t, testutil.InitTestRepo(t))

	// Step 1: Add a worktree.
	stdout, stderr, err := runBinary(t, binPath, repo, "add", "lifecycle-branch")
	if err != nil {
		t.Fatalf("add failed: %v\nstdout: %s\nstderr: %s", err, stdout, stderr)
	}

	repoName := filepath.Base(repo)
	wtPath := filepath.Join(filepath.Dir(repo), repoName+"-lifecycle-branch")

	// Step 2: List worktrees - should show the new one.
	stdout, _, err = runBinary(t, binPath, repo, "ls")
	if err != nil {
		t.Fatalf("ls failed: %v", err)
	}
	if !strings.Contains(stdout, "lifecycle-branch") {
		t.Errorf("expected 'lifecycle-branch' in ls output, got: %s", stdout)
	}

	// Step 3: Make a commit in the worktree.
	testutil.MakeCommit(t, wtPath, "lifecycle feature work")

	// Step 4: Merge into master.
	gitRun(t, repo, "merge", "lifecycle-branch")

	// Step 5: Clean merged worktrees.
	stdout, _, err = runBinary(t, binPath, repo, "clean", "--merged", "--force")
	if err != nil {
		t.Fatalf("clean failed: %v", err)
	}

	// Verify worktree was removed.
	if _, statErr := os.Stat(wtPath); !os.IsNotExist(statErr) {
		t.Error("expected lifecycle worktree to be removed after clean")
	}

	// Step 6: List again - should not show the removed worktree.
	stdout, _, err = runBinary(t, binPath, repo, "ls")
	if err != nil {
		t.Fatalf("ls after clean failed: %v", err)
	}
	if strings.Contains(stdout, "lifecycle-branch") {
		t.Errorf("expected 'lifecycle-branch' to be gone from ls output, got: %s", stdout)
	}
}

func TestWorkflow_MultipleWorktrees(t *testing.T) {
	repo := evalDir(t, testutil.InitTestRepo(t))

	// Add several worktrees.
	branches := []string{"multi-a", "multi-b", "multi-c"}
	wtPaths := make(map[string]string)
	for _, branch := range branches {
		stdout, stderr, err := runBinary(t, binPath, repo, "add", branch)
		if err != nil {
			t.Fatalf("add %s failed: %v\nstdout: %s\nstderr: %s", branch, err, stdout, stderr)
		}
		repoName := filepath.Base(repo)
		wtPaths[branch] = filepath.Join(filepath.Dir(repo), repoName+"-"+branch)
	}

	// List should show all worktrees.
	stdout, _, err := runBinary(t, binPath, repo, "ls")
	if err != nil {
		t.Fatalf("ls failed: %v", err)
	}
	for _, branch := range branches {
		if !strings.Contains(stdout, branch) {
			t.Errorf("expected '%s' in ls output, got: %s", branch, stdout)
		}
	}

	// Merge only multi-a and multi-b.
	for _, branch := range []string{"multi-a", "multi-b"} {
		testutil.MakeCommit(t, wtPaths[branch], "work on "+branch)
		gitRun(t, repo, "merge", branch)
	}

	// Also make a commit in multi-c (unmerged).
	testutil.MakeCommit(t, wtPaths["multi-c"], "wip multi-c")

	// Clean merged.
	stdout, _, err = runBinary(t, binPath, repo, "clean", "--merged", "--force")
	if err != nil {
		t.Fatalf("clean failed: %v", err)
	}

	// multi-a and multi-b should be removed.
	for _, branch := range []string{"multi-a", "multi-b"} {
		if _, statErr := os.Stat(wtPaths[branch]); !os.IsNotExist(statErr) {
			t.Errorf("expected merged worktree '%s' to be removed", branch)
		}
	}

	// multi-c should remain.
	if _, statErr := os.Stat(wtPaths["multi-c"]); os.IsNotExist(statErr) {
		t.Error("expected unmerged worktree 'multi-c' to be preserved")
	}
}

// ===========================================================================
// SWITCH COMMAND TESTS
// ===========================================================================

func TestSwitch_ExactMatch(t *testing.T) {
	repo := evalDir(t, testutil.InitTestRepo(t))

	wtPath := evalDir(t, testutil.AddWorktree(t, repo, "feature-switch"))

	stdout, stderr, err := runBinary(t, binPath, repo, "switch", "feature-switch")
	if err != nil {
		t.Fatalf("switch failed: %v\nstdout: %s\nstderr: %s", err, stdout, stderr)
	}

	got := filepath.Clean(strings.TrimSpace(stdout))
	if got != wtPath {
		t.Errorf("expected path %q, got %q", wtPath, got)
	}
}

func TestSwitch_PartialMatch(t *testing.T) {
	repo := evalDir(t, testutil.InitTestRepo(t))

	wtPath := evalDir(t, testutil.AddWorktree(t, repo, "unique-branch"))

	stdout, stderr, err := runBinary(t, binPath, repo, "switch", "unique")
	if err != nil {
		t.Fatalf("switch partial match failed: %v\nstdout: %s\nstderr: %s", err, stdout, stderr)
	}

	got := filepath.Clean(strings.TrimSpace(stdout))
	if got != wtPath {
		t.Errorf("expected path %q, got %q", wtPath, got)
	}
}

func TestSwitch_SubstringMatch(t *testing.T) {
	repo := evalDir(t, testutil.InitTestRepo(t))

	wtPath := evalDir(t, testutil.AddWorktree(t, repo, "feature-auth-v2"))

	stdout, stderr, err := runBinary(t, binPath, repo, "switch", "auth")
	if err != nil {
		t.Fatalf("switch substring match failed: %v\nstdout: %s\nstderr: %s", err, stdout, stderr)
	}

	got := filepath.Clean(strings.TrimSpace(stdout))
	if got != wtPath {
		t.Errorf("expected path %q, got %q", wtPath, got)
	}
}

func TestSwitch_NoMatch(t *testing.T) {
	repo := evalDir(t, testutil.InitTestRepo(t))

	_, stderr, err := runBinary(t, binPath, repo, "switch", "nonexistent")
	if err == nil {
		t.Fatal("expected error when no worktree matches, got nil")
	}
	if !strings.Contains(stderr, "no worktree matching") {
		t.Errorf("expected 'no worktree matching' in stderr, got: %s", stderr)
	}
}

func TestSwitch_InitZsh(t *testing.T) {
	repo := evalDir(t, testutil.InitTestRepo(t))

	stdout, stderr, err := runBinary(t, binPath, repo, "switch", "--init", "zsh")
	if err != nil {
		t.Fatalf("switch --init zsh failed: %v\nstdout: %s\nstderr: %s", err, stdout, stderr)
	}

	if !strings.Contains(stdout, "wt()") {
		t.Errorf("expected shell function 'wt()' in output, got: %s", stdout)
	}
	if !strings.Contains(stdout, "cd \"$dir\"") {
		t.Errorf("expected 'cd' in shell function, got: %s", stdout)
	}
}

func TestSwitch_InitBash(t *testing.T) {
	repo := evalDir(t, testutil.InitTestRepo(t))

	stdout, stderr, err := runBinary(t, binPath, repo, "switch", "--init", "bash")
	if err != nil {
		t.Fatalf("switch --init bash failed: %v\nstdout: %s\nstderr: %s", err, stdout, stderr)
	}

	if !strings.Contains(stdout, "wt()") {
		t.Errorf("expected shell function 'wt()' in output, got: %s", stdout)
	}
}

func TestSwitch_InitFish(t *testing.T) {
	repo := evalDir(t, testutil.InitTestRepo(t))

	stdout, stderr, err := runBinary(t, binPath, repo, "switch", "--init", "fish")
	if err != nil {
		t.Fatalf("switch --init fish failed: %v\nstdout: %s\nstderr: %s", err, stdout, stderr)
	}

	if !strings.Contains(stdout, "function wt") {
		t.Errorf("expected 'function wt' in output, got: %s", stdout)
	}
}

func TestSwitch_InitUnsupported(t *testing.T) {
	repo := evalDir(t, testutil.InitTestRepo(t))

	_, stderr, err := runBinary(t, binPath, repo, "switch", "--init", "powershell")
	if err == nil {
		t.Fatal("expected error for unsupported shell, got nil")
	}
	if !strings.Contains(stderr, "unsupported shell") {
		t.Errorf("expected 'unsupported shell' in stderr, got: %s", stderr)
	}
}

func TestSwitch_CurrentWorktree(t *testing.T) {
	repo := evalDir(t, testutil.InitTestRepo(t))

	// Switch to the current worktree (master) should work
	stdout, stderr, err := runBinary(t, binPath, repo, "switch", "master")
	if err != nil {
		t.Fatalf("switch to current worktree failed: %v\nstdout: %s\nstderr: %s", err, stdout, stderr)
	}

	got := filepath.Clean(strings.TrimSpace(stdout))
	if got != repo {
		t.Errorf("expected path %q, got %q", repo, got)
	}
}

func TestSwitch_CaseInsensitive(t *testing.T) {
	repo := evalDir(t, testutil.InitTestRepo(t))

	testutil.AddWorktree(t, repo, "MyFeature")

	stdout, stderr, err := runBinary(t, binPath, repo, "switch", "myfeature")
	if err != nil {
		t.Fatalf("case insensitive switch failed: %v\nstdout: %s\nstderr: %s", err, stdout, stderr)
	}

	got := strings.TrimSpace(stdout)
	if got == "" {
		t.Error("expected non-empty path output")
	}
}

// ===========================================================================
// Helper for init tests that need a custom HOME
// ===========================================================================

func runBinaryWithHome(t *testing.T, bin, dir, home string, args ...string) (stdout, stderr string, err error) {
	t.Helper()
	cmd := exec.Command(bin, args...)
	cmd.Dir = dir

	// Build env with custom HOME, filtering out the original HOME/USERPROFILE.
	env := []string{
		"HOME=" + home,
		"USERPROFILE=" + home,
		"GIT_CONFIG_GLOBAL=" + devNull(),
		"GIT_CONFIG_SYSTEM=" + devNull(),
		"NO_COLOR=1",
	}
	for _, e := range os.Environ() {
		if strings.HasPrefix(e, "HOME=") || strings.HasPrefix(e, "USERPROFILE=") {
			continue
		}
		if strings.HasPrefix(e, "GIT_CONFIG_GLOBAL=") {
			continue
		}
		if strings.HasPrefix(e, "GIT_CONFIG_SYSTEM=") {
			continue
		}
		if strings.HasPrefix(e, "NO_COLOR=") {
			continue
		}
		env = append(env, e)
	}
	cmd.Env = env

	var outBuf, errBuf bytes.Buffer
	cmd.Stdout = &outBuf
	cmd.Stderr = &errBuf
	err = cmd.Run()
	return outBuf.String(), errBuf.String(), err
}
