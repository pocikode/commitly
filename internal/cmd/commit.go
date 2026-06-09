package cmd

import (
	"context"
	"fmt"
	"io"

	"github.com/pocikode/commitly/internal/config"
	"github.com/pocikode/commitly/internal/git"
	"github.com/pocikode/commitly/internal/prompt"
	"github.com/pocikode/commitly/internal/provider"
	"github.com/pocikode/commitly/internal/tokens"
	"github.com/pocikode/commitly/internal/ui"
	"github.com/spf13/cobra"
)

func newCommitCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "commit",
		Short: "Generate a commit message from staged changes and commit",
		RunE:  runCommit,
	}
}

// ErrCancelled signals the user aborted the commit; the root handler maps it to
// a non-zero exit without an alarming "Error:" prefix.
var ErrCancelled = fmt.Errorf("commit cancelled")

// commitDeps bundles the collaborators of the commit flow so it can be unit
// tested with fakes.
type commitDeps struct {
	ctx      context.Context
	git      *git.Git
	provider provider.Provider
	ui       ui.UI
	cfg      config.Config
	out      io.Writer
}

// runCommit is the cobra entry point: it assembles real dependencies and runs
// the flow.
func runCommit(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()
	if ctx == nil {
		ctx = context.Background()
	}

	cfg, _, err := config.Resolve(config.Options{ConfigPath: flagConfig})
	if err != nil {
		return err
	}

	prov, err := provider.New(cfg)
	if err != nil {
		return err
	}

	g := git.New(".")
	if !g.IsRepo(ctx) {
		return fmt.Errorf("not a git repository")
	}

	return runCommitFlow(commitDeps{
		ctx:      ctx,
		git:      g,
		provider: prov,
		ui:       ui.Select(flagYes, cmd.OutOrStdout()),
		cfg:      cfg,
		out:      cmd.OutOrStdout(),
	})
}

// runCommitFlow orchestrates: ensure staged diff -> build prompt -> generate ->
// review (confirm/edit/regenerate/cancel) -> commit -> optional push.
func runCommitFlow(d commitDeps) error {
	if err := d.ensureStaged(); err != nil {
		return err
	}

	system, diff, truncated, err := prepareGeneration(d.ctx, d.cfg, d.git)
	if err != nil {
		return err
	}
	if truncated {
		d.ui.Info("warning: diff exceeded token budget and was reduced")
	}

	message, err := d.generate(system, diff)
	if err != nil {
		return err
	}

	for {
		d.ui.Preview(message)
		action, err := d.ui.Menu()
		if err != nil {
			return err
		}
		switch action {
		case ui.ActionConfirm:
			return d.commitAndPush(message)
		case ui.ActionEdit:
			edited, err := d.ui.Edit(message)
			if err != nil {
				return err
			}
			message = prompt.Clean(edited, d.cfg.OneLineCommit)
		case ui.ActionRegenerate:
			message, err = d.generate(system, diff)
			if err != nil {
				return err
			}
		case ui.ActionCancel:
			return ErrCancelled
		}
	}
}

// ErrNoFiles signals nothing remained in the diff after ignore filtering.
var ErrNoFiles = fmt.Errorf("no changes to commit after ignore filtering")

// prepareGeneration builds the system prompt and the (token-budgeted, ignore-
// filtered) diff for a commit. Shared by the interactive flow and the git hook.
func prepareGeneration(ctx context.Context, cfg config.Config, g *git.Git) (system, diff string, truncated bool, err error) {
	ig, err := git.LoadIgnore(g.Dir)
	if err != nil {
		return "", "", false, err
	}
	d, files, err := g.DiffFiltered(ctx, ig)
	if err != nil {
		return "", "", false, err
	}
	if len(files) == 0 {
		return "", "", false, ErrNoFiles
	}

	d, truncated = tokens.FitDiff(d, cfg.TokensMaxInput)

	override, _, err := prompt.LoadOverride(cfg.PromptModule)
	if err != nil {
		return "", "", false, err
	}
	system = prompt.System(cfg, prompt.Options{
		Override:   override,
		Commitlint: cfg.PromptModule == "@commitlint",
	})
	return system, d, truncated, nil
}

// ensureStaged makes sure there is something staged, offering to stage all when
// only unstaged changes exist, and erroring clearly when nothing changed.
func (d commitDeps) ensureStaged() error {
	staged, err := d.git.HasStagedChanges(d.ctx)
	if err != nil {
		return err
	}
	if staged {
		return nil
	}

	any, err := d.git.HasAnyChanges(d.ctx)
	if err != nil {
		return err
	}
	if !any {
		return fmt.Errorf("no changes detected — nothing to commit")
	}

	ok, err := d.ui.Confirm("No staged changes. Stage all changes?")
	if err != nil {
		return err
	}
	if !ok {
		return ErrCancelled
	}
	if err := d.git.StageAll(d.ctx); err != nil {
		return err
	}
	staged, err = d.git.HasStagedChanges(d.ctx)
	if err != nil {
		return err
	}
	if !staged {
		return fmt.Errorf("nothing to commit after staging")
	}
	return nil
}

// generate calls the provider behind a spinner and cleans the reply.
func (d commitDeps) generate(system, diff string) (string, error) {
	req := provider.CommitRequest{
		System:          system,
		User:            diff,
		Model:           d.cfg.Model,
		MaxOutputTokens: d.cfg.TokensMaxOutput,
	}
	raw, err := d.ui.Spinner(d.ctx, "generating commit message...", func() (string, error) {
		return d.provider.GenerateCommitMessage(d.ctx, req)
	})
	if err != nil {
		return "", err
	}
	msg := prompt.Clean(raw, d.cfg.OneLineCommit)
	if msg == "" {
		return "", fmt.Errorf("provider returned an empty commit message")
	}
	return msg, nil
}

// commitAndPush creates the commit and pushes when configured.
func (d commitDeps) commitAndPush(message string) error {
	if err := d.git.Commit(d.ctx, message); err != nil {
		return err
	}
	d.ui.Info("✔ committed")

	if !d.cfg.GitPush {
		return nil
	}
	ok, err := d.ui.Confirm("Push to remote?")
	if err != nil {
		return err
	}
	if !ok {
		return nil
	}
	if err := d.git.Push(d.ctx); err != nil {
		return err
	}
	d.ui.Info("✔ pushed")
	return nil
}
