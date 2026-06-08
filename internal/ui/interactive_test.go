package ui

import (
	"bytes"
	"strings"
	"testing"
)

func TestInteractivePreviewRendersMessage(t *testing.T) {
	var buf bytes.Buffer
	i := NewInteractive(&buf)
	i.Preview("feat: add thing")
	out := buf.String()
	if !strings.Contains(out, "feat: add thing") {
		t.Errorf("preview missing message: %q", out)
	}
	if !strings.Contains(out, "generated commit") {
		t.Errorf("preview missing label: %q", out)
	}
}

func TestInteractiveInfoRendersMessage(t *testing.T) {
	var buf bytes.Buffer
	i := NewInteractive(&buf)
	i.Info("committed")
	if !strings.Contains(buf.String(), "committed") {
		t.Errorf("info missing message: %q", buf.String())
	}
}
