package prompt

import (
	"strings"
	"testing"

	"github.com/pocikode/commitly/internal/config"
)

func TestSystemDefault(t *testing.T) {
	s := System(config.Defaults(), Options{})
	if !strings.Contains(s, "Conventional Commit") {
		t.Error("expected conventional commit mention")
	}
	// Default: no emoji, no description.
	if !strings.Contains(s, conventionalKeywords) {
		t.Error("expected conventional keywords when emoji off")
	}
	if !strings.Contains(s, "only the commit message") {
		t.Error("expected no-description instruction by default")
	}
	if !strings.Contains(s, "<type>(<scope>)") {
		t.Error("expected scope format by default")
	}
}

func TestSystemEmoji(t *testing.T) {
	cfg := config.Defaults()
	cfg.Emoji = true
	s := System(cfg, Options{})
	if !strings.Contains(s, "GitMoji") {
		t.Error("expected gitmoji guide when emoji enabled")
	}
	if strings.Contains(s, conventionalKeywords) {
		t.Error("should not include conventional keywords when emoji on")
	}
}

func TestSystemDescription(t *testing.T) {
	cfg := config.Defaults()
	cfg.Description = true
	s := System(cfg, Options{})
	if !strings.Contains(s, "WHY the changes were made") {
		t.Error("expected description instruction")
	}
}

func TestSystemOmitScope(t *testing.T) {
	cfg := config.Defaults()
	cfg.OmitScope = true
	s := System(cfg, Options{})
	if !strings.Contains(s, "<type>: <subject>") {
		t.Error("expected scope-less format")
	}
	if strings.Contains(s, "<type>(<scope>)") {
		t.Error("should not include scope format when omit_scope set")
	}
}

func TestSystemOneLine(t *testing.T) {
	cfg := config.Defaults()
	cfg.OneLineCommit = true
	s := System(cfg, Options{})
	if !strings.Contains(s, "single-sentence") {
		t.Error("expected one-line instruction")
	}
}

func TestSystemContext(t *testing.T) {
	s := System(config.Defaults(), Options{Context: "fixes JIRA-123"})
	if !strings.Contains(s, "<context>fixes JIRA-123</context>") {
		t.Error("expected user context embedded")
	}
}

func TestSystemOverride(t *testing.T) {
	s := System(config.Defaults(), Options{Override: "  custom prompt here  "})
	if s != "custom prompt here" {
		t.Errorf("override should be returned trimmed, got %q", s)
	}
}

func TestSystemCommitlintInjection(t *testing.T) {
	s := System(config.Defaults(), Options{Commitlint: true})
	if !strings.Contains(s, "commitlint rules") {
		t.Error("expected commitlint rules injected")
	}
	if !strings.Contains(s, "imperative mood") {
		t.Error("expected specific commitlint rule")
	}
	// Off by default.
	if strings.Contains(System(config.Defaults(), Options{}), "commitlint rules") {
		t.Error("commitlint rules should not appear unless enabled")
	}
}

func TestClean(t *testing.T) {
	cases := []struct {
		name    string
		in      string
		oneLine bool
		want    string
	}{
		{"trim", "  feat: x  ", false, "feat: x"},
		{"backticks", "`feat: x`", false, "feat: x"},
		{"fence", "```\nfeat: x\nbody line\n```", false, "feat: x\nbody line"},
		{"fence lang", "```text\nfeat: x\n```", false, "feat: x"},
		{"oneline collapse", "feat: x\n\nlong body", true, "feat: x"},
		{"oneline noop", "feat: x", true, "feat: x"},
	}
	for _, c := range cases {
		if got := Clean(c.in, c.oneLine); got != c.want {
			t.Errorf("%s: Clean(%q,%v) = %q, want %q", c.name, c.in, c.oneLine, got, c.want)
		}
	}
}
