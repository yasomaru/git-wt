# git-wt

A smarter way to manage git worktrees.

`git-wt` wraps git's built-in worktree support with sensible defaults,
rich status output, an interactive TUI for cleanup, and configurable
layout strategies -- so you can juggle multiple branches without juggling
multiple terminal windows.

## Features

- Smart worktree creation with automatic path resolution and branch management
- Rich status display with modified/untracked counts, sync info, and merge status
- Interactive TUI for multi-select cleanup
- Configurable layout strategies (adjacent or subdirectory)
- Post-add hooks for automation (e.g., `npm install`)

## Installation

### Homebrew

```sh
brew install git-wt/tap/git-wt
```

### From source

```sh
go install github.com/yasomaru/git-wt@latest
```

### Binary releases

Download pre-built binaries for Linux, macOS, and Windows from the
[GitHub Releases](https://github.com/yasomaru/git-wt/releases) page.

## Quick Start

```sh
# Open the interactive worktree manager
git wt

# Create a new worktree for a feature branch
git wt add feature-auth

# Create a worktree branching off main
git wt add feature-auth -b main

# List all worktrees with status information
git wt ls

# Clean up merged or stale worktrees
git wt clean
git wt clean --merged
git wt clean --stale 30
git wt clean --dry-run

# Initialize configuration
git wt init
git wt init --local

# Show version information
git wt version
```

## Configuration

`git-wt` reads configuration from two TOML files. The local file takes
precedence over the global file so you can override settings per repository.

| Scope  | Path                              |
|--------|-----------------------------------|
| Global | `~/.config/git-wt/config.toml`    |
| Local  | `.git-wt.toml` (repository root)  |

Run `git wt init` to generate a global config file, or `git wt init --local`
to generate a local one.

### Default configuration

```toml
[layout]
# "adjacent" places worktrees next to the main repo (../repo-branch/).
# "subdirectory" places them inside the repo (.worktrees/branch/).
strategy = "adjacent"

# Naming pattern for worktree directories.
# Available variables: {repo}, {branch}
pattern = "{repo}-{branch}"

[cleanup]
# Number of days of inactivity before a worktree is considered stale.
stale_days = 30

# Automatically prune stale remote-tracking references during cleanup.
auto_prune = true

[hooks]
# Command executed after a new worktree is created.
# Example: "npm install" or "make deps"
post_add = ""
```

### Configuration reference

| Key                  | Type    | Default              | Description                                         |
|----------------------|---------|----------------------|-----------------------------------------------------|
| `layout.strategy`    | string  | `"adjacent"`         | `"adjacent"` or `"subdirectory"`                     |
| `layout.pattern`     | string  | `"{repo}-{branch}"`  | Directory name pattern with `{repo}` and `{branch}` |
| `cleanup.stale_days` | integer | `30`                 | Days of inactivity before a worktree is stale        |
| `cleanup.auto_prune` | boolean | `true`               | Prune stale remote refs on cleanup                   |
| `hooks.post_add`     | string  | `""`                 | Shell command to run after `git wt add`              |

## TUI Keybindings

The interactive TUI (launched with `git wt` or `git wt clean`) supports the
following keys.

| Key              | Action                      |
|------------------|-----------------------------|
| `Up` / `k`       | Move cursor up              |
| `Down` / `j`     | Move cursor down            |
| `Space` / `x`    | Toggle selection            |
| `a`              | Select all merged worktrees |
| `n`              | Deselect all                |
| `d` / `Enter`    | Confirm deletion            |
| `q` / `Esc`      | Quit without changes        |

## License

This project is licensed under the [MIT License](LICENSE).
