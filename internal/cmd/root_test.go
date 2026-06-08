package cmd

import (
	"bytes"
	"strings"
	"testing"

	"github.com/pocikode/opencommit/internal/version"
)

// execute runs the root command with the given args, capturing stdout/stderr.
func execute(t *testing.T, args ...string) (string, error) {
	t.Helper()
	root := newRootCmd()
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs(args)
	err := root.Execute()
	return out.String(), err
}

func TestVersionFlag(t *testing.T) {
	version.Version = "v9.9.9"
	out, err := execute(t, "--version")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, "v9.9.9") {
		t.Fatalf("version output missing version string: %q", out)
	}
	if !strings.HasPrefix(out, "oco ") {
		t.Fatalf("version output should start with 'oco ': %q", out)
	}
}

func TestHelpListsSubcommands(t *testing.T) {
	out, err := execute(t, "--help")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for _, sub := range []string{"commit", "config", "hook", "models", "commitlint"} {
		if !strings.Contains(out, sub) {
			t.Errorf("help output missing subcommand %q", sub)
		}
	}
}

func TestGlobalFlagsRegistered(t *testing.T) {
	root := newRootCmd()
	for _, name := range []string{"yes", "config"} {
		if root.PersistentFlags().Lookup(name) == nil {
			t.Errorf("global flag --%s not registered", name)
		}
	}
}

func TestBareCommandDefaultsToCommit(t *testing.T) {
	// Bare `oco` should route to the commit flow. The stub returns a
	// "commit: not implemented" error, which proves routing reached runCommit.
	_, err := execute(t)
	if err == nil || !strings.Contains(err.Error(), "commit:") {
		t.Fatalf("bare oco should default to commit, got err: %v", err)
	}
}

func TestSubcommandStubsRegistered(t *testing.T) {
	root := newRootCmd()
	want := map[string]bool{"commit": false, "config": false, "hook": false, "models": false, "commitlint": false}
	for _, c := range root.Commands() {
		if _, ok := want[c.Name()]; ok {
			want[c.Name()] = true
		}
	}
	for name, found := range want {
		if !found {
			t.Errorf("subcommand %q not registered", name)
		}
	}
}
