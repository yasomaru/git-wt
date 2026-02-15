## What's New

### `git wt switch` command

Quickly switch between worktrees by branch name without manually typing paths.

```sh
git wt switch feature-auth   # exact match
git wt switch feat           # prefix/substring match
git wt switch                # interactive selector
```

**Matching priority**: exact > prefix > substring (all case-insensitive).
When multiple worktrees match, an interactive TUI selector is shown.

### Shell integration

Set up the `wt` shell function for automatic `cd`:

```sh
# zsh
eval "$(git wt switch --init zsh)"

# bash
eval "$(git wt switch --init bash)"

# fish
git wt switch --init fish | source
```

Then use `wt switch <branch>` (or `wt sw <branch>`) to switch and cd in one step.

## Internal improvements

- Extracted shared `BuildTags` helper for consistent tag rendering across TUIs
- New single-select TUI component (`selector`) for worktree selection
