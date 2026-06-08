package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newHookCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "hook",
		Short: "Manage the prepare-commit-msg git hook",
	}

	cmd.AddCommand(
		&cobra.Command{
			Use:   "set",
			Short: "Install the prepare-commit-msg hook",
			RunE: func(cmd *cobra.Command, args []string) error {
				return fmt.Errorf("hook set: not implemented yet")
			},
		},
		&cobra.Command{
			Use:   "unset",
			Short: "Remove the prepare-commit-msg hook",
			RunE: func(cmd *cobra.Command, args []string) error {
				return fmt.Errorf("hook unset: not implemented yet")
			},
		},
	)

	return cmd
}
