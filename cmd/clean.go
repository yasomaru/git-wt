package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/fatih/color"
	"github.com/spf13/cobra"

	"github.com/yasomaru/git-wt/internal/config"
	"github.com/yasomaru/git-wt/internal/git"
)

var cleanCmd = &cobra.Command{
	Use:   "clean",
	Short: "Remove merged or stale worktrees",
	Long: `Identify and remove worktrees that are no longer needed.

By default, shows candidates interactively for confirmation.
Use --merged to target only branches merged into the default branch.
Use --stale to target branches inactive for a specified number of days.`,
	Example: `  git wt clean              # interactive cleanup
  git wt clean --merged     # remove merged worktrees
  git wt clean --stale 30   # remove worktrees inactive for 30+ days
  git wt clean --dry-run    # preview only, no changes`,
	RunE: runClean,
}

var (
	cleanMerged    bool
	cleanStaleDays int
	cleanDryRun    bool
	cleanForce     bool
)

func init() {
	cleanCmd.Flags().BoolVar(&cleanMerged, "merged", false, "remove worktrees with merged branches")
	cleanCmd.Flags().IntVar(&cleanStaleDays, "stale", 0, "remove worktrees inactive for N days")
	cleanCmd.Flags().BoolVar(&cleanDryRun, "dry-run", false, "preview candidates without removing")
	cleanCmd.Flags().BoolVarP(&cleanForce, "force", "f", false, "skip confirmation prompt")
	rootCmd.AddCommand(cleanCmd)
}

func runClean(cmd *cobra.Command, args []string) error {
	repoRoot, err := git.RepoRoot("")
	if err != nil {
		return fmt.Errorf("not a git repository")
	}

	cfg := config.LoadForRepo(repoRoot)

	worktrees, err := git.ListWorktrees(repoRoot)
	if err != nil {
		return err
	}

	defaultBranch, _ := git.DefaultBranch(repoRoot)

	// Enrich all worktrees
	for i := range worktrees {
		git.EnrichWorktree(&worktrees[i], defaultBranch)
	}

	// Determine effective stale days threshold
	staleDays := cleanStaleDays
	hasExplicitFlags := cleanMerged || cleanStaleDays > 0
	if !hasExplicitFlags {
		staleDays = cfg.Cleanup.StaleDays
	}

	type candidate struct {
		worktree git.Worktree
		reason   string
	}

	// Filter candidates
	var candidates []candidate
	for _, wt := range worktrees {
		if wt.IsBare || wt.IsCurrent {
			continue
		}
		branch := wt.BranchShort()
		if branch == defaultBranch {
			continue
		}

		var reasons []string

		if hasExplicitFlags {
			// Explicit flags: only match requested criteria
			if cleanMerged && wt.IsMerged {
				reasons = append(reasons, "merged")
			}
			if cleanStaleDays > 0 && wt.InactiveDays() >= staleDays {
				reasons = append(reasons, fmt.Sprintf("%dd inactive", wt.InactiveDays()))
			}
		} else {
			// No flags: show both merged and stale (using config threshold)
			if wt.IsMerged {
				reasons = append(reasons, "merged")
			}
			if staleDays > 0 && wt.InactiveDays() >= staleDays {
				reasons = append(reasons, fmt.Sprintf("%dd inactive", wt.InactiveDays()))
			}
		}

		if len(reasons) > 0 {
			candidates = append(candidates, candidate{
				worktree: wt,
				reason:   strings.Join(reasons, ", "),
			})
		}
	}

	if len(candidates) == 0 {
		color.Green("  No worktrees to clean up.")
		if cfg.Cleanup.AutoPrune {
			_ = git.PruneWorktrees(repoRoot)
		}
		return nil
	}

	// Display candidates
	fmt.Printf("\n  Worktrees to remove (%d):\n\n", len(candidates))
	for _, c := range candidates {
		wt := c.worktree
		branch := wt.BranchShort()
		tags := []string{c.reason}
		if !wt.IsClean() {
			tags = append(tags, color.YellowString(wt.StatusText()))
		}

		fmt.Printf("    %s  %s  [%s]\n",
			color.CyanString(branch),
			wt.Path,
			strings.Join(tags, ", "),
		)
	}
	fmt.Println()

	if cleanDryRun {
		color.Yellow("  Dry run - no changes made.")
		return nil
	}

	// Confirm
	if !cleanForce {
		fmt.Print("  Remove these worktrees? (y/N): ")
		reader := bufio.NewReader(os.Stdin)
		answer, _ := reader.ReadString('\n')
		answer = strings.TrimSpace(strings.ToLower(answer))
		if answer != "y" && answer != "yes" {
			fmt.Println("  Cancelled.")
			return nil
		}
	}

	// Remove
	removed := 0
	for _, c := range candidates {
		wt := c.worktree
		branch := wt.BranchShort()
		deleteBranch := wt.IsMerged
		if err := git.RemoveWorktree(repoRoot, wt.Path, deleteBranch); err != nil {
			color.Red("  Failed to remove %s: %v", branch, err)
			continue
		}
		color.Green("  Removed: %s", branch)
		removed++
	}

	// Prune
	if cfg.Cleanup.AutoPrune {
		_ = git.PruneWorktrees(repoRoot)
	}

	fmt.Printf("\n  Cleaned up %d worktree(s).\n", removed)
	return nil
}
