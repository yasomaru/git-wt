package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/yasomaru/git-wt/internal/git"
	"github.com/yasomaru/git-wt/internal/tui"
)

var switchCmd = &cobra.Command{
	Use:     "switch [branch]",
	Aliases: []string{"sw"},
	Short:   "Print the path of a worktree to switch to",
	Long: `Print the worktree path matching the given branch name.

Use this with cd or a shell wrapper to quickly switch between worktrees.
Without arguments, an interactive selector is shown.

Matching priority:
  1. Exact match
  2. Prefix match
  3. Substring match (case-insensitive)

Shell integration:
  eval "$(git wt switch --init zsh)"   # add to .zshrc
  eval "$(git wt switch --init bash)"  # add to .bashrc
  git wt switch --init fish | source   # add to config.fish`,
	Args: cobra.MaximumNArgs(1),
	Example: `  git wt switch feature-auth
  git wt switch feat
  git wt switch
  git wt switch --init zsh`,
	RunE: runSwitch,
}

var switchInitShell string

func init() {
	switchCmd.Flags().StringVar(&switchInitShell, "init", "", "output shell integration function (bash, zsh, fish)")
	rootCmd.AddCommand(switchCmd)
}

func runSwitch(cmd *cobra.Command, args []string) error {
	if switchInitShell != "" {
		return printShellInit(switchInitShell)
	}

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

	// Filter out bare and detached worktrees
	var candidates []git.Worktree
	for _, wt := range worktrees {
		if wt.IsBare || wt.IsDetached {
			continue
		}
		candidates = append(candidates, wt)
	}

	if len(candidates) == 0 {
		return fmt.Errorf("no worktrees available")
	}

	// No argument: launch interactive selector
	if len(args) == 0 {
		selected, err := tui.RunSelector(candidates)
		if err != nil {
			return err
		}
		if selected == nil {
			return fmt.Errorf("cancelled")
		}
		fmt.Println(selected.Path)
		return nil
	}

	// With argument: match by branch name
	query := args[0]
	matches := matchWorktrees(candidates, query)

	switch len(matches) {
	case 0:
		return fmt.Errorf("no worktree matching %q", query)
	case 1:
		fmt.Println(matches[0].Path)
		return nil
	default:
		// Multiple matches: launch selector with filtered list
		fmt.Fprintf(os.Stderr, "Multiple worktrees match %q:\n", query)
		selected, err := tui.RunSelector(matches)
		if err != nil {
			return err
		}
		if selected == nil {
			return fmt.Errorf("cancelled")
		}
		fmt.Println(selected.Path)
		return nil
	}
}

// matchWorktrees returns worktrees matching the query with the following
// priority: exact match > prefix match > substring match.
// All comparisons are case-insensitive.
func matchWorktrees(worktrees []git.Worktree, query string) []git.Worktree {
	queryLower := strings.ToLower(query)

	// 1. Exact match (case-insensitive)
	for _, wt := range worktrees {
		if strings.EqualFold(wt.BranchShort(), query) {
			return []git.Worktree{wt}
		}
	}

	// 2. Prefix match (case-insensitive)
	var prefixMatches []git.Worktree
	for _, wt := range worktrees {
		if strings.HasPrefix(strings.ToLower(wt.BranchShort()), queryLower) {
			prefixMatches = append(prefixMatches, wt)
		}
	}
	if len(prefixMatches) > 0 {
		return prefixMatches
	}

	// 3. Substring match (case-insensitive)
	var substringMatches []git.Worktree
	for _, wt := range worktrees {
		if strings.Contains(strings.ToLower(wt.BranchShort()), queryLower) {
			substringMatches = append(substringMatches, wt)
		}
	}
	return substringMatches
}

func printShellInit(shell string) error {
	switch strings.ToLower(shell) {
	case "bash", "zsh":
		fmt.Print(shellInitBashZsh)
	case "fish":
		fmt.Print(shellInitFish)
	default:
		return fmt.Errorf("unsupported shell: %s (supported: bash, zsh, fish)", shell)
	}
	return nil
}

const shellInitBashZsh = `wt() {
  if [ "$1" = "switch" ] || [ "$1" = "sw" ]; then
    shift
    local dir
    dir=$(command git wt switch "$@")
    if [ $? -eq 0 ] && [ -n "$dir" ] && [ -d "$dir" ]; then
      cd "$dir"
    fi
  else
    command git wt "$@"
  fi
}
`

const shellInitFish = `function wt
  if test (count $argv) -ge 1; and test "$argv[1]" = "switch" -o "$argv[1]" = "sw"
    set -l dir (command git wt switch $argv[2..])
    if test $status -eq 0; and test -n "$dir"; and test -d "$dir"
      cd "$dir"
    end
  else
    command git wt $argv
  end
end
`
