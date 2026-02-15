package git

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

type Worktree struct {
	Path       string
	Head       string
	Branch     string
	IsBare     bool
	IsDetached bool
	IsCurrent  bool

	// Status info (populated separately)
	Modified   int
	Untracked  int
	Ahead      int
	Behind     int
	IsMerged   bool
	LastCommit time.Time
}

func (w *Worktree) IsClean() bool {
	return w.Modified == 0 && w.Untracked == 0
}

func (w *Worktree) BranchShort() string {
	return strings.TrimPrefix(w.Branch, "refs/heads/")
}

func (w *Worktree) StatusText() string {
	if w.IsBare {
		return "bare"
	}
	if w.IsClean() {
		return "clean"
	}
	parts := []string{}
	if w.Modified > 0 {
		parts = append(parts, fmt.Sprintf("%d modified", w.Modified))
	}
	if w.Untracked > 0 {
		parts = append(parts, fmt.Sprintf("%d untracked", w.Untracked))
	}
	return strings.Join(parts, ", ")
}

func (w *Worktree) SyncText() string {
	if w.Ahead == 0 && w.Behind == 0 {
		return "-"
	}
	parts := []string{}
	if w.Behind > 0 {
		parts = append(parts, fmt.Sprintf("↓%d", w.Behind))
	}
	if w.Ahead > 0 {
		parts = append(parts, fmt.Sprintf("↑%d", w.Ahead))
	}
	return strings.Join(parts, " ")
}

func (w *Worktree) ShortHead() string {
	if len(w.Head) >= 8 {
		return w.Head[:8]
	}
	return w.Head
}

func (w *Worktree) InactiveDays() int {
	if w.LastCommit.IsZero() {
		return 0
	}
	return int(time.Since(w.LastCommit).Hours() / 24)
}

// ListWorktrees parses `git worktree list --porcelain` output.
func ListWorktrees(repoDir string) ([]Worktree, error) {
	out, err := run(repoDir, "worktree", "list", "--porcelain")
	if err != nil {
		return nil, err
	}

	var worktrees []Worktree
	var current *Worktree

	scanner := bufio.NewScanner(strings.NewReader(out))
	for scanner.Scan() {
		line := scanner.Text()
		switch {
		case strings.HasPrefix(line, "worktree "):
			if current != nil {
				worktrees = append(worktrees, *current)
			}
			current = &Worktree{Path: strings.TrimPrefix(line, "worktree ")}
		case strings.HasPrefix(line, "HEAD "):
			if current != nil {
				current.Head = strings.TrimPrefix(line, "HEAD ")
			}
		case strings.HasPrefix(line, "branch "):
			if current != nil {
				current.Branch = strings.TrimPrefix(line, "branch ")
			}
		case line == "bare":
			if current != nil {
				current.IsBare = true
			}
		case line == "detached":
			if current != nil {
				current.IsDetached = true
			}
		}
	}
	if current != nil {
		worktrees = append(worktrees, *current)
	}

	// Determine which one is current
	cwd, err := os.Getwd()
	if err == nil {
		cwd, _ = filepath.Abs(cwd)
		for i := range worktrees {
			wtAbs, _ := filepath.Abs(worktrees[i].Path)
			if cwd == wtAbs || strings.HasPrefix(cwd, wtAbs+string(os.PathSeparator)) {
				worktrees[i].IsCurrent = true
			}
		}
	}

	return worktrees, nil
}

// EnrichWorktree populates status, ahead/behind, merge status, and last commit.
func EnrichWorktree(w *Worktree, defaultBranch string) {
	if w.IsBare {
		return
	}
	if _, err := os.Stat(w.Path); os.IsNotExist(err) {
		return
	}

	// Modified + untracked count
	if out, err := run(w.Path, "status", "--porcelain"); err == nil && out != "" {
		for _, line := range strings.Split(out, "\n") {
			if len(line) < 2 {
				continue
			}
			if line[:2] == "??" {
				w.Untracked++
			} else {
				w.Modified++
			}
		}
	}

	// Ahead/behind upstream
	if out, err := run(w.Path, "rev-list", "--left-right", "--count", "HEAD...@{upstream}"); err == nil {
		parts := strings.Fields(out)
		if len(parts) == 2 {
			w.Ahead, _ = strconv.Atoi(parts[0])
			w.Behind, _ = strconv.Atoi(parts[1])
		}
	}

	// Merged into default branch
	branch := w.BranchShort()
	if branch != "" && branch != defaultBranch {
		if out, err := run(w.Path, "branch", "--merged", defaultBranch); err == nil {
			for _, line := range strings.Split(out, "\n") {
				name := strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(line), "* "))
				if name == branch {
					w.IsMerged = true
					break
				}
			}
		}
	}

	// Last commit time
	if out, err := run(w.Path, "log", "-1", "--format=%ct"); err == nil && out != "" {
		if ts, err := strconv.ParseInt(out, 10, 64); err == nil {
			w.LastCommit = time.Unix(ts, 0)
		}
	}
}

// AddWorktree creates a new worktree at targetPath for the given branch.
func AddWorktree(repoDir, targetPath, branch, baseBranch string) error {
	if BranchExists(repoDir, branch) {
		_, err := run(repoDir, "worktree", "add", targetPath, branch)
		return err
	}
	// Create new branch from baseBranch
	args := []string{"worktree", "add", "-b", branch, targetPath}
	if baseBranch != "" {
		args = append(args, baseBranch)
	}
	_, err := run(repoDir, args...)
	return err
}

// RemoveWorktree removes a worktree and optionally deletes the branch.
func RemoveWorktree(repoDir, wtPath string, deleteBranch bool) error {
	// Get branch name before removal
	var branchName string
	if deleteBranch {
		worktrees, err := ListWorktrees(repoDir)
		if err == nil {
			for _, wt := range worktrees {
				absWt, _ := filepath.Abs(wt.Path)
				absTarget, _ := filepath.Abs(wtPath)
				if absWt == absTarget {
					branchName = wt.BranchShort()
					break
				}
			}
		}
	}

	if _, err := run(repoDir, "worktree", "remove", wtPath); err != nil {
		// Try force removal
		if _, err := run(repoDir, "worktree", "remove", "--force", wtPath); err != nil {
			return err
		}
	}

	if deleteBranch && branchName != "" {
		_, _ = run(repoDir, "branch", "-d", branchName)
	}
	return nil
}

// PruneWorktrees cleans up stale worktree references.
func PruneWorktrees(repoDir string) error {
	_, err := run(repoDir, "worktree", "prune")
	return err
}
