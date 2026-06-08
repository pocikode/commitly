package cmd

import (
	"bytes"
	"context"
	"path/filepath"
	"strings"
	"testing"
)

func TestModelsListPlain(t *testing.T) {
	path := filepath.Join(t.TempDir(), "cfg.yaml")
	// openai preset, default model gpt-4o; offline so only static list.
	var buf bytes.Buffer
	if err := runModels(context.Background(), &buf, path, true, ""); err != nil {
		t.Fatalf("models: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "gpt-4o") {
		t.Errorf("expected preset model listed: %q", out)
	}
	if !strings.Contains(out, "* gpt-4o") {
		t.Errorf("current model should be marked: %q", out)
	}
}

func TestModelsSetPersists(t *testing.T) {
	path := filepath.Join(t.TempDir(), "cfg.yaml")
	var buf bytes.Buffer
	if err := runModels(context.Background(), &buf, path, true, "gpt-4o-mini"); err != nil {
		t.Fatalf("models set: %v", err)
	}
	if !strings.Contains(buf.String(), "Model set to gpt-4o-mini") {
		t.Errorf("unexpected output: %q", buf.String())
	}

	// Verify persisted.
	got, err := execute(t, "config", "get", "model", "--config", path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(got, "model=gpt-4o-mini") {
		t.Errorf("model not persisted: %q", got)
	}
}

func TestModelsViaCommand(t *testing.T) {
	path := filepath.Join(t.TempDir(), "cfg.yaml")
	// --yes makes it non-interactive (list only, no select prompt).
	out, err := execute(t, "models", "--config", path, "--yes")
	if err != nil {
		t.Fatalf("models cmd: %v", err)
	}
	if !strings.Contains(out, "gpt-4o") {
		t.Errorf("expected model list: %q", out)
	}
}

func TestModelsSetViaCommand(t *testing.T) {
	path := filepath.Join(t.TempDir(), "cfg.yaml")
	out, err := execute(t, "models", "--set", "gpt-4o-mini", "--config", path)
	if err != nil {
		t.Fatalf("models --set: %v", err)
	}
	if !strings.Contains(out, "Model set to gpt-4o-mini") {
		t.Errorf("unexpected: %q", out)
	}
}

func TestModelsSetUnknownWarns(t *testing.T) {
	path := filepath.Join(t.TempDir(), "cfg.yaml")
	var buf bytes.Buffer
	if err := runModels(context.Background(), &buf, path, true, "made-up-model"); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), "not in the known model list") {
		t.Errorf("expected warning for unknown model: %q", buf.String())
	}
}
