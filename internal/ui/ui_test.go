package ui

import (
	"bytes"
	"context"
	"strings"
	"testing"
)

func TestPlainAutoConfirm(t *testing.T) {
	var buf bytes.Buffer
	p := NewPlain(&buf, true)

	a, err := p.Menu()
	if err != nil || a != ActionConfirm {
		t.Errorf("auto-confirm Menu = %v, %v", a, err)
	}
	ok, _ := p.Confirm("push?")
	if !ok {
		t.Error("auto-confirm Confirm should be true")
	}
}

func TestPlainNoAutoConfirm(t *testing.T) {
	var buf bytes.Buffer
	p := NewPlain(&buf, false)
	if a, _ := p.Menu(); a != ActionCancel {
		t.Errorf("non-auto Menu should cancel, got %v", a)
	}
	if ok, _ := p.Confirm("?"); ok {
		t.Error("non-auto Confirm should be false")
	}
}

func TestPlainPreviewAndSpinner(t *testing.T) {
	var buf bytes.Buffer
	p := NewPlain(&buf, true)

	got, err := p.Spinner(context.Background(), "working", func() (string, error) {
		return "result", nil
	})
	if err != nil || got != "result" {
		t.Fatalf("spinner = %q, %v", got, err)
	}
	p.Preview("feat: hello")
	p.Info("done")
	p.Edit("unchanged") // no-op

	out := buf.String()
	for _, want := range []string{"working", "feat: hello", "done"} {
		if !strings.Contains(out, want) {
			t.Errorf("output missing %q: %q", want, out)
		}
	}
}

func TestPlainEditReturnsUnchanged(t *testing.T) {
	p := NewPlain(&bytes.Buffer{}, true)
	got, err := p.Edit("feat: x")
	if err != nil || got != "feat: x" {
		t.Errorf("Edit = %q, %v", got, err)
	}
}

func TestSelectReturnsPlainOnYes(t *testing.T) {
	u := Select(true, &bytes.Buffer{})
	if _, ok := u.(*Plain); !ok {
		t.Errorf("--yes should yield *Plain, got %T", u)
	}
}

func TestSelectReturnsPlainWhenNoColor(t *testing.T) {
	t.Setenv("NO_COLOR", "1")
	u := Select(false, &bytes.Buffer{})
	if _, ok := u.(*Plain); !ok {
		t.Errorf("NO_COLOR should yield *Plain, got %T", u)
	}
}
