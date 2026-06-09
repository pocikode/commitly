package cmd

import (
	"fmt"
	"os"

	"github.com/pocikode/commitly/internal/version"
	"github.com/spf13/cobra"
)

// Global flags shared across commands.
var (
	flagYes    bool
	flagConfig string
)

// newRootCmd builds the root `cly` command tree. It is a function (not a
// package-level var) so tests can construct a fresh command with isolated flag
// state on each run.
func newRootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:           "cly",
		Aliases:       []string{"commitly"},
		Short:         "AI-powered git commit message generator",
		Long:          "cly (Commitly) generates git commit messages from your staged diff using a configurable AI provider.",
		Version:       version.String(),
		SilenceUsage:  true,
		SilenceErrors: true,
		// Bare `cly` with no subcommand defaults to commit.
		RunE: func(cmd *cobra.Command, args []string) error {
			return runCommit(cmd, args)
		},
	}

	root.PersistentFlags().BoolVarP(&flagYes, "yes", "y", false, "skip confirmation prompts (auto-confirm)")
	root.PersistentFlags().StringVar(&flagConfig, "config", "", "path to config file (overrides default and CLY_CONFIG_PATH)")

	root.SetVersionTemplate("cly {{.Version}}\n")

	root.AddCommand(
		newCommitCmd(),
		newConfigCmd(),
		newHookCmd(),
		newModelsCmd(),
		newCommitlintCmd(),
	)

	return root
}

// Execute runs the root command and handles top-level error reporting + exit
// codes. This is the single entry point called by main.
func Execute() {
	if err := newRootCmd().Execute(); err != nil {
		fmt.Fprintln(os.Stderr, "Error: "+err.Error())
		os.Exit(1)
	}
}
