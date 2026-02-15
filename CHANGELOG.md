# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/),
and this project adheres to [Semantic Versioning](https://semver.org/).

## [1.0.0] - 2025-XX-XX

### Added

- `git wt add` -- Create worktrees with smart path resolution and branch
  management.
- `git wt ls` -- List worktrees with status, sync info, merge status, and
  stale indicators.
- `git wt clean` -- Remove merged or stale worktrees with `--merged`,
  `--stale`, `--dry-run`, and `--force` flags.
- `git wt init` -- Generate default configuration (global or local with
  `--local`).
- `git wt version` -- Display version information.
- Interactive TUI for multi-select worktree management.
- Configurable layout strategies: `adjacent` and `subdirectory`.
- Post-add hook support for automation (e.g., `npm install`).
- TOML-based configuration with global and local config cascade.

### Fixed

- Panic on detached HEAD with short SHA.
- Silent TOML parse error handling (now warns to stderr).
- `clean` command incorrectly marking all worktrees as stale when
  `staleDays=0`.
- `sanitizeBranch` not handling `@`, `*`, `?`, `[]`, `~`, `^`, and space
  characters.
- Config loading using the current working directory instead of the repository
  root for local config resolution.
