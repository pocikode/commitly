// Package git wraps the git binary for the operations OpenCommit-Go needs:
// reading the staged diff, detecting changes, staging, committing, and pushing.
// It shells out to git rather than using a pure-Go implementation.
package git

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
)

// Git runs git commands in a working directory.
type Git struct {
	Dir string
	// run executes git with args and returns stdout. Overridable in tests;
	// defaults to execGit.
	run func(ctx context.Context, dir string, args ...string) (string, error)
}

// New returns a Git rooted at dir.
func New(dir string) *Git {
	return &Git{Dir: dir, run: execGit}
}

// execGit runs the git binary and returns trimmed stdout, mapping non-zero
// exits to an error that includes stderr.
func execGit(ctx context.Context, dir string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = dir
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			msg = err.Error()
		}
		return "", fmt.Errorf("git %s: %s", strings.Join(args, " "), msg)
	}
	return stdout.String(), nil
}

func (g *Git) exec(ctx context.Context, args ...string) (string, error) {
	return g.run(ctx, g.Dir, args...)
}

// IsRepo reports whether Dir is inside a git work tree.
func (g *Git) IsRepo(ctx context.Context) bool {
	out, err := g.exec(ctx, "rev-parse", "--is-inside-work-tree")
	return err == nil && strings.TrimSpace(out) == "true"
}

// Root returns the absolute path to the repository top level.
func (g *Git) Root(ctx context.Context) (string, error) {
	out, err := g.exec(ctx, "rev-parse", "--show-toplevel")
	return strings.TrimSpace(out), err
}

// HooksDir returns the absolute path to the repository's hooks directory.
func (g *Git) HooksDir(ctx context.Context) (string, error) {
	out, err := g.exec(ctx, "rev-parse", "--git-path", "hooks")
	if err != nil {
		return "", err
	}
	dir := strings.TrimSpace(out)
	if dir == "" {
		return "", fmt.Errorf("could not resolve git hooks directory")
	}
	// git-path may be relative to the work tree; make it absolute.
	if !filepath.IsAbs(dir) {
		dir = filepath.Join(g.Dir, dir)
	}
	return dir, nil
}

// StagedFiles returns the paths staged for commit.
func (g *Git) StagedFiles(ctx context.Context) ([]string, error) {
	out, err := g.exec(ctx, "diff", "--staged", "--name-only")
	if err != nil {
		return nil, err
	}
	return splitLines(out), nil
}

// StagedDiff returns the staged diff, optionally restricted to the given paths.
// With no paths it returns the full staged diff.
func (g *Git) StagedDiff(ctx context.Context, paths ...string) (string, error) {
	args := []string{"diff", "--staged"}
	if len(paths) > 0 {
		args = append(args, "--")
		args = append(args, paths...)
	}
	return g.exec(ctx, args...)
}

// HasStagedChanges reports whether anything is staged.
func (g *Git) HasStagedChanges(ctx context.Context) (bool, error) {
	files, err := g.StagedFiles(ctx)
	if err != nil {
		return false, err
	}
	return len(files) > 0, nil
}

// HasAnyChanges reports whether the work tree has any tracked/untracked changes.
func (g *Git) HasAnyChanges(ctx context.Context) (bool, error) {
	out, err := g.exec(ctx, "status", "--porcelain")
	if err != nil {
		return false, err
	}
	return strings.TrimSpace(out) != "", nil
}

// StageAll stages all changes (git add -A).
func (g *Git) StageAll(ctx context.Context) error {
	_, err := g.exec(ctx, "add", "-A")
	return err
}

// Commit creates a commit with the given message.
func (g *Git) Commit(ctx context.Context, message string) error {
	_, err := g.exec(ctx, "commit", "-m", message)
	return err
}

// Push pushes the current branch to its upstream.
func (g *Git) Push(ctx context.Context) error {
	_, err := g.exec(ctx, "push")
	return err
}

func splitLines(s string) []string {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	return strings.Split(s, "\n")
}
