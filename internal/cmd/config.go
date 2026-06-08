package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newConfigCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Get or set configuration values",
		RunE: func(cmd *cobra.Command, args []string) error {
			// No-args invocation runs the first-run setup wizard (task 2.9).
			return fmt.Errorf("config: setup wizard not implemented yet")
		},
	}

	cmd.AddCommand(
		&cobra.Command{
			Use:   "get <KEY>",
			Short: "Print a configuration value",
			Args:  cobra.ExactArgs(1),
			RunE: func(cmd *cobra.Command, args []string) error {
				return fmt.Errorf("config get: not implemented yet")
			},
		},
		&cobra.Command{
			Use:   "set <KEY>=<VALUE> ...",
			Short: "Set one or more configuration values",
			Args:  cobra.MinimumNArgs(1),
			RunE: func(cmd *cobra.Command, args []string) error {
				return fmt.Errorf("config set: not implemented yet")
			},
		},
	)

	return cmd
}
