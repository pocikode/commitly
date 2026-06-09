package cmd

import (
	"fmt"
	"io"

	"github.com/pocikode/commitly/internal/config"
	"github.com/pocikode/commitly/internal/prompt"
	"github.com/spf13/cobra"
)

func newCommitlintCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "commitlint",
		Short: "Show / enable commitlint rule injection for generated messages",
		Long: "Displays the simplified commitlint rules injected into the prompt. " +
			"Enable injection by setting prompt_module to @commitlint:\n\n  cly config set prompt_module=@commitlint",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runCommitlint(cmd.OutOrStdout(), flagConfig)
		},
	}
}

// runCommitlint prints the commitlint rules and whether injection is active.
func runCommitlint(out io.Writer, configPath string) error {
	cfg, _, err := config.Resolve(config.Options{ConfigPath: configPath})
	if err != nil {
		return err
	}

	active := cfg.PromptModule == "@commitlint"
	fmt.Fprintf(out, "commitlint injection: %s\n\n", enabledLabel(active))
	fmt.Fprintln(out, prompt.CommitlintRules())
	if !active {
		fmt.Fprintln(out, "\nEnable with: cly config set prompt_module=@commitlint")
	}
	return nil
}

func enabledLabel(b bool) string {
	if b {
		return "enabled"
	}
	return "disabled"
}
