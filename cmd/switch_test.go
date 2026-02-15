package cmd

import (
	"testing"

	"github.com/yasomaru/git-wt/internal/git"
)

func testCandidates() []git.Worktree {
	return []git.Worktree{
		{Path: "/repo", Branch: "refs/heads/main", IsCurrent: true},
		{Path: "/repo-feature-auth", Branch: "refs/heads/feature-auth"},
		{Path: "/repo-feature-api", Branch: "refs/heads/feature-api"},
		{Path: "/repo-hotfix-123", Branch: "refs/heads/hotfix-123"},
	}
}

func TestMatchWorktrees_ExactMatch(t *testing.T) {
	t.Parallel()
	matches := matchWorktrees(testCandidates(), "feature-auth")
	if len(matches) != 1 {
		t.Fatalf("expected 1 exact match, got %d", len(matches))
	}
	if matches[0].BranchShort() != "feature-auth" {
		t.Errorf("expected feature-auth, got %s", matches[0].BranchShort())
	}
}

func TestMatchWorktrees_ExactMatchTakesPrecedence(t *testing.T) {
	t.Parallel()
	// "main" is both an exact match and a substring of other potential branches
	matches := matchWorktrees(testCandidates(), "main")
	if len(matches) != 1 {
		t.Fatalf("expected 1 match for exact 'main', got %d", len(matches))
	}
	if matches[0].BranchShort() != "main" {
		t.Errorf("expected main, got %s", matches[0].BranchShort())
	}
}

func TestMatchWorktrees_PrefixMatch(t *testing.T) {
	t.Parallel()
	matches := matchWorktrees(testCandidates(), "feature")
	if len(matches) != 2 {
		t.Fatalf("expected 2 prefix matches for 'feature', got %d", len(matches))
	}
	branches := map[string]bool{}
	for _, m := range matches {
		branches[m.BranchShort()] = true
	}
	if !branches["feature-auth"] || !branches["feature-api"] {
		t.Errorf("expected feature-auth and feature-api, got %v", branches)
	}
}

func TestMatchWorktrees_ContainsMatch(t *testing.T) {
	t.Parallel()
	matches := matchWorktrees(testCandidates(), "auth")
	if len(matches) != 1 {
		t.Fatalf("expected 1 substring match for 'auth', got %d", len(matches))
	}
	if matches[0].BranchShort() != "feature-auth" {
		t.Errorf("expected feature-auth, got %s", matches[0].BranchShort())
	}
}

func TestMatchWorktrees_CaseInsensitive(t *testing.T) {
	t.Parallel()

	t.Run("exact match ignores case", func(t *testing.T) {
		t.Parallel()
		matches := matchWorktrees(testCandidates(), "MAIN")
		if len(matches) != 1 {
			t.Fatalf("expected 1 match for 'MAIN', got %d", len(matches))
		}
		if matches[0].BranchShort() != "main" {
			t.Errorf("expected main, got %s", matches[0].BranchShort())
		}
	})

	t.Run("prefix match ignores case", func(t *testing.T) {
		t.Parallel()
		matches := matchWorktrees(testCandidates(), "FEATURE")
		if len(matches) != 2 {
			t.Fatalf("expected 2 matches for 'FEATURE', got %d", len(matches))
		}
	})

	t.Run("substring match ignores case", func(t *testing.T) {
		t.Parallel()
		matches := matchWorktrees(testCandidates(), "AUTH")
		if len(matches) != 1 {
			t.Fatalf("expected 1 match for 'AUTH', got %d", len(matches))
		}
	})
}

func TestMatchWorktrees_NoMatch(t *testing.T) {
	t.Parallel()
	matches := matchWorktrees(testCandidates(), "nonexistent")
	if len(matches) != 0 {
		t.Errorf("expected 0 matches for 'nonexistent', got %d", len(matches))
	}
}

func TestMatchWorktrees_SinglePrefixMatch(t *testing.T) {
	t.Parallel()
	matches := matchWorktrees(testCandidates(), "hotfix")
	if len(matches) != 1 {
		t.Fatalf("expected 1 prefix match for 'hotfix', got %d", len(matches))
	}
	if matches[0].BranchShort() != "hotfix-123" {
		t.Errorf("expected hotfix-123, got %s", matches[0].BranchShort())
	}
}

func TestMatchWorktrees_EmptyCandidates(t *testing.T) {
	t.Parallel()
	matches := matchWorktrees(nil, "anything")
	if len(matches) != 0 {
		t.Errorf("expected 0 matches on empty candidates, got %d", len(matches))
	}
}
