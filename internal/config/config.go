package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/BurntSushi/toml"
)

type LayoutStrategy string

const (
	LayoutAdjacent     LayoutStrategy = "adjacent"
	LayoutSubdirectory LayoutStrategy = "subdirectory"
)

type Config struct {
	Layout  LayoutConfig  `toml:"layout"`
	Cleanup CleanupConfig `toml:"cleanup"`
	Hooks   HooksConfig   `toml:"hooks"`
}

type LayoutConfig struct {
	Strategy LayoutStrategy `toml:"strategy"`
	Pattern  string         `toml:"pattern"`
}

type CleanupConfig struct {
	StaleDays int  `toml:"stale_days"`
	AutoPrune bool `toml:"auto_prune"`
}

type HooksConfig struct {
	PostAdd string `toml:"post_add"`
}

func Default() *Config {
	return &Config{
		Layout: LayoutConfig{
			Strategy: LayoutAdjacent,
			Pattern:  "{repo}-{branch}",
		},
		Cleanup: CleanupConfig{
			StaleDays: 30,
			AutoPrune: true,
		},
	}
}

// Load returns the config using cwd for local config lookup.
// Prefer LoadForRepo when the repo root is known.
func Load() *Config {
	return LoadForRepo("")
}

// LoadForRepo loads config with local config resolved from repoRoot.
// If repoRoot is empty, falls back to os.Getwd().
func LoadForRepo(repoRoot string) *Config {
	cfg := Default()

	// Global config: ~/.config/git-wt/config.toml
	if home, err := os.UserHomeDir(); err == nil {
		globalPath := filepath.Join(home, ".config", "git-wt", "config.toml")
		loadFile(globalPath, cfg)
	}

	// Local config: .git-wt.toml in repo root
	localDir := repoRoot
	if localDir == "" {
		localDir, _ = os.Getwd()
	}
	if localDir != "" {
		localPath := filepath.Join(localDir, ".git-wt.toml")
		loadFile(localPath, cfg)
	}

	return cfg
}

func loadFile(path string, cfg *Config) {
	if _, err := os.Stat(path); err != nil {
		return
	}
	if _, err := toml.DecodeFile(path, cfg); err != nil {
		fmt.Fprintf(os.Stderr, "warning: failed to parse config %s: %v\n", path, err)
	}
}

// WorktreePath computes the target path for a new worktree.
func (c *Config) WorktreePath(repoRoot, branch string) string {
	repoName := filepath.Base(repoRoot)
	// Sanitize branch name for filesystem
	safeBranch := sanitizeBranch(branch)

	switch c.Layout.Strategy {
	case LayoutSubdirectory:
		return filepath.Join(repoRoot, ".worktrees", safeBranch)
	default: // adjacent
		pattern := c.Layout.Pattern
		if pattern == "" {
			pattern = "{repo}-{branch}"
		}
		dirName := strings.ReplaceAll(pattern, "{repo}", repoName)
		dirName = strings.ReplaceAll(dirName, "{branch}", safeBranch)
		return filepath.Join(filepath.Dir(repoRoot), dirName)
	}
}

func sanitizeBranch(branch string) string {
	r := strings.NewReplacer(
		"/", "-",
		"\\", "-",
		":", "-",
		"@", "-",
		"*", "-",
		"?", "-",
		"[", "-",
		"]", "-",
		"~", "-",
		"^", "-",
		" ", "-",
	)
	return r.Replace(branch)
}

// GenerateDefaultConfig returns the default config as TOML string.
func GenerateDefaultConfig() string {
	return `# git-wt configuration

[layout]
# "adjacent" places worktrees next to the repo: ../repo-branch/
# "subdirectory" places them inside: .worktrees/branch/
strategy = "adjacent"

# Directory naming pattern. Available variables: {repo}, {branch}
pattern = "{repo}-{branch}"

[cleanup]
# Days of inactivity before a worktree is considered stale
stale_days = 30

# Automatically prune stale worktree references
auto_prune = true

[hooks]
# Command to run after creating a new worktree
# post_add = "npm install"
`
}

// InitConfig creates a default config file.
func InitConfig(path string) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create config directory: %w", err)
	}
	return os.WriteFile(path, []byte(GenerateDefaultConfig()), 0o644)
}
