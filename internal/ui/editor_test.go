package ui

import (
	"os"
	"strings"
	"testing"
)

func TestResolveEditorPrefersVisual(t *testing.T) {
	t.Setenv("EDITOR", "nano")
	t.Setenv("VISUAL", "vim")
	if got := resolveEditor(); got != "vim" {
		t.Errorf("resolveEditor = %q, want vim", got)
	}
	os.Unsetenv("VISUAL")
	if got := resolveEditor(); got != "nano" {
		t.Errorf("resolveEditor = %q, want nano", got)
	}
}

func TestExternalEditRoundTrip(t *testing.T) {
	// Fake editor: rewrites the file contents.
	orig := editorRunner
	t.Cleanup(func() { editorRunner = orig })
	editorRunner = func(editor, path string) error {
		return os.WriteFile(path, []byte("fix: edited by editor\n"), 0o644)
	}

	got, err := externalEdit("fake-editor", "feat: original")
	if err != nil {
		t.Fatal(err)
	}
	if got != "fix: edited by editor" {
		t.Errorf("edited = %q", got)
	}
}

func TestEditMessageUsesEditorWhenSet(t *testing.T) {
	t.Setenv("EDITOR", "fake")
	orig := editorRunner
	t.Cleanup(func() { editorRunner = orig })
	editorRunner = func(editor, path string) error {
		data, _ := os.ReadFile(path)
		// Append a line to prove the original was seeded into the file.
		return os.WriteFile(path, append(data, []byte("\nmore")...), 0o644)
	}

	got, err := EditMessage("feat: seed")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(got, "feat: seed") || !strings.Contains(got, "more") {
		t.Errorf("edited = %q", got)
	}
}

func TestExternalEditEditorError(t *testing.T) {
	orig := editorRunner
	t.Cleanup(func() { editorRunner = orig })
	editorRunner = func(editor, path string) error { return os.ErrPermission }
	if _, err := externalEdit("fake", "x"); err == nil {
		t.Fatal("expected editor error to propagate")
	}
}
