package cmd

import (
	"fmt"
	"strings"

	"github.com/fatih/color"
	"github.com/spf13/cobra"

	"github.com/yasomaru/git-wt/internal/git"
)

var lsCmd = &cobra.Command{
	Use:     "ls",
	Aliases: []string{"list"},
	Short:   "List all worktrees with status",
	RunE:    runLs,
}

func init() {
	rootCmd.AddCommand(lsCmd)
}

func runLs(cmd *cobra.Command, args []string) error {
	repoRoot, err := git.RepoRoot("")
	if err != nil {
		return fmt.Errorf("not a git repository")
	}

	worktrees, err := git.ListWorktrees(repoRoot)
	if err != nil {
		return err
	}

	if len(worktrees) == 0 {
		fmt.Println("No worktrees found.")
		return nil
	}

	defaultBranch, _ := git.DefaultBranch(repoRoot)

	// Enrich each worktree with status info
	for i := range worktrees {
		git.EnrichWorktree(&worktrees[i], defaultBranch)
	}

	printWorktreeTable(worktrees)
	return nil
}

func printWorktreeTable(worktrees []git.Worktree) {
	// Calculate column widths
	branchW := len("Branch")
	pathW := len("Path")
	statusW := len("Status")

	for _, wt := range worktrees {
		name := wt.BranchShort()
		if wt.IsDetached {
			name = wt.ShortHead() + " (detached)"
		}
		if wt.IsBare {
			name = "(bare)"
		}
		if len(name) > branchW {
			branchW = len(name)
		}
		if len(wt.Path) > pathW {
			pathW = len(wt.Path)
		}
		st := wt.StatusText()
		if len(st) > statusW {
			statusW = len(st)
		}
	}

	// Cap path width
	if pathW > 50 {
		pathW = 50
	}

	// Header
	header := color.New(color.Bold)
	header.Printf("  %-*s  %-*s  %-*s  %s\n",
		branchW, "Branch",
		pathW, "Path",
		statusW, "Status",
		"Sync",
	)
	fmt.Println("  " + strings.Repeat("─", branchW+pathW+statusW+20))

	for _, wt := range worktrees {
		name := wt.BranchShort()
		if wt.IsDetached {
			name = wt.ShortHead() + " (detached)"
		}
		if wt.IsBare {
			name = "(bare)"
		}

		// Color the branch name
		branchStr := name
		if wt.IsCurrent {
			branchStr = color.GreenString("* " + name)
			// Adjust for the "* " prefix when padding
			branchStr = fmt.Sprintf("%-*s", branchW+2, branchStr)
		} else {
			branchStr = fmt.Sprintf("  %-*s", branchW, name)
		}

		// Path (truncate if needed)
		pathStr := wt.Path
		if len(pathStr) > pathW {
			pathStr = "..." + pathStr[len(pathStr)-pathW+3:]
		}

		// Status with color
		statusText := wt.StatusText()
		var statusStr string
		if wt.IsClean() {
			statusStr = color.GreenString("%-*s", statusW, statusText)
		} else {
			statusStr = color.YellowString("%-*s", statusW, statusText)
		}

		// Sync info
		syncText := wt.SyncText()
		if wt.IsMerged {
			syncText += " " + color.GreenString("(merged)")
		}
		days := wt.InactiveDays()
		if days > 30 {
			syncText += " " + color.RedString("(%dd stale)", days)
		}

		fmt.Printf("%s  %-*s  %s  %s\n",
			branchStr,
			pathW, pathStr,
			statusStr,
			syncText,
		)
	}

	fmt.Println()
}
