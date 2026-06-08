package cmd

import (
	"context"
	"fmt"
	"io"

	"github.com/charmbracelet/huh"
	"github.com/pocikode/opencommit/internal/config"
	"github.com/pocikode/opencommit/internal/provider"
	"github.com/spf13/cobra"
)

func newModelsCmd() *cobra.Command {
	var set string
	cmd := &cobra.Command{
		Use:   "models",
		Short: "List and select models for the active provider",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runModels(cmd.Context(), cmd.OutOrStdout(), flagConfig, flagYes, set)
		},
	}
	cmd.Flags().StringVar(&set, "set", "", "set the model non-interactively and persist it")
	return cmd
}

// runModels lists the provider's models and (interactively or via --set)
// persists a selection to config.
func runModels(ctx context.Context, out io.Writer, configPath string, noPrompt bool, set string) error {
	if ctx == nil {
		ctx = context.Background()
	}
	cfg, _, err := config.Resolve(config.Options{ConfigPath: configPath})
	if err != nil {
		return err
	}

	models := provider.ListModels(ctx, cfg)
	if len(models) == 0 {
		return fmt.Errorf("no models available for provider %q; set one with: oco config set model=<name>", cfg.AIProvider)
	}

	// Non-interactive set path.
	if set != "" {
		if !contains(models, set) {
			fmt.Fprintf(out, "warning: %q is not in the known model list\n", set)
		}
		return persistModel(configPath, set, out)
	}

	// List.
	fmt.Fprintf(out, "Models for %s (current: %s):\n", cfg.AIProvider, cfg.Model)
	for _, m := range models {
		marker := "  "
		if m == cfg.Model {
			marker = "* "
		}
		fmt.Fprintf(out, "%s%s\n", marker, m)
	}

	if noPrompt {
		return nil
	}

	// Interactive select.
	choice := cfg.Model
	opts := make([]huh.Option[string], 0, len(models))
	for _, m := range models {
		opts = append(opts, huh.NewOption(m, m))
	}
	if err := huh.NewSelect[string]().Title("Select default model").Options(opts...).Value(&choice).Run(); err != nil {
		return nil // aborted; leave config unchanged
	}
	if choice == cfg.Model {
		return nil
	}
	return persistModel(configPath, choice, out)
}

// persistModel writes the chosen model to the global config file.
func persistModel(configPath, model string, out io.Writer) error {
	path, err := config.GlobalPath(configPath)
	if err != nil {
		return err
	}
	cfg, err := config.Load(path)
	if err != nil {
		return err
	}
	if err := config.Set(&cfg, "model", model); err != nil {
		return err
	}
	if err := config.Save(cfg, path); err != nil {
		return err
	}
	fmt.Fprintf(out, "Model set to %s\n", model)
	return nil
}

func contains(s []string, v string) bool {
	for _, x := range s {
		if x == v {
			return true
		}
	}
	return false
}
