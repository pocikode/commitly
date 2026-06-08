package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newCommitCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "commit",
		Short: "Generate a commit message from staged changes and commit",
		RunE:  runCommit,
	}
}

// runCommit orchestrates the full commit flow. Stub for now; wired in task 6.0.
func runCommit(cmd *cobra.Command, args []string) error {
	return fmt.Errorf("commit: not implemented yet")
}
