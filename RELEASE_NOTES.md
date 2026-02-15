## git-wt v1.0.0

A smarter way to manage git worktrees.

### Highlights

- **Interactive TUI** -- Select and remove worktrees visually with keyboard navigation
- **Smart worktree creation** -- Automatic path resolution, branch management, and post-add hooks
- **Rich status display** -- See modified files, sync status, merge state, and stale indicators at a glance
- **Easy cleanup** -- Remove merged or stale worktrees with a single command
- **Flexible configuration** -- TOML-based config with global/local cascade and layout strategies

### Commands

| Command | Description |
|---------|-------------|
| `git wt` | Open interactive TUI |
| `git wt add <branch>` | Create a worktree with smart defaults |
| `git wt ls` | List all worktrees with status |
| `git wt clean` | Remove merged or stale worktrees |
| `git wt init` | Generate default configuration |
| `git wt version` | Print version information |

### Installation

**Homebrew** (coming soon)
```
brew install yasomaru/tap/git-wt
```

**Go**
```
go install github.com/yasomaru/git-wt@v1.0.0
```

**Binary**

Download from the assets below for your platform.

### Supported Platforms

| OS | Architecture |
|----|-------------|
| Linux | amd64, arm64 |
| macOS | amd64 (Intel), arm64 (Apple Silicon) |
| Windows | amd64, arm64 |

**Full Changelog**: https://github.com/yasomaru/git-wt/blob/main/CHANGELOG.md
