package git

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestLoadIgnoreMissingFile(t *testing.T) {
	ig, err := LoadIgnore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	// Lock files still excluded even without an ignore file.
	if !ig.Match("package-lock.json") {
		t.Error("expected lock file to be ignored by default")
	}
	if ig.Match("main.go") {
		t.Error("regular file should not be ignored")
	}
}

func TestLockFileInSubdir(t *testing.T) {
	ig, _ := LoadIgnore(t.TempDir())
	if !ig.Match("frontend/yarn.lock") {
		t.Error("lock file in subdir should be ignored")
	}
}

func TestIgnorePatterns(t *testing.T) {
	dir := t.TempDir()
	content := "# comment\n*.snap\ndist/\nsecret.txt\n\n"
	if err := os.WriteFile(filepath.Join(dir, IgnoreFileName), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	ig, err := LoadIgnore(dir)
	if err != nil {
		t.Fatal(err)
	}

	cases := map[string]bool{
		"foo.snap":            true,  // glob basename
		"comp/Button.snap":    true,  // glob basename in subdir
		"dist/bundle.js":      true,  // dir prefix
		"dist":                true,  // dir exact
		"secret.txt":          true,  // exact name
		"src/secret.txt":      true,  // basename match
		"main.go":             false, // not matched
		"distinct/file.go":    false, // not under dist/
	}
	for p, want := range cases {
		if got := ig.Match(p); got != want {
			t.Errorf("Match(%q) = %v, want %v", p, got, want)
		}
	}
}

func TestFilter(t *testing.T) {
	ig, _ := LoadIgnore(t.TempDir())
	in := []string{"main.go", "go.sum", "pkg/util.go", "yarn.lock"}
	out := ig.Filter(in)
	want := []string{"main.go", "pkg/util.go"}
	if len(out) != len(want) {
		t.Fatalf("filtered = %v, want %v", out, want)
	}
	for i := range want {
		if out[i] != want[i] {
			t.Errorf("filtered[%d] = %q, want %q", i, out[i], want[i])
		}
	}
}

func TestDiffFiltered(t *testing.T) {
	ctx := context.Background()
	g := newTestRepo(t)
	writeFile(t, g, "main.go", "package main\n")
	writeFile(t, g, "go.sum", "h1:fakehash\n")
	if err := g.StageAll(ctx); err != nil {
		t.Fatal(err)
	}

	ig, _ := LoadIgnore(g.Dir)
	diff, files, err := g.DiffFiltered(ctx, ig)
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 1 || files[0] != "main.go" {
		t.Errorf("filtered files = %v, want [main.go]", files)
	}
	if contains(diff, "go.sum") {
		t.Errorf("diff should not include go.sum: %q", diff)
	}
	if !contains(diff, "main.go") {
		t.Errorf("diff should include main.go: %q", diff)
	}
}

func TestDiffFilteredAllIgnored(t *testing.T) {
	ctx := context.Background()
	g := newTestRepo(t)
	writeFile(t, g, "go.sum", "h1:x\n")
	if err := g.StageAll(ctx); err != nil {
		t.Fatal(err)
	}
	ig, _ := LoadIgnore(g.Dir)
	diff, files, err := g.DiffFiltered(ctx, ig)
	if err != nil {
		t.Fatal(err)
	}
	if diff != "" || files != nil {
		t.Errorf("expected empty result when all ignored, got diff=%q files=%v", diff, files)
	}
}
