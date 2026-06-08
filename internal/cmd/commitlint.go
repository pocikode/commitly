package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newCommitlintCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "commitlint",
		Short: "Align generated messages with the project's commitlint rules",
		RunE: func(cmd *cobra.Command, args []string) error {
			return fmt.Errorf("commitlint: not implemented yet")
		},
	}
}
