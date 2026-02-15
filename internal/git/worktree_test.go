package git

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/yasomaru/git-wt/testutil"
)

// realAbs returns a symlink-resolved absolute path. On macOS, /var is a
// symlink to /private/var which causes naive filepath.Abs comparisons to fail.
func realAbs(t *testing.T, path string) string {
	t.Helper()
	abs, err := filepath.Abs(path)
	if err != nil {
		t.Fatalf("filepath.Abs(%q): %v", path, err)
	}
	resolved, err := filepath.EvalSymlinks(abs)
	if err != nil {
		// Path may not exist (e.g. after removal); return abs as fallback.
		return abs
	}
	return resolved
}

// findWorktreeByPath returns the Worktree whose resolved path matches target,
// or nil if not found.
func findWorktreeByPath(t *testing.T, worktrees []Worktree, target string) *Worktree {
	t.Helper()
	resolvedTarget := realAbs(t, target)
	for i := range worktrees {
		if realAbs(t, worktrees[i].Path) == resolvedTarget {
			return &worktrees[i]
		}
	}
	return nil
}

// ---------------------------------------------------------------------------
// Table-driven method tests
// ---------------------------------------------------------------------------

func TestBranchShort(t *testing.T) {
	tests := []struct {
		name   string
		branch string
		want   string
	}{
		{name: "full ref", branch: "refs/heads/feature-x", want: "feature-x"},
		{name: "nested ref", branch: "refs/heads/team/feature-y", want: "team/feature-y"},
		{name: "already short", branch: "main", want: "main"},
		{name: "empty string", branch: "", want: ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := &Worktree{Branch: tt.branch}
			got := w.BranchShort()
			if got != tt.want {
				t.Errorf("BranchShort() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestStatusText(t *testing.T) {
	tests := []struct {
		name string
		w    Worktree
		want string
	}{
		{
			name: "bare repository",
			w:    Worktree{IsBare: true, Modified: 5, Untracked: 3},
			want: "bare",
		},
		{
			name: "clean worktree",
			w:    Worktree{Modified: 0, Untracked: 0},
			want: "clean",
		},
		{
			name: "modified only",
			w:    Worktree{Modified: 3},
			want: "3 modified",
		},
		{
			name: "untracked only",
			w:    Worktree{Untracked: 7},
			want: "7 untracked",
		},
		{
			name: "both modified and untracked",
			w:    Worktree{Modified: 2, Untracked: 4},
			want: "2 modified, 4 untracked",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.w.StatusText()
			if got != tt.want {
				t.Errorf("StatusText() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestSyncText(t *testing.T) {
	tests := []struct {
		name   string
		ahead  int
		behind int
		want   string
	}{
		{name: "both zero", ahead: 0, behind: 0, want: "-"},
		{name: "ahead only", ahead: 3, behind: 0, want: "\u2191" + "3"},
		{name: "behind only", ahead: 0, behind: 5, want: "\u2193" + "5"},
		{name: "both ahead and behind", ahead: 2, behind: 4, want: "\u2193" + "4 " + "\u2191" + "2"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := &Worktree{Ahead: tt.ahead, Behind: tt.behind}
			got := w.SyncText()
			if got != tt.want {
				t.Errorf("SyncText() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestShortHead(t *testing.T) {
	tests := []struct {
		name string
		head string
		want string
	}{
		{
			name: "full 40 char SHA",
			head: "abc123def456789012345678901234567890abcd",
			want: "abc123de",
		},
		{
			name: "exactly 8 chars",
			head: "abc123de",
			want: "abc123de",
		},
		{
			name: "short SHA under 8 chars",
			head: "abc12",
			want: "abc12",
		},
		{
			name: "empty head",
			head: "",
			want: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := &Worktree{Head: tt.head}
			got := w.ShortHead()
			if got != tt.want {
				t.Errorf("ShortHead() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestIsClean(t *testing.T) {
	tests := []struct {
		name      string
		modified  int
		untracked int
		want      bool
	}{
		{name: "clean", modified: 0, untracked: 0, want: true},
		{name: "has modified", modified: 1, untracked: 0, want: false},
		{name: "has untracked", modified: 0, untracked: 1, want: false},
		{name: "has both", modified: 2, untracked: 3, want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := &Worktree{Modified: tt.modified, Untracked: tt.untracked}
			got := w.IsClean()
			if got != tt.want {
				t.Errorf("IsClean() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestInactiveDays(t *testing.T) {
	tests := []struct {
		name       string
		lastCommit time.Time
		wantZero   bool
		wantMin    int
	}{
		{
			name:       "zero time returns 0",
			lastCommit: time.Time{},
			wantZero:   true,
		},
		{
			name:       "recent commit today",
			lastCommit: time.Now().Add(-1 * time.Hour),
			wantZero:   true,
		},
		{
			name:       "commit 10 days ago",
			lastCommit: time.Now().Add(-10 * 24 * time.Hour),
			wantMin:    9, // allow 1 day tolerance for time boundaries
		},
		{
			name:       "commit 100 days ago",
			lastCommit: time.Now().Add(-100 * 24 * time.Hour),
			wantMin:    99,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := &Worktree{LastCommit: tt.lastCommit}
			got := w.InactiveDays()
			if tt.wantZero {
				if got != 0 {
					t.Errorf("InactiveDays() = %d, want 0", got)
				}
				return
			}
			if got < tt.wantMin {
				t.Errorf("InactiveDays() = %d, want >= %d", got, tt.wantMin)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Integration tests using real git repos
// ---------------------------------------------------------------------------

func TestListWorktrees_MainOnly(t *testing.T) {
	dir := testutil.InitTestRepo(t)

	worktrees, err := ListWorktrees(dir)
	if err != nil {
		t.Fatalf("ListWorktrees() error: %v", err)
	}

	if len(worktrees) != 1 {
		t.Fatalf("expected 1 worktree, got %d", len(worktrees))
	}

	wt := worktrees[0]
	if realAbs(t, wt.Path) != realAbs(t, dir) {
		t.Errorf("worktree path = %q, want %q", realAbs(t, wt.Path), realAbs(t, dir))
	}
	if wt.Head == "" {
		t.Error("expected non-empty HEAD")
	}
	if wt.Branch == "" {
		t.Error("expected non-empty branch")
	}
	if wt.IsBare {
		t.Error("expected IsBare = false")
	}
}

func TestListWorktrees_WithAdded(t *testing.T) {
	dir := testutil.InitTestRepo(t)
	_ = testutil.AddWorktree(t, dir, "feature-a")
	_ = testutil.AddWorktree(t, dir, "feature-b")

	worktrees, err := ListWorktrees(dir)
	if err != nil {
		t.Fatalf("ListWorktrees() error: %v", err)
	}

	if len(worktrees) != 3 {
		t.Fatalf("expected 3 worktrees, got %d", len(worktrees))
	}

	branches := make(map[string]bool)
	for _, wt := range worktrees {
		branches[wt.BranchShort()] = true
	}
	for _, want := range []string{"feature-a", "feature-b"} {
		if !branches[want] {
			t.Errorf("expected branch %q in worktrees, got branches: %v", want, branches)
		}
	}
}

func TestListWorktrees_InvalidDir(t *testing.T) {
	_, err := ListWorktrees("/nonexistent/path")
	if err == nil {
		t.Error("expected error for nonexistent directory")
	}
}

func TestAddWorktree_NewBranch(t *testing.T) {
	dir := testutil.InitTestRepo(t)
	targetPath := filepath.Join(t.TempDir(), "new-feature")

	err := AddWorktree(dir, targetPath, "new-feature", "")
	if err != nil {
		t.Fatalf("AddWorktree() error: %v", err)
	}
	t.Cleanup(func() {
		run(dir, "worktree", "remove", "--force", targetPath)
	})

	if _, err := os.Stat(targetPath); os.IsNotExist(err) {
		t.Fatal("expected worktree directory to exist")
	}
	if !BranchExists(dir, "new-feature") {
		t.Error("expected branch 'new-feature' to exist")
	}
}

func TestAddWorktree_ExistingBranch(t *testing.T) {
	dir := testutil.InitTestRepo(t)
	testutil.CreateBranch(t, dir, "existing-branch")
	targetPath := filepath.Join(t.TempDir(), "existing-branch")

	err := AddWorktree(dir, targetPath, "existing-branch", "")
	if err != nil {
		t.Fatalf("AddWorktree() error: %v", err)
	}
	t.Cleanup(func() {
		run(dir, "worktree", "remove", "--force", targetPath)
	})

	if _, err := os.Stat(targetPath); os.IsNotExist(err) {
		t.Fatal("expected worktree directory to exist")
	}

	worktrees, err := ListWorktrees(dir)
	if err != nil {
		t.Fatalf("ListWorktrees() error: %v", err)
	}

	found := false
	for _, wt := range worktrees {
		if wt.BranchShort() == "existing-branch" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected 'existing-branch' in worktree list")
	}
}

func TestAddWorktree_WithBaseBranch(t *testing.T) {
	dir := testutil.InitTestRepo(t)
	testutil.MakeCommit(t, dir, "second commit on main")

	targetPath := filepath.Join(t.TempDir(), "from-main")

	err := AddWorktree(dir, targetPath, "from-main", "HEAD~1")
	if err != nil {
		t.Fatalf("AddWorktree() with baseBranch error: %v", err)
	}
	t.Cleanup(func() {
		run(dir, "worktree", "remove", "--force", targetPath)
	})

	if _, err := os.Stat(targetPath); os.IsNotExist(err) {
		t.Fatal("expected worktree directory to exist")
	}
	if !BranchExists(dir, "from-main") {
		t.Error("expected branch 'from-main' to exist")
	}
}

func TestRemoveWorktree_Basic(t *testing.T) {
	dir := testutil.InitTestRepo(t)
	wtPath := testutil.AddWorktree(t, dir, "to-remove")

	err := RemoveWorktree(dir, wtPath, false)
	if err != nil {
		t.Fatalf("RemoveWorktree() error: %v", err)
	}

	if _, err := os.Stat(wtPath); !os.IsNotExist(err) {
		t.Error("expected worktree directory to be removed")
	}

	// Branch should still exist when deleteBranch is false
	if !BranchExists(dir, "to-remove") {
		t.Error("expected branch 'to-remove' to still exist")
	}
}

func TestRemoveWorktree_WithBranchDeletion(t *testing.T) {
	dir := testutil.InitTestRepo(t)
	wtPath := testutil.AddWorktree(t, dir, "delete-me")

	// The branch was created from the same commit as the default branch, so
	// git branch -d considers it merged. However, RemoveWorktree compares
	// resolved paths internally with filepath.Abs. We pass the resolved path
	// to ensure the match succeeds on platforms with symlinks (e.g. macOS).
	resolvedWtPath := realAbs(t, wtPath)

	err := RemoveWorktree(dir, resolvedWtPath, true)
	if err != nil {
		t.Fatalf("RemoveWorktree() error: %v", err)
	}

	if _, err := os.Stat(wtPath); !os.IsNotExist(err) {
		t.Error("expected worktree directory to be removed")
	}

	// Branch should be deleted when deleteBranch is true
	if BranchExists(dir, "delete-me") {
		t.Error("expected branch 'delete-me' to be deleted")
	}
}

func TestEnrichWorktree_Status(t *testing.T) {
	dir := testutil.InitTestRepo(t)

	// Create an untracked file
	testutil.WriteFile(t, dir, "untracked.txt", "untracked content\n")

	// Modify an already-tracked file
	testutil.WriteFile(t, dir, "README.md", "modified content\n")

	worktrees, err := ListWorktrees(dir)
	if err != nil {
		t.Fatalf("ListWorktrees() error: %v", err)
	}
	if len(worktrees) == 0 {
		t.Fatal("expected at least 1 worktree")
	}

	wt := &worktrees[0]
	EnrichWorktree(wt, "main")

	if wt.Modified < 1 {
		t.Errorf("expected Modified >= 1, got %d", wt.Modified)
	}
	if wt.Untracked < 1 {
		t.Errorf("expected Untracked >= 1, got %d", wt.Untracked)
	}
	if wt.LastCommit.IsZero() {
		t.Error("expected LastCommit to be populated")
	}
}

func TestEnrichWorktree_MergedBranch(t *testing.T) {
	dir := testutil.InitTestRepo(t)
	wtPath := testutil.AddWorktree(t, dir, "merged-feature")

	worktrees, err := ListWorktrees(dir)
	if err != nil {
		t.Fatalf("ListWorktrees() error: %v", err)
	}

	wt := findWorktreeByPath(t, worktrees, wtPath)
	if wt == nil {
		t.Fatal("could not find worktree for merged-feature")
	}

	defaultBranch, err := DefaultBranch(dir)
	if err != nil {
		defaultBranch = "main"
	}

	EnrichWorktree(wt, defaultBranch)

	// Branch was created from the same commit as the default branch, so it
	// should be considered merged.
	if !wt.IsMerged {
		t.Error("expected IsMerged = true for a branch at the same commit as default")
	}
}

func TestEnrichWorktree_UnmergedBranch(t *testing.T) {
	dir := testutil.InitTestRepo(t)
	wtPath := testutil.AddWorktree(t, dir, "unmerged-feature")
	testutil.MakeCommit(t, wtPath, "diverge")

	worktrees, err := ListWorktrees(dir)
	if err != nil {
		t.Fatalf("ListWorktrees() error: %v", err)
	}

	wt := findWorktreeByPath(t, worktrees, wtPath)
	if wt == nil {
		t.Fatal("could not find worktree for unmerged-feature")
	}

	defaultBranch, err := DefaultBranch(dir)
	if err != nil {
		defaultBranch = "main"
	}

	EnrichWorktree(wt, defaultBranch)

	if wt.IsMerged {
		t.Error("expected IsMerged = false for a branch with commits ahead of default")
	}
}

func TestEnrichWorktree_BareSkipped(t *testing.T) {
	w := &Worktree{IsBare: true}
	EnrichWorktree(w, "main")

	if w.Modified != 0 || w.Untracked != 0 || !w.LastCommit.IsZero() {
		t.Error("expected bare worktree to be unchanged after EnrichWorktree")
	}
}

func TestEnrichWorktree_LastCommitPopulated(t *testing.T) {
	dir := testutil.InitTestRepo(t)

	worktrees, err := ListWorktrees(dir)
	if err != nil {
		t.Fatalf("ListWorktrees() error: %v", err)
	}

	wt := &worktrees[0]
	EnrichWorktree(wt, "main")

	if wt.LastCommit.IsZero() {
		t.Error("expected LastCommit to be populated after enrichment")
	}

	elapsed := time.Since(wt.LastCommit)
	if elapsed > 5*time.Minute {
		t.Errorf("LastCommit is too old: %v ago", elapsed)
	}
}

func TestPruneWorktrees(t *testing.T) {
	dir := testutil.InitTestRepo(t)

	err := PruneWorktrees(dir)
	if err != nil {
		t.Fatalf("PruneWorktrees() error: %v", err)
	}

	worktrees, err := ListWorktrees(dir)
	if err != nil {
		t.Fatalf("ListWorktrees() after prune error: %v", err)
	}
	if len(worktrees) != 1 {
		t.Errorf("expected 1 worktree after prune, got %d", len(worktrees))
	}
}

func TestPruneWorktrees_StaleReference(t *testing.T) {
	dir := testutil.InitTestRepo(t)
	wtPath := testutil.AddWorktree(t, dir, "stale-branch")

	// Manually remove the worktree directory to create a stale reference.
	if err := os.RemoveAll(wtPath); err != nil {
		t.Fatalf("failed to remove worktree dir: %v", err)
	}

	before, err := ListWorktrees(dir)
	if err != nil {
		t.Fatalf("ListWorktrees() error: %v", err)
	}
	if len(before) < 2 {
		t.Fatalf("expected at least 2 worktrees before prune (including stale), got %d", len(before))
	}

	err = PruneWorktrees(dir)
	if err != nil {
		t.Fatalf("PruneWorktrees() error: %v", err)
	}

	after, err := ListWorktrees(dir)
	if err != nil {
		t.Fatalf("ListWorktrees() after prune error: %v", err)
	}
	if len(after) != 1 {
		t.Errorf("expected 1 worktree after pruning stale reference, got %d", len(after))
	}
}
