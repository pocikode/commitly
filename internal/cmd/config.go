package cmd

import (
	"bufio"
	"fmt"
	"io"
	"strings"

	"github.com/pocikode/opencommit/internal/config"
	"github.com/spf13/cobra"
)

func newConfigCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Get or set configuration values",
		Long:  "Manage OpenCommit-Go configuration. With no subcommand, runs the interactive setup wizard.",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runSetupWizard(cmd.InOrStdin(), cmd.OutOrStdout(), flagConfig)
		},
	}

	cmd.AddCommand(
		&cobra.Command{
			Use:   "get <KEY> [<KEY> ...]",
			Short: "Print one or more configuration values",
			Args:  cobra.MinimumNArgs(1),
			RunE: func(cmd *cobra.Command, args []string) error {
				return runConfigGet(cmd.OutOrStdout(), flagConfig, args)
			},
		},
		&cobra.Command{
			Use:   "set <KEY>=<VALUE> [<KEY>=<VALUE> ...]",
			Short: "Set one or more configuration values",
			Args:  cobra.MinimumNArgs(1),
			RunE: func(cmd *cobra.Command, args []string) error {
				return runConfigSet(cmd.OutOrStdout(), flagConfig, args)
			},
		},
	)

	return cmd
}

// runConfigGet prints the effective value for each requested key. The api_key
// value is redacted.
func runConfigGet(out io.Writer, configPath string, keys []string) error {
	cfg, _, err := config.Resolve(config.Options{ConfigPath: configPath})
	if err != nil {
		return err
	}
	for _, key := range keys {
		val, err := config.Get(&cfg, key)
		if err != nil {
			return err
		}
		if key == "api_key" {
			val = redact(val)
		}
		fmt.Fprintf(out, "%s=%s\n", key, val)
	}
	return nil
}

// runConfigSet parses KEY=VALUE pairs, validates each, and persists them to the
// global config file.
func runConfigSet(out io.Writer, configPath string, pairs []string) error {
	path, err := config.GlobalPath(configPath)
	if err != nil {
		return err
	}
	cfg, err := config.Load(path)
	if err != nil {
		return err
	}

	for _, pair := range pairs {
		key, value, ok := strings.Cut(pair, "=")
		if !ok {
			return fmt.Errorf("invalid argument %q: expected KEY=VALUE", pair)
		}
		key = strings.TrimSpace(key)
		if err := config.Set(&cfg, key, value); err != nil {
			return err
		}
	}

	if err := config.Save(cfg, path); err != nil {
		return err
	}
	fmt.Fprintf(out, "Saved %d value(s) to %s\n", len(pairs), path)
	return nil
}

// runSetupWizard walks the user through provider selection and key/model entry,
// then persists the result. Reads from in, writes prompts to out.
func runSetupWizard(in io.Reader, out io.Writer, configPath string) error {
	path, err := config.GlobalPath(configPath)
	if err != nil {
		return err
	}
	cfg, err := config.Load(path)
	if err != nil {
		return err
	}

	r := bufio.NewReader(in)

	fmt.Fprintf(out, "OpenCommit-Go setup\n")
	fmt.Fprintf(out, "Providers: %s\n", strings.Join(config.KnownProviders, ", "))

	provider := ask(r, out, "AI provider", cfg.AIProvider)
	if err := config.Set(&cfg, "ai_provider", provider); err != nil {
		return err
	}

	ptype := ask(r, out, "Provider type ("+config.ProviderTypeOpenAICompatible+"|"+config.ProviderTypeAnthropicCompatible+")", cfg.ProviderType)
	if err := config.Set(&cfg, "provider_type", ptype); err != nil {
		return err
	}

	apiURL := ask(r, out, "API base URL", cfg.APIURL)
	if err := config.Set(&cfg, "api_url", apiURL); err != nil {
		return err
	}

	apiKey := ask(r, out, "API key", cfg.APIKey)
	if err := config.Set(&cfg, "api_key", apiKey); err != nil {
		return err
	}

	model := ask(r, out, "Model", cfg.Model)
	if err := config.Set(&cfg, "model", model); err != nil {
		return err
	}

	if err := config.Save(cfg, path); err != nil {
		return err
	}
	fmt.Fprintf(out, "Configuration saved to %s\n", path)
	return nil
}

// ask prompts for a value, showing the current default, and returns the
// trimmed input or the default when the user just presses Enter.
func ask(r *bufio.Reader, out io.Writer, label, def string) string {
	if def != "" {
		fmt.Fprintf(out, "%s [%s]: ", label, def)
	} else {
		fmt.Fprintf(out, "%s: ", label)
	}
	line, _ := r.ReadString('\n')
	line = strings.TrimSpace(line)
	if line == "" {
		return def
	}
	return line
}

// redact masks a secret for display, preserving only a short prefix hint.
func redact(s string) string {
	if s == "" {
		return ""
	}
	if len(s) <= 4 {
		return "****"
	}
	return s[:4] + "****"
}
