package cmd

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"
)

func TestConfigSetAndGet(t *testing.T) {
	path := filepath.Join(t.TempDir(), "cfg.yaml")

	out, err := execute(t, "config", "set", "model=gpt-4o-mini", "emoji=true", "--config", path)
	if err != nil {
		t.Fatalf("config set: %v", err)
	}
	if !strings.Contains(out, "Saved 2 value(s)") {
		t.Errorf("unexpected set output: %q", out)
	}

	out, err = execute(t, "config", "get", "model", "emoji", "--config", path)
	if err != nil {
		t.Fatalf("config get: %v", err)
	}
	if !strings.Contains(out, "model=gpt-4o-mini") || !strings.Contains(out, "emoji=true") {
		t.Errorf("unexpected get output: %q", out)
	}
}

func TestConfigGetRedactsAPIKey(t *testing.T) {
	path := filepath.Join(t.TempDir(), "cfg.yaml")
	if _, err := execute(t, "config", "set", "api_key=sk-supersecret", "--config", path); err != nil {
		t.Fatalf("set: %v", err)
	}
	out, err := execute(t, "config", "get", "api_key", "--config", path)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if strings.Contains(out, "supersecret") {
		t.Errorf("api_key was not redacted: %q", out)
	}
	if !strings.Contains(out, "****") {
		t.Errorf("expected redaction marker: %q", out)
	}
}

func TestConfigSetInvalidPair(t *testing.T) {
	path := filepath.Join(t.TempDir(), "cfg.yaml")
	if _, err := execute(t, "config", "set", "noequalsign", "--config", path); err == nil {
		t.Fatal("expected error for malformed pair")
	}
}

func TestConfigSetInvalidValue(t *testing.T) {
	path := filepath.Join(t.TempDir(), "cfg.yaml")
	if _, err := execute(t, "config", "set", "provider_type=grpc", "--config", path); err == nil {
		t.Fatal("expected validation error")
	}
}

func TestSetupWizard(t *testing.T) {
	path := filepath.Join(t.TempDir(), "cfg.yaml")
	// Provider, provider_type, api_url, api_key, model. Blank lines keep defaults.
	input := strings.NewReader("anthropic\nanthropic_compatible\n\nsk-key\nclaude-3-5-sonnet\n")
	var out bytes.Buffer
	if err := runSetupWizard(input, &out, path); err != nil {
		t.Fatalf("wizard: %v", err)
	}
	if !strings.Contains(out.String(), "Configuration saved") {
		t.Errorf("wizard output: %q", out.String())
	}

	got, err := execute(t, "config", "get", "ai_provider", "model", "--config", path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(got, "ai_provider=anthropic") || !strings.Contains(got, "model=claude-3-5-sonnet") {
		t.Errorf("wizard did not persist values: %q", got)
	}
}
