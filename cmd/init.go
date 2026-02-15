package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/fatih/color"
	"github.com/spf13/cobra"

	"github.com/yasomaru/git-wt/internal/config"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Create a default configuration file",
	Long: `Generate a default git-wt configuration file.

By default creates ~/.config/git-wt/config.toml.
Use --local to create .git-wt.toml in the current directory.`,
	RunE: runInit,
}

var initLocal bool

func init() {
	initCmd.Flags().BoolVar(&initLocal, "local", false, "create config in current directory")
	rootCmd.AddCommand(initCmd)
}

func runInit(cmd *cobra.Command, args []string) error {
	var path string

	if initLocal {
		cwd, err := os.Getwd()
		if err != nil {
			return err
		}
		path = filepath.Join(cwd, ".git-wt.toml")
	} else {
		home, err := os.UserHomeDir()
		if err != nil {
			return err
		}
		path = filepath.Join(home, ".config", "git-wt", "config.toml")
	}

	if _, err := os.Stat(path); err == nil {
		return fmt.Errorf("config already exists: %s", path)
	}

	if err := config.InitConfig(path); err != nil {
		return err
	}

	color.Green("  Created config: %s", path)
	return nil
}
