package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/yasomaru/git-wt/internal/git"
	"github.com/yasomaru/git-wt/internal/tui"
)

var (
	appVersion = "dev"
	appCommit  = "none"
	appDate    = "unknown"
)

func SetVersionInfo(version, commit, date string) {
	appVersion = version
	appCommit = commit
	appDate = date
}

var rootCmd = &cobra.Command{
	Use:   "git-wt",
	Short: "A smarter way to manage git worktrees",
	Long: `git-wt simplifies git worktree management with smart defaults,
rich status display, and easy cleanup.

Run without arguments to open the interactive TUI.

Usage as a git subcommand:
  git wt              Open interactive worktree manager
  git wt add <branch>     Create a worktree with automatic path and branch setup
  git wt ls               List all worktrees with status information
  git wt clean            Remove merged or stale worktrees`,
	SilenceUsage:  true,
	SilenceErrors: true,
	RunE:          runRoot,
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print version information",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("git-wt %s\ncommit: %s\nbuilt:  %s\n", appVersion, appCommit, appDate)
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}

func runRoot(cmd *cobra.Command, args []string) error {
	repoDir, err := git.RepoRoot("")
	if err != nil {
		return fmt.Errorf("not a git repository")
	}

	worktrees, err := git.ListWorktrees(repoDir)
	if err != nil {
		return err
	}

	defaultBranch, _ := git.DefaultBranch(repoDir)
	for i := range worktrees {
		git.EnrichWorktree(&worktrees[i], defaultBranch)
	}

	return tui.Run(worktrees, repoDir)
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
