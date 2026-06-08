package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newModelsCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "models",
		Short: "List and select models for the active provider",
		RunE: func(cmd *cobra.Command, args []string) error {
			return fmt.Errorf("models: not implemented yet")
		},
	}
}
