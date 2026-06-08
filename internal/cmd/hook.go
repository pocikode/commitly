package cmd

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/pocikode/opencommit/internal/config"
	"github.com/pocikode/opencommit/internal/git"
	"github.com/pocikode/opencommit/internal/prompt"
	"github.com/pocikode/opencommit/internal/provider"
	"github.com/spf13/cobra"
)

// hookName is the git hook OpenCommit-Go manages.
const hookName = "prepare-commit-msg"

// hookMarker identifies hooks installed by oco so unset only removes our own.
const hookMarker = "# opencommit-go managed hook"

// hookScript is the prepare-commit-msg hook body. It delegates to `oco hook
// run`, passing git's hook arguments through.
const hookScript = `#!/bin/sh
` + hookMarker + `
oco hook run "$1" "$2" "$3"
`

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
				return runHookSet(cmd.Context(), cmd.OutOrStdout())
			},
		},
		&cobra.Command{
			Use:   "unset",
			Short: "Remove the prepare-commit-msg hook (only if installed by oco)",
			RunE: func(cmd *cobra.Command, args []string) error {
				return runHookUnset(cmd.Context(), cmd.OutOrStdout())
			},
		},
		&cobra.Command{
			Use:    "run <msg-file> [source] [sha]",
			Short:  "Hook runtime: generate and write the commit message",
			Hidden: true,
			Args:   cobra.MinimumNArgs(1),
			RunE: func(cmd *cobra.Command, args []string) error {
				source := ""
				if len(args) > 1 {
					source = args[1]
				}
				return runHookRuntime(cmd.Context(), cmd.ErrOrStderr(), args[0], source)
			},
		},
	)

	return cmd
}

// hookPath resolves the absolute path to the prepare-commit-msg hook.
func hookPath(ctx context.Context) (string, error) {
	g := git.New(".")
	if !g.IsRepo(ctx) {
		return "", fmt.Errorf("not a git repository")
	}
	dir, err := g.HooksDir(ctx)
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, hookName), nil
}

// runHookSet installs the hook, refusing to overwrite a foreign hook.
func runHookSet(ctx context.Context, out io.Writer) error {
	path, err := hookPath(ctx)
	if err != nil {
		return err
	}
	if existing, err := os.ReadFile(path); err == nil {
		if !isOcoHook(existing) {
			return fmt.Errorf("a non-oco %s hook already exists at %s; remove it first", hookName, path)
		}
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	if err := os.WriteFile(path, []byte(hookScript), 0o755); err != nil {
		return err
	}
	fmt.Fprintf(out, "Installed %s hook at %s\n", hookName, path)
	return nil
}

// runHookUnset removes the hook only when it is owned by oco.
func runHookUnset(ctx context.Context, out io.Writer) error {
	path, err := hookPath(ctx)
	if err != nil {
		return err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			fmt.Fprintf(out, "No %s hook installed\n", hookName)
			return nil
		}
		return err
	}
	if !isOcoHook(data) {
		return fmt.Errorf("%s hook at %s was not installed by oco; leaving it untouched", hookName, path)
	}
	if err := os.Remove(path); err != nil {
		return err
	}
	fmt.Fprintf(out, "Removed %s hook\n", hookName)
	return nil
}

// isOcoHook reports whether hook content carries the oco ownership marker.
func isOcoHook(content []byte) bool {
	return bytes.Contains(content, []byte(hookMarker))
}

// dropCommentLines removes lines that begin with '#' (git comment lines).
func dropCommentLines(s string) string {
	lines := strings.Split(s, "\n")
	kept := lines[:0]
	for _, l := range lines {
		if strings.HasPrefix(strings.TrimSpace(l), "#") {
			continue
		}
		kept = append(kept, l)
	}
	return strings.TrimLeft(strings.Join(kept, "\n"), "\n")
}

// runHookRuntime generates a commit message and writes it into git's commit
// message file. It degrades gracefully: on any failure it warns and returns nil
// so the commit is not blocked (the user can still edit/commit manually).
func runHookRuntime(ctx context.Context, errOut io.Writer, msgFile, source string) error {
	if ctx == nil {
		ctx = context.Background()
	}
	// Skip when git already has a message source (e.g. -m, merge, squash,
	// template, or an existing commit) to avoid clobbering the user's intent.
	if source != "" {
		return nil
	}

	cfg, _, err := config.Resolve(config.Options{ConfigPath: flagConfig})
	if err != nil {
		fmt.Fprintf(errOut, "oco hook: %v\n", err)
		return nil
	}

	msg, err := hookGenerate(ctx, cfg)
	if err != nil {
		fmt.Fprintf(errOut, "oco hook: %v\n", err)
		return nil
	}

	if err := writeHookMessage(msgFile, msg, cfg.HookAutoUncomment); err != nil {
		fmt.Fprintf(errOut, "oco hook: %v\n", err)
		return nil
	}
	return nil
}

// hookGenerate runs the non-interactive generation pipeline.
func hookGenerate(ctx context.Context, cfg config.Config) (string, error) {
	g := git.New(".")
	prov, err := provider.New(cfg)
	if err != nil {
		return "", err
	}
	system, diff, _, err := prepareGeneration(ctx, cfg, g)
	if err != nil {
		return "", err
	}
	raw, err := prov.GenerateCommitMessage(ctx, provider.CommitRequest{
		System:          system,
		User:            diff,
		Model:           cfg.Model,
		MaxOutputTokens: cfg.TokensMaxOutput,
	})
	if err != nil {
		return "", err
	}
	msg := prompt.Clean(raw, cfg.OneLineCommit)
	if msg == "" {
		return "", fmt.Errorf("provider returned an empty commit message")
	}
	return msg, nil
}

// writeHookMessage prepends the generated message above any existing content
// (git's comment template). When autoUncomment is set, existing leading comment
// lines from a prior template are dropped.
func writeHookMessage(msgFile, msg string, autoUncomment bool) error {
	existing, _ := os.ReadFile(msgFile)
	body := string(existing)
	if autoUncomment {
		body = dropCommentLines(body)
	}
	out := msg + "\n"
	if body != "" {
		out += "\n" + body
	}
	return os.WriteFile(msgFile, []byte(out), 0o644)
}
