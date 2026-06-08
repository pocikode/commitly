// Package ui provides the interactive terminal experience for the commit flow
// (spinner, colored preview, confirm/edit/regenerate/cancel menu) plus a plain
// non-interactive fallback for --yes, non-TTY, and NO_COLOR environments.
package ui

import (
	"context"
	"fmt"
	"io"
	"os"

	"github.com/mattn/go-isatty"
)

// Action is the user's choice from the commit menu.
type Action int

const (
	ActionConfirm Action = iota
	ActionEdit
	ActionRegenerate
	ActionCancel
)

// UI is the interaction surface used by the commit flow. It is an interface so
// the flow can be driven by a real TUI, a plain fallback, or a test double.
type UI interface {
	// Spinner runs fn while displaying label, returning fn's result.
	Spinner(ctx context.Context, label string, fn func() (string, error)) (string, error)
	// Preview renders the generated commit message.
	Preview(message string)
	// Menu asks the user what to do with the message.
	Menu() (Action, error)
	// Edit lets the user modify the message and returns the result.
	Edit(message string) (string, error)
	// Confirm asks a yes/no question.
	Confirm(prompt string) (bool, error)
	// Info prints an informational line.
	Info(message string)
}

// Select chooses the appropriate UI based on flags and environment:
// --yes, non-TTY, or NO_COLOR fall back to plain auto-confirm output.
func Select(yes bool, out io.Writer) UI {
	if yes || !isInteractive() {
		return NewPlain(out, true)
	}
	return NewInteractive(out)
}

// isInteractive reports whether the process is attached to a terminal and color
// is permitted.
func isInteractive() bool {
	if _, ok := os.LookupEnv("NO_COLOR"); ok {
		return false
	}
	return isatty.IsTerminal(os.Stdout.Fd())
}

// Plain is a non-interactive UI: it prints output and auto-confirms. Used for
// --yes, scripts, hooks, non-TTY, and NO_COLOR.
type Plain struct {
	out         io.Writer
	autoConfirm bool
}

// NewPlain builds a plain UI. When autoConfirm is true, Menu confirms and
// Confirm returns true without prompting.
func NewPlain(out io.Writer, autoConfirm bool) *Plain {
	return &Plain{out: out, autoConfirm: autoConfirm}
}

func (p *Plain) Spinner(ctx context.Context, label string, fn func() (string, error)) (string, error) {
	fmt.Fprintln(p.out, label)
	return fn()
}

func (p *Plain) Preview(message string) {
	fmt.Fprintf(p.out, "\nGenerated commit message:\n%s\n\n", message)
}

func (p *Plain) Menu() (Action, error) {
	if p.autoConfirm {
		return ActionConfirm, nil
	}
	return ActionCancel, nil
}

func (p *Plain) Edit(message string) (string, error) { return message, nil }

func (p *Plain) Confirm(prompt string) (bool, error) { return p.autoConfirm, nil }

func (p *Plain) Info(message string) { fmt.Fprintln(p.out, message) }
