package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestGlobalPathPrecedence(t *testing.T) {
	// Explicit beats env.
	t.Setenv("CLY_CONFIG_PATH", "/env/path.yaml")
	p, err := GlobalPath("/explicit/path.yaml")
	if err != nil {
		t.Fatal(err)
	}
	if p != "/explicit/path.yaml" {
		t.Errorf("explicit path = %q", p)
	}

	// Env beats home default.
	p, err = GlobalPath("")
	if err != nil {
		t.Fatal(err)
	}
	if p != "/env/path.yaml" {
		t.Errorf("env path = %q", p)
	}
}

func TestResolvePrecedence(t *testing.T) {
	dir := t.TempDir()
	global := filepath.Join(dir, "global.yaml")
	if err := os.WriteFile(global, []byte("model: from-global\nemoji: false\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	env := map[string]string{
		"CLY_MODEL": "from-env",
	}
	lookup := func(k string) (string, bool) { v, ok := env[k]; return v, ok }

	cfg, _, err := Resolve(Options{
		ConfigPath: global,
		Env:        lookup,
		Flags:      map[string]string{"model": "from-flag"},
	})
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	// flag > env > file
	if cfg.Model != "from-flag" {
		t.Errorf("model = %q, want from-flag", cfg.Model)
	}
}

func TestResolveEnvOverridesFile(t *testing.T) {
	dir := t.TempDir()
	global := filepath.Join(dir, "global.yaml")
	if err := os.WriteFile(global, []byte("model: from-global\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	env := map[string]string{"CLY_MODEL": "from-env"}
	cfg, _, err := Resolve(Options{
		ConfigPath: global,
		Env:        func(k string) (string, bool) { v, ok := env[k]; return v, ok },
	})
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Model != "from-env" {
		t.Errorf("model = %q, want from-env", cfg.Model)
	}
}

func TestResolveInvalidEnvErrors(t *testing.T) {
	dir := t.TempDir()
	global := filepath.Join(dir, "global.yaml")
	if err := os.WriteFile(global, []byte("model: x\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	env := map[string]string{"CLY_TOKENS_MAX_INPUT": "-1"}
	_, _, err := Resolve(Options{
		ConfigPath: global,
		Env:        func(k string) (string, bool) { v, ok := env[k]; return v, ok },
	})
	if err == nil {
		t.Fatal("expected error from invalid env value")
	}
}

func TestResolveProjectOverridesGlobal(t *testing.T) {
	dir := t.TempDir()
	global := filepath.Join(dir, "global.yaml")
	if err := os.WriteFile(global, []byte("model: from-global\nemoji: false\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	// Work in a dedicated dir holding a project-local config.
	projDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(projDir, ProjectFileName), []byte("model: from-project\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	wd, _ := os.Getwd()
	t.Cleanup(func() { _ = os.Chdir(wd) })
	if err := os.Chdir(projDir); err != nil {
		t.Fatal(err)
	}

	cfg, _, err := Resolve(Options{
		ConfigPath: global,
		Env:        func(string) (string, bool) { return "", false },
	})
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	// project file overrides global file
	if cfg.Model != "from-project" {
		t.Errorf("model = %q, want from-project", cfg.Model)
	}
}

func TestProjectPathAbsentReturnsEmpty(t *testing.T) {
	wd, _ := os.Getwd()
	t.Cleanup(func() { _ = os.Chdir(wd) })
	if err := os.Chdir(t.TempDir()); err != nil {
		t.Fatal(err)
	}
	if p := ProjectPath(); p != "" {
		t.Errorf("expected empty project path, got %q", p)
	}
}

func TestResolveDefaultsWhenNoFile(t *testing.T) {
	cfg, _, err := Resolve(Options{
		ConfigPath: filepath.Join(t.TempDir(), "missing.yaml"),
		Env:        func(string) (string, bool) { return "", false },
	})
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Model != Defaults().Model {
		t.Errorf("model = %q, want default", cfg.Model)
	}
}

func TestResolveActiveProfileOverlay(t *testing.T) {
	dir := t.TempDir()
	global := filepath.Join(dir, "global.yaml")
	yaml := "active_profile: a\n" +
		"profiles:\n" +
		"  a:\n" +
		"    ai_provider: openai\n" +
		"    provider_type: openai_compatible\n" +
		"    api_url: https://a/v1\n" +
		"    api_key: ka\n" +
		"    model: model-a\n" +
		"  b:\n" +
		"    ai_provider: anthropic\n" +
		"    provider_type: anthropic_compatible\n" +
		"    api_url: https://b/v1\n" +
		"    api_key: kb\n" +
		"    model: model-b\n"
	if err := os.WriteFile(global, []byte(yaml), 0o600); err != nil {
		t.Fatal(err)
	}

	// File active_profile=a overlays its fields.
	cfg, _, err := Resolve(Options{ConfigPath: global})
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Model != "model-a" || cfg.APIURL != "https://a/v1" {
		t.Errorf("profile a not applied: model=%q url=%q", cfg.Model, cfg.APIURL)
	}

	// Env CLY_ACTIVE_PROFILE switches to b; flag/env per-key still wins.
	env := map[string]string{"CLY_ACTIVE_PROFILE": "b"}
	cfg, _, err = Resolve(Options{
		ConfigPath: global,
		Env:        func(k string) (string, bool) { v, ok := env[k]; return v, ok },
		Flags:      map[string]string{"model": "override"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if cfg.AIProvider != "anthropic" || cfg.APIKey != "kb" {
		t.Errorf("profile b not applied: provider=%q key=%q", cfg.AIProvider, cfg.APIKey)
	}
	if cfg.Model != "override" {
		t.Errorf("flag should override profile model, got %q", cfg.Model)
	}

	// Unknown active profile is ignored (no overlay, no error).
	env = map[string]string{"CLY_ACTIVE_PROFILE": "ghost"}
	cfg, _, err = Resolve(Options{
		ConfigPath: global,
		Env:        func(k string) (string, bool) { v, ok := env[k]; return v, ok },
	})
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Model != "model-a" {
		t.Errorf("unknown profile should keep file active a, got model=%q", cfg.Model)
	}
}
