package cmd

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"
)

func TestCommitlintDisabledByDefault(t *testing.T) {
	path := filepath.Join(t.TempDir(), "cfg.yaml")
	var buf bytes.Buffer
	if err := runCommitlint(&buf, path); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	if !strings.Contains(out, "disabled") {
		t.Errorf("expected disabled by default: %q", out)
	}
	if !strings.Contains(out, "feat, fix") {
		t.Errorf("expected rule text: %q", out)
	}
	if !strings.Contains(out, "prompt_module=@commitlint") {
		t.Errorf("expected enable hint: %q", out)
	}
}

func TestCommitlintViaCommand(t *testing.T) {
	path := filepath.Join(t.TempDir(), "cfg.yaml")
	out, err := execute(t, "commitlint", "--config", path)
	if err != nil {
		t.Fatalf("commitlint cmd: %v", err)
	}
	if !strings.Contains(out, "commitlint injection") {
		t.Errorf("unexpected output: %q", out)
	}
}

func TestCommitlintEnabled(t *testing.T) {
	path := filepath.Join(t.TempDir(), "cfg.yaml")
	if _, err := execute(t, "config", "set", "prompt_module=@commitlint", "--config", path); err != nil {
		t.Fatal(err)
	}
	var buf bytes.Buffer
	if err := runCommitlint(&buf, path); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), "enabled") {
		t.Errorf("expected enabled: %q", buf.String())
	}
}
