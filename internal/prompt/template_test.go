package prompt

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadOverrideBuiltIn(t *testing.T) {
	for _, name := range []string{"", "conventional-commit", "@commitlint"} {
		override, ok, err := LoadOverride(name)
		if err != nil || ok || override != "" {
			t.Errorf("built-in %q should yield no override, got (%q,%v,%v)", name, override, ok, err)
		}
	}
}

func TestLoadOverrideFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "tmpl.txt")
	if err := os.WriteFile(path, []byte("my custom system prompt"), 0o644); err != nil {
		t.Fatal(err)
	}
	override, ok, err := LoadOverride(path)
	if err != nil || !ok {
		t.Fatalf("expected override loaded, got ok=%v err=%v", ok, err)
	}
	if override != "my custom system prompt" {
		t.Errorf("override = %q", override)
	}
}

func TestLoadOverrideMissingFile(t *testing.T) {
	if _, _, err := LoadOverride("/no/such/template/file"); err == nil {
		t.Fatal("expected error for missing template file")
	}
}

func TestApplyTemplatePlaceholder(t *testing.T) {
	tmpl := "Prefix\n$msg\nSuffix"
	got := ApplyTemplatePlaceholder(tmpl, "feat: x", "$msg")
	want := "Prefix\nfeat: x\nSuffix"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}

	// Placeholder absent -> message returned as-is.
	if got := ApplyTemplatePlaceholder("no placeholder", "feat: x", "$msg"); got != "feat: x" {
		t.Errorf("absent placeholder should return message, got %q", got)
	}
	// Empty placeholder -> message as-is.
	if got := ApplyTemplatePlaceholder("$msg", "feat: x", ""); got != "feat: x" {
		t.Errorf("empty placeholder should return message, got %q", got)
	}
}
