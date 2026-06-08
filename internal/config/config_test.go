package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefaults(t *testing.T) {
	d := Defaults()
	if d.AIProvider != "openai" {
		t.Errorf("ai_provider default = %q, want openai", d.AIProvider)
	}
	if d.TokensMaxInput != 40960 {
		t.Errorf("tokens_max_input default = %d, want 40960", d.TokensMaxInput)
	}
	if d.GitPush {
		t.Errorf("gitpush default should be false")
	}
	if d.APICustomHeaders == nil {
		t.Errorf("api_custom_headers default should be non-nil map")
	}
}

func TestLoadMissingReturnsDefaults(t *testing.T) {
	cfg, err := Load(filepath.Join(t.TempDir(), "nope.yaml"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Model != Defaults().Model {
		t.Errorf("missing file should yield defaults, got model %q", cfg.Model)
	}
}

func TestSaveLoadRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "sub", "cfg.yaml")
	in := Defaults()
	in.Model = "gpt-4o-mini"
	in.Emoji = true
	in.APIKey = "sk-secret"
	if err := Save(in, path); err != nil {
		t.Fatalf("save: %v", err)
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat: %v", err)
	}
	if perm := info.Mode().Perm(); perm != 0o600 {
		t.Errorf("config perms = %o, want 600", perm)
	}

	out, err := Load(path)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if out.Model != "gpt-4o-mini" || !out.Emoji || out.APIKey != "sk-secret" {
		t.Errorf("round trip mismatch: %+v", out)
	}
}

func TestLoadOnlyOverridesPresentKeys(t *testing.T) {
	path := filepath.Join(t.TempDir(), "cfg.yaml")
	if err := os.WriteFile(path, []byte("model: custom-model\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if cfg.Model != "custom-model" {
		t.Errorf("model = %q, want custom-model", cfg.Model)
	}
	// Untouched key keeps default.
	if cfg.TokensMaxInput != Defaults().TokensMaxInput {
		t.Errorf("tokens_max_input = %d, want default", cfg.TokensMaxInput)
	}
}

func TestLoadParseError(t *testing.T) {
	path := filepath.Join(t.TempDir(), "bad.yaml")
	if err := os.WriteFile(path, []byte("model: [unterminated\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(path); err == nil {
		t.Fatal("expected parse error")
	}
}
