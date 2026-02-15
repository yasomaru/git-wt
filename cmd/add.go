package cmd

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/fatih/color"
	"github.com/spf13/cobra"

	"github.com/yasomaru/git-wt/internal/config"
	"github.com/yasomaru/git-wt/internal/git"
)

var addCmd = &cobra.Command{
	Use:   "add <branch>",
	Short: "Create a new worktree",
	Long: `Create a new worktree with automatic path resolution and branch management.

If the branch already exists, it checks it out in the new worktree.
If it doesn't exist, a new branch is created from the base branch.`,
	Args: cobra.ExactArgs(1),
	Example: `  git wt add feature-auth
  git wt add feature-auth -b main
  git wt add hotfix-123 -b release/v2`,
	RunE: runAdd,
}

var addBase string

func init() {
	addCmd.Flags().StringVarP(&addBase, "base", "b", "", "base branch to create from (default: current HEAD)")
	rootCmd.AddCommand(addCmd)
}

func runAdd(cmd *cobra.Command, args []string) error {
	branch := args[0]

	repoRoot, err := git.RepoRoot("")
	if err != nil {
		return fmt.Errorf("not a git repository")
	}

	cfg := config.LoadForRepo(repoRoot)
	targetPath := cfg.WorktreePath(repoRoot, branch)

	// Check if path already exists
	if _, err := os.Stat(targetPath); err == nil {
		return fmt.Errorf("path already exists: %s", targetPath)
	}

	if err := git.AddWorktree(repoRoot, targetPath, branch, addBase); err != nil {
		return err
	}

	success := color.New(color.FgGreen, color.Bold)
	success.Printf("  Created worktree\n")
	fmt.Printf("  Branch: %s\n", color.CyanString(branch))
	fmt.Printf("  Path:   %s\n", targetPath)

	// Run post-add hook
	if cfg.Hooks.PostAdd != "" {
		fmt.Printf("  Running: %s\n", color.YellowString(cfg.Hooks.PostAdd))
		hookCmd := exec.Command("sh", "-c", cfg.Hooks.PostAdd)
		hookCmd.Dir = targetPath
		hookCmd.Stdout = os.Stdout
		hookCmd.Stderr = os.Stderr
		if err := hookCmd.Run(); err != nil {
			color.Yellow("  Warning: post_add hook failed: %v", err)
		}
	}

	fmt.Printf("\n  cd %s\n", targetPath)
	return nil
}
