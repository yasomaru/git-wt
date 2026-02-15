package config

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestDefault(t *testing.T) {
	cfg := Default()

	if cfg.Layout.Strategy != LayoutAdjacent {
		t.Errorf("expected layout strategy %q, got %q", LayoutAdjacent, cfg.Layout.Strategy)
	}
	if cfg.Layout.Pattern != "{repo}-{branch}" {
		t.Errorf("expected layout pattern %q, got %q", "{repo}-{branch}", cfg.Layout.Pattern)
	}
	if cfg.Cleanup.StaleDays != 30 {
		t.Errorf("expected stale_days 30, got %d", cfg.Cleanup.StaleDays)
	}
	if cfg.Cleanup.AutoPrune != true {
		t.Error("expected auto_prune true, got false")
	}
	if cfg.Hooks.PostAdd != "" {
		t.Errorf("expected empty post_add hook, got %q", cfg.Hooks.PostAdd)
	}
}

func TestSanitizeBranch(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		expect string
	}{
		{
			name:   "slash separated feature branch",
			input:  "feature/auth",
			expect: "feature-auth",
		},
		{
			name:   "at symbol",
			input:  "user@feature",
			expect: "user-feature",
		},
		{
			name:   "spaces in branch name",
			input:  "my branch name",
			expect: "my-branch-name",
		},
		{
			name:   "square brackets",
			input:  "fix[issue]",
			expect: "fix-issue-",
		},
		{
			name:   "clean branch name unchanged",
			input:  "main",
			expect: "main",
		},
		{
			name:   "multiple special characters",
			input:  "feature/auth@v2:fix~1^2",
			expect: "feature-auth-v2-fix-1-2",
		},
		{
			name:   "backslash",
			input:  "path\\to\\branch",
			expect: "path-to-branch",
		},
		{
			name:   "asterisk and question mark",
			input:  "glob*pattern?",
			expect: "glob-pattern-",
		},
		{
			name:   "tilde and caret",
			input:  "HEAD~3^2",
			expect: "HEAD-3-2",
		},
		{
			name:   "empty string",
			input:  "",
			expect: "",
		},
		{
			name:   "hyphenated branch unchanged",
			input:  "feature-auth-v2",
			expect: "feature-auth-v2",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := sanitizeBranch(tc.input)
			if got != tc.expect {
				t.Errorf("sanitizeBranch(%q) = %q, want %q", tc.input, got, tc.expect)
			}
		})
	}
}

func TestWorktreePath_Adjacent(t *testing.T) {
	t.Run("default pattern", func(t *testing.T) {
		cfg := Default()
		repoRoot := filepath.Join("/home", "user", "projects", "myrepo")
		got := cfg.WorktreePath(repoRoot, "feature/login")

		want := filepath.Join("/home", "user", "projects", "myrepo-feature-login")
		if got != want {
			t.Errorf("WorktreePath() = %q, want %q", got, want)
		}
	})

	t.Run("custom pattern with repo and branch", func(t *testing.T) {
		cfg := Default()
		cfg.Layout.Pattern = "wt-{repo}-{branch}"
		repoRoot := filepath.Join("/home", "user", "projects", "myrepo")
		got := cfg.WorktreePath(repoRoot, "bugfix")

		want := filepath.Join("/home", "user", "projects", "wt-myrepo-bugfix")
		if got != want {
			t.Errorf("WorktreePath() = %q, want %q", got, want)
		}
	})

	t.Run("pattern with only branch", func(t *testing.T) {
		cfg := Default()
		cfg.Layout.Pattern = "worktree-{branch}"
		repoRoot := filepath.Join("/home", "user", "projects", "myrepo")
		got := cfg.WorktreePath(repoRoot, "develop")

		want := filepath.Join("/home", "user", "projects", "worktree-develop")
		if got != want {
			t.Errorf("WorktreePath() = %q, want %q", got, want)
		}
	})
}

func TestWorktreePath_Subdirectory(t *testing.T) {
	cfg := Default()
	cfg.Layout.Strategy = LayoutSubdirectory
	repoRoot := filepath.Join("/home", "user", "projects", "myrepo")
	got := cfg.WorktreePath(repoRoot, "feature/auth")

	want := filepath.Join("/home", "user", "projects", "myrepo", ".worktrees", "feature-auth")
	if got != want {
		t.Errorf("WorktreePath() = %q, want %q", got, want)
	}
}

func TestWorktreePath_EmptyPattern(t *testing.T) {
	cfg := Default()
	cfg.Layout.Pattern = ""
	repoRoot := filepath.Join("/home", "user", "projects", "myrepo")
	got := cfg.WorktreePath(repoRoot, "develop")

	// Empty pattern should fall back to "{repo}-{branch}"
	want := filepath.Join("/home", "user", "projects", "myrepo-develop")
	if got != want {
		t.Errorf("WorktreePath() with empty pattern = %q, want %q", got, want)
	}
}

func TestGenerateDefaultConfig(t *testing.T) {
	output := GenerateDefaultConfig()

	requiredSections := []string{
		"[layout]",
		"[cleanup]",
		"[hooks]",
	}
	for _, section := range requiredSections {
		if !strings.Contains(output, section) {
			t.Errorf("GenerateDefaultConfig() missing section %q", section)
		}
	}

	requiredKeys := []string{
		`strategy = "adjacent"`,
		`pattern = "{repo}-{branch}"`,
		"stale_days = 30",
		"auto_prune = true",
		"post_add",
	}
	for _, key := range requiredKeys {
		if !strings.Contains(output, key) {
			t.Errorf("GenerateDefaultConfig() missing key/value %q", key)
		}
	}
}

func TestInitConfig(t *testing.T) {
	t.Run("creates file at specified path", func(t *testing.T) {
		tmpDir := t.TempDir()
		cfgPath := filepath.Join(tmpDir, "config.toml")

		err := InitConfig(cfgPath)
		if err != nil {
			t.Fatalf("InitConfig() error = %v", err)
		}

		data, err := os.ReadFile(cfgPath)
		if err != nil {
			t.Fatalf("failed to read created config: %v", err)
		}

		content := string(data)
		if !strings.Contains(content, "[layout]") {
			t.Error("created config file missing [layout] section")
		}
	})

	t.Run("creates parent directories", func(t *testing.T) {
		tmpDir := t.TempDir()
		cfgPath := filepath.Join(tmpDir, "nested", "deep", "config.toml")

		err := InitConfig(cfgPath)
		if err != nil {
			t.Fatalf("InitConfig() error = %v", err)
		}

		if _, err := os.Stat(cfgPath); os.IsNotExist(err) {
			t.Error("InitConfig() did not create file at nested path")
		}
	})
}

func TestInitConfig_ExistingFile(t *testing.T) {
	tmpDir := t.TempDir()
	cfgPath := filepath.Join(tmpDir, "config.toml")

	// Create the file first with custom content.
	originalContent := []byte("# existing config\n")
	if err := os.WriteFile(cfgPath, originalContent, 0o644); err != nil {
		t.Fatalf("failed to create existing file: %v", err)
	}

	// InitConfig overwrites without checking existence.
	err := InitConfig(cfgPath)
	if err != nil {
		t.Fatalf("InitConfig() on existing file error = %v", err)
	}

	data, err := os.ReadFile(cfgPath)
	if err != nil {
		t.Fatalf("failed to read config after overwrite: %v", err)
	}

	// The file should now contain the default config, not the original content.
	content := string(data)
	if strings.Contains(content, "# existing config") {
		t.Error("InitConfig() did not overwrite existing file content")
	}
	if !strings.Contains(content, "[layout]") {
		t.Error("InitConfig() overwritten file missing [layout] section")
	}
}

func TestLoadForRepo_WithLocalConfig(t *testing.T) {
	tmpDir := t.TempDir()

	configContent := `
[layout]
strategy = "subdirectory"
pattern = "wt-{branch}"

[cleanup]
stale_days = 14
auto_prune = false

[hooks]
post_add = "make setup"
`
	cfgPath := filepath.Join(tmpDir, ".git-wt.toml")
	if err := os.WriteFile(cfgPath, []byte(configContent), 0o644); err != nil {
		t.Fatalf("failed to write test config: %v", err)
	}

	cfg := LoadForRepo(tmpDir)

	if cfg.Layout.Strategy != LayoutSubdirectory {
		t.Errorf("expected strategy %q, got %q", LayoutSubdirectory, cfg.Layout.Strategy)
	}
	if cfg.Layout.Pattern != "wt-{branch}" {
		t.Errorf("expected pattern %q, got %q", "wt-{branch}", cfg.Layout.Pattern)
	}
	if cfg.Cleanup.StaleDays != 14 {
		t.Errorf("expected stale_days 14, got %d", cfg.Cleanup.StaleDays)
	}
	if cfg.Cleanup.AutoPrune != false {
		t.Error("expected auto_prune false, got true")
	}
	if cfg.Hooks.PostAdd != "make setup" {
		t.Errorf("expected post_add %q, got %q", "make setup", cfg.Hooks.PostAdd)
	}
}

func TestLoadForRepo_WithoutLocalConfig(t *testing.T) {
	tmpDir := t.TempDir()
	// No .git-wt.toml created -- should return defaults.

	cfg := LoadForRepo(tmpDir)

	defaults := Default()
	if cfg.Layout.Strategy != defaults.Layout.Strategy {
		t.Errorf("expected default strategy %q, got %q", defaults.Layout.Strategy, cfg.Layout.Strategy)
	}
	if cfg.Layout.Pattern != defaults.Layout.Pattern {
		t.Errorf("expected default pattern %q, got %q", defaults.Layout.Pattern, cfg.Layout.Pattern)
	}
	if cfg.Cleanup.StaleDays != defaults.Cleanup.StaleDays {
		t.Errorf("expected default stale_days %d, got %d", defaults.Cleanup.StaleDays, cfg.Cleanup.StaleDays)
	}
}

func TestLoadForRepo_InvalidTOML(t *testing.T) {
	tmpDir := t.TempDir()

	invalidContent := `
[layout
strategy = adjacent"
this is not valid toml at all !!!
`
	cfgPath := filepath.Join(tmpDir, ".git-wt.toml")
	if err := os.WriteFile(cfgPath, []byte(invalidContent), 0o644); err != nil {
		t.Fatalf("failed to write invalid config: %v", err)
	}

	// Must not panic -- should fall back to defaults.
	cfg := LoadForRepo(tmpDir)

	if cfg == nil {
		t.Fatal("LoadForRepo() returned nil for invalid TOML")
	}

	// With invalid TOML, the defaults should remain intact.
	defaults := Default()
	if cfg.Layout.Strategy != defaults.Layout.Strategy {
		t.Errorf("expected default strategy %q after invalid TOML, got %q",
			defaults.Layout.Strategy, cfg.Layout.Strategy)
	}
	if cfg.Cleanup.StaleDays != defaults.Cleanup.StaleDays {
		t.Errorf("expected default stale_days %d after invalid TOML, got %d",
			defaults.Cleanup.StaleDays, cfg.Cleanup.StaleDays)
	}
}

func TestLoadForRepo_PartialConfig(t *testing.T) {
	tmpDir := t.TempDir()

	// Only override layout strategy; everything else should remain default.
	partialContent := `
[layout]
strategy = "subdirectory"
`
	cfgPath := filepath.Join(tmpDir, ".git-wt.toml")
	if err := os.WriteFile(cfgPath, []byte(partialContent), 0o644); err != nil {
		t.Fatalf("failed to write partial config: %v", err)
	}

	cfg := LoadForRepo(tmpDir)

	if cfg.Layout.Strategy != LayoutSubdirectory {
		t.Errorf("expected strategy %q, got %q", LayoutSubdirectory, cfg.Layout.Strategy)
	}
	// Pattern should retain the default since it was not overridden.
	if cfg.Layout.Pattern != "{repo}-{branch}" {
		t.Errorf("expected default pattern %q, got %q", "{repo}-{branch}", cfg.Layout.Pattern)
	}
	// Cleanup should retain defaults.
	if cfg.Cleanup.StaleDays != 30 {
		t.Errorf("expected default stale_days 30, got %d", cfg.Cleanup.StaleDays)
	}
	if cfg.Cleanup.AutoPrune != true {
		t.Error("expected default auto_prune true, got false")
	}
}

func TestLayoutStrategyConstants(t *testing.T) {
	if LayoutAdjacent != "adjacent" {
		t.Errorf("LayoutAdjacent = %q, want %q", LayoutAdjacent, "adjacent")
	}
	if LayoutSubdirectory != "subdirectory" {
		t.Errorf("LayoutSubdirectory = %q, want %q", LayoutSubdirectory, "subdirectory")
	}
}

func TestWorktreePath_SubdirectoryIgnoresPattern(t *testing.T) {
	cfg := Default()
	cfg.Layout.Strategy = LayoutSubdirectory
	cfg.Layout.Pattern = "custom-{repo}-{branch}"

	repoRoot := filepath.Join("/home", "user", "projects", "myrepo")
	got := cfg.WorktreePath(repoRoot, "develop")

	// Subdirectory strategy should ignore the pattern entirely.
	want := filepath.Join("/home", "user", "projects", "myrepo", ".worktrees", "develop")
	if got != want {
		t.Errorf("WorktreePath() subdirectory with custom pattern = %q, want %q", got, want)
	}
}

func TestInitConfig_FilePermissions(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Unix file permissions not applicable on Windows")
	}

	tmpDir := t.TempDir()
	cfgPath := filepath.Join(tmpDir, "config.toml")

	if err := InitConfig(cfgPath); err != nil {
		t.Fatalf("InitConfig() error = %v", err)
	}

	info, err := os.Stat(cfgPath)
	if err != nil {
		t.Fatalf("failed to stat config file: %v", err)
	}

	// WriteFile is called with 0o644; verify the file is not world-writable.
	perm := info.Mode().Perm()
	if perm&0o002 != 0 {
		t.Errorf("config file is world-writable: %o", perm)
	}
}
