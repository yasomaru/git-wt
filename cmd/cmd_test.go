package cmd

import (
	"bytes"
	"testing"
)

func TestVersionCommand(t *testing.T) {
	SetVersionInfo("1.2.3", "abc123", "2025-01-15")

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	rootCmd.SetArgs([]string{"version"})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("version command returned error: %v", err)
	}

	// versionCmd uses fmt.Printf which writes to os.Stdout, not cmd.OutOrStdout().
	// We verify the command executed without error and that SetVersionInfo updated
	// the package-level variables correctly.
	if appVersion != "1.2.3" {
		t.Errorf("expected appVersion = %q, got %q", "1.2.3", appVersion)
	}
	if appCommit != "abc123" {
		t.Errorf("expected appCommit = %q, got %q", "abc123", appCommit)
	}
	if appDate != "2025-01-15" {
		t.Errorf("expected appDate = %q, got %q", "2025-01-15", appDate)
	}
}

func TestAddCommandRequiresArgs(t *testing.T) {
	rootCmd.SetArgs([]string{"add"})
	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("expected error when running 'add' without arguments, got nil")
	}
}

func TestAddCommandBaseFlag(t *testing.T) {
	f := addCmd.Flags().Lookup("base")
	if f == nil {
		t.Fatal("--base flag not registered on add command")
	}
	if f.Shorthand != "b" {
		t.Errorf("expected --base shorthand = %q, got %q", "b", f.Shorthand)
	}
	if f.DefValue != "" {
		t.Errorf("expected --base default = %q, got %q", "", f.DefValue)
	}
}

func TestCleanCommandFlags(t *testing.T) {
	flags := []struct {
		name      string
		shorthand string
		defValue  string
	}{
		{"merged", "", "false"},
		{"stale", "", "0"},
		{"dry-run", "", "false"},
		{"force", "f", "false"},
	}

	for _, tc := range flags {
		t.Run(tc.name, func(t *testing.T) {
			f := cleanCmd.Flags().Lookup(tc.name)
			if f == nil {
				t.Fatalf("--%s flag not registered on clean command", tc.name)
			}
			if f.Shorthand != tc.shorthand {
				t.Errorf("--%s shorthand: expected %q, got %q", tc.name, tc.shorthand, f.Shorthand)
			}
			if f.DefValue != tc.defValue {
				t.Errorf("--%s default: expected %q, got %q", tc.name, tc.defValue, f.DefValue)
			}
		})
	}
}

func TestInitCommandFlags(t *testing.T) {
	f := initCmd.Flags().Lookup("local")
	if f == nil {
		t.Fatal("--local flag not registered on init command")
	}
	if f.DefValue != "false" {
		t.Errorf("expected --local default = %q, got %q", "false", f.DefValue)
	}
}

func TestLsAliases(t *testing.T) {
	aliases := lsCmd.Aliases
	if len(aliases) == 0 {
		t.Fatal("ls command has no aliases")
	}

	found := false
	for _, a := range aliases {
		if a == "list" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected 'list' in ls aliases, got %v", aliases)
	}
}

func TestRootCommandHelp(t *testing.T) {
	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	rootCmd.SetArgs([]string{"--help"})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("--help returned error: %v", err)
	}

	output := buf.String()
	if output == "" {
		t.Error("expected help output, got empty string")
	}
}
