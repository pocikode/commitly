package cmd

import (
	"bytes"
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pocikode/commitly/internal/config"
	"github.com/pocikode/commitly/internal/git"
	"github.com/pocikode/commitly/internal/provider"
	"github.com/pocikode/commitly/internal/ui"
)

// --- test doubles ---

type fakeProvider struct {
	msg   string
	err   error
	calls int
}

func (f *fakeProvider) GenerateCommitMessage(ctx context.Context, req provider.CommitRequest) (string, error) {
	f.calls++
	return f.msg, f.err
}

type fakeUI struct {
	actions  []ui.Action
	idx      int
	confirm  bool
	edited   string
	infos    []string
	previews []string
}

func (f *fakeUI) Spinner(ctx context.Context, label string, fn func() (string, error)) (string, error) {
	return fn()
}
func (f *fakeUI) Preview(m string) { f.previews = append(f.previews, m) }
func (f *fakeUI) Menu() (ui.Action, error) {
	if f.idx >= len(f.actions) {
		return ui.ActionCancel, nil
	}
	a := f.actions[f.idx]
	f.idx++
	return a, nil
}
func (f *fakeUI) Edit(m string) (string, error) {
	if f.edited != "" {
		return f.edited, nil
	}
	return m, nil
}
func (f *fakeUI) Confirm(string) (bool, error) { return f.confirm, nil }
func (f *fakeUI) Info(m string)                { f.infos = append(f.infos, m) }

// --- helpers ---

func gitInit(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	for _, args := range [][]string{
		{"init", "-q"},
		{"config", "user.email", "t@e.com"},
		{"config", "user.name", "T"},
		{"config", "commit.gpgsign", "false"},
	} {
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	return dir
}

func write(t *testing.T, dir, name, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func stage(t *testing.T, dir string) {
	t.Helper()
	cmd := exec.Command("git", "add", "-A")
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git add: %v\n%s", err, out)
	}
}

func lastCommitMsg(t *testing.T, dir string) string {
	t.Helper()
	cmd := exec.Command("git", "log", "-1", "--pretty=%B")
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

func deps(dir string, prov provider.Provider, u ui.UI, cfg config.Config) commitDeps {
	return commitDeps{
		ctx:      context.Background(),
		git:      git.New(dir),
		provider: prov,
		ui:       u,
		cfg:      cfg,
		out:      &bytes.Buffer{},
	}
}

// --- tests ---

func TestCommitFlowConfirm(t *testing.T) {
	dir := gitInit(t)
	write(t, dir, "a.go", "package main\n")
	stage(t, dir)

	u := &fakeUI{actions: []ui.Action{ui.ActionConfirm}}
	d := deps(dir, &fakeProvider{msg: "feat: add a"}, u, config.Defaults())

	if err := runCommitFlow(d); err != nil {
		t.Fatalf("flow: %v", err)
	}
	if got := lastCommitMsg(t, dir); got != "feat: add a" {
		t.Errorf("commit message = %q", got)
	}
}

func TestCommitFlowCancel(t *testing.T) {
	dir := gitInit(t)
	write(t, dir, "a.go", "x\n")
	stage(t, dir)

	u := &fakeUI{actions: []ui.Action{ui.ActionCancel}}
	d := deps(dir, &fakeProvider{msg: "feat: x"}, u, config.Defaults())

	if err := runCommitFlow(d); !errors.Is(err, ErrCancelled) {
		t.Fatalf("want ErrCancelled, got %v", err)
	}
	if lastCommitMsg(t, dir) != "" {
		t.Error("no commit should exist after cancel")
	}
}

func TestCommitFlowRegenerate(t *testing.T) {
	dir := gitInit(t)
	write(t, dir, "a.go", "x\n")
	stage(t, dir)

	prov := &fakeProvider{msg: "feat: x"}
	u := &fakeUI{actions: []ui.Action{ui.ActionRegenerate, ui.ActionConfirm}}
	d := deps(dir, prov, u, config.Defaults())

	if err := runCommitFlow(d); err != nil {
		t.Fatal(err)
	}
	if prov.calls != 2 {
		t.Errorf("provider calls = %d, want 2 (initial + regenerate)", prov.calls)
	}
}

func TestCommitFlowEdit(t *testing.T) {
	dir := gitInit(t)
	write(t, dir, "a.go", "x\n")
	stage(t, dir)

	u := &fakeUI{actions: []ui.Action{ui.ActionEdit, ui.ActionConfirm}, edited: "fix: edited message"}
	d := deps(dir, &fakeProvider{msg: "feat: original"}, u, config.Defaults())

	if err := runCommitFlow(d); err != nil {
		t.Fatal(err)
	}
	if got := lastCommitMsg(t, dir); got != "fix: edited message" {
		t.Errorf("commit message = %q, want edited", got)
	}
}

func TestEnsureStagedNoChanges(t *testing.T) {
	dir := gitInit(t)
	d := deps(dir, &fakeProvider{msg: "x"}, &fakeUI{}, config.Defaults())
	err := runCommitFlow(d)
	if err == nil || !strings.Contains(err.Error(), "no changes") {
		t.Fatalf("want no-changes error, got %v", err)
	}
}

func TestEnsureStagedPromptStageAccepted(t *testing.T) {
	dir := gitInit(t)
	write(t, dir, "a.go", "x\n") // unstaged

	u := &fakeUI{actions: []ui.Action{ui.ActionConfirm}, confirm: true}
	d := deps(dir, &fakeProvider{msg: "feat: staged"}, u, config.Defaults())

	if err := runCommitFlow(d); err != nil {
		t.Fatalf("flow: %v", err)
	}
	if got := lastCommitMsg(t, dir); got != "feat: staged" {
		t.Errorf("commit message = %q", got)
	}
}

func TestEnsureStagedPromptStageDeclined(t *testing.T) {
	dir := gitInit(t)
	write(t, dir, "a.go", "x\n")

	u := &fakeUI{confirm: false}
	d := deps(dir, &fakeProvider{msg: "x"}, u, config.Defaults())

	if err := runCommitFlow(d); !errors.Is(err, ErrCancelled) {
		t.Fatalf("want ErrCancelled when staging declined, got %v", err)
	}
}

func TestCommitFlowPushPrompt(t *testing.T) {
	dir := gitInit(t)
	write(t, dir, "a.go", "x\n")
	stage(t, dir)

	cfg := config.Defaults()
	cfg.GitPush = true
	// confirm=false so the push is declined (no remote needed).
	u := &fakeUI{actions: []ui.Action{ui.ActionConfirm}, confirm: false}
	d := deps(dir, &fakeProvider{msg: "feat: x"}, u, cfg)

	if err := runCommitFlow(d); err != nil {
		t.Fatalf("flow: %v", err)
	}
	if lastCommitMsg(t, dir) != "feat: x" {
		t.Error("commit should still be created when push declined")
	}
}

func TestCommitFlowIgnoresLockFiles(t *testing.T) {
	dir := gitInit(t)
	write(t, dir, "go.sum", "h1:x\n") // only a lock file staged
	stage(t, dir)

	u := &fakeUI{actions: []ui.Action{ui.ActionConfirm}}
	d := deps(dir, &fakeProvider{msg: "x"}, u, config.Defaults())

	err := runCommitFlow(d)
	if err == nil || !strings.Contains(err.Error(), "after ignore filtering") {
		t.Fatalf("want ignore-filtering error, got %v", err)
	}
}
