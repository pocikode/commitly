package git

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// newTestRepo creates an initialized temp git repo and returns a Git rooted
// there. It configures identity so commits work in CI.
func newTestRepo(t *testing.T) *Git {
	t.Helper()
	dir := t.TempDir()
	ctx := context.Background()
	for _, args := range [][]string{
		{"init", "-q"},
		{"config", "user.email", "test@example.com"},
		{"config", "user.name", "Test"},
		{"config", "commit.gpgsign", "false"},
	} {
		cmd := exec.CommandContext(ctx, "git", args...)
		cmd.Dir = dir
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	return New(dir)
}

func writeFile(t *testing.T, g *Git, name, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(g.Dir, name), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestIsRepo(t *testing.T) {
	g := newTestRepo(t)
	if !g.IsRepo(context.Background()) {
		t.Error("expected IsRepo true")
	}
	if New(t.TempDir()).IsRepo(context.Background()) {
		t.Error("expected IsRepo false for non-repo")
	}
}

func TestStagedDiffAndCommit(t *testing.T) {
	ctx := context.Background()
	g := newTestRepo(t)

	// Nothing staged yet.
	if staged, err := g.HasStagedChanges(ctx); err != nil || staged {
		t.Fatalf("expected no staged changes, got staged=%v err=%v", staged, err)
	}
	if any, err := g.HasAnyChanges(ctx); err != nil || any {
		t.Fatalf("expected no changes at all, got any=%v err=%v", any, err)
	}

	writeFile(t, g, "a.txt", "hello\n")
	if any, _ := g.HasAnyChanges(ctx); !any {
		t.Fatal("expected changes after writing file")
	}

	if err := g.StageAll(ctx); err != nil {
		t.Fatalf("stage: %v", err)
	}
	staged, err := g.HasStagedChanges(ctx)
	if err != nil || !staged {
		t.Fatalf("expected staged changes, got %v err=%v", staged, err)
	}

	files, err := g.StagedFiles(ctx)
	if err != nil || len(files) != 1 || files[0] != "a.txt" {
		t.Fatalf("staged files = %v err=%v", files, err)
	}

	diff, err := g.StagedDiff(ctx)
	if err != nil || !contains(diff, "hello") {
		t.Fatalf("diff = %q err=%v", diff, err)
	}

	if err := g.Commit(ctx, "feat: add a.txt"); err != nil {
		t.Fatalf("commit: %v", err)
	}
	if staged, _ := g.HasStagedChanges(ctx); staged {
		t.Error("expected clean stage after commit")
	}
}

func TestRoot(t *testing.T) {
	g := newTestRepo(t)
	root, err := g.Root(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if root == "" {
		t.Error("expected non-empty root")
	}
}

func TestErrorPropagation(t *testing.T) {
	// Running in a non-repo should error for diff.
	g := New(t.TempDir())
	if _, err := g.StagedFiles(context.Background()); err == nil {
		t.Error("expected error in non-repo")
	}
}

func TestPushUsesRunner(t *testing.T) {
	var gotArgs []string
	g := &Git{Dir: "/x", run: func(ctx context.Context, dir string, args ...string) (string, error) {
		gotArgs = args
		return "", nil
	}}
	if err := g.Push(context.Background()); err != nil {
		t.Fatalf("push: %v", err)
	}
	if len(gotArgs) != 1 || gotArgs[0] != "push" {
		t.Errorf("push args = %v, want [push]", gotArgs)
	}
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && (len(sub) == 0 || indexOf(s, sub) >= 0)
}

func indexOf(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}
