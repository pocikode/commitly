package ui

import (
	"context"
	"fmt"
	"io"

	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/huh/spinner"
	"github.com/charmbracelet/lipgloss"
)

// Interactive is the rich TUI: spinner during generation, colored preview, and
// a select menu for confirm/edit/regenerate/cancel.
type Interactive struct {
	out io.Writer
}

// NewInteractive builds an interactive UI writing to out.
func NewInteractive(out io.Writer) *Interactive {
	return &Interactive{out: out}
}

var (
	previewBorder = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("63")).
			Padding(0, 1)
	previewLabel = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("42"))
	infoStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
)

func (i *Interactive) Spinner(ctx context.Context, label string, fn func() (string, error)) (string, error) {
	var (
		result string
		runErr error
	)
	err := spinner.New().
		Title(" " + label).
		Context(ctx).
		Action(func() { result, runErr = fn() }).
		Run()
	if err != nil {
		return "", err
	}
	return result, runErr
}

func (i *Interactive) Preview(message string) {
	fmt.Fprintln(i.out, previewLabel.Render("◇ generated commit:"))
	fmt.Fprintln(i.out, previewBorder.Render(message))
}

func (i *Interactive) Menu() (Action, error) {
	var choice Action
	err := huh.NewSelect[Action]().
		Title("What now?").
		Options(
			huh.NewOption("Confirm — commit as-is", ActionConfirm),
			huh.NewOption("Edit — change the message", ActionEdit),
			huh.NewOption("Regenerate — ask the model again", ActionRegenerate),
			huh.NewOption("Cancel — abort", ActionCancel),
		).
		Value(&choice).
		Run()
	if err != nil {
		// huh returns ErrUserAborted on Ctrl-C; treat as cancel.
		return ActionCancel, nil
	}
	return choice, nil
}

func (i *Interactive) Edit(message string) (string, error) {
	return EditMessage(message)
}

func (i *Interactive) Confirm(prompt string) (bool, error) {
	var ok bool
	err := huh.NewConfirm().Title(prompt).Value(&ok).Run()
	if err != nil {
		return false, nil
	}
	return ok, nil
}

func (i *Interactive) Info(message string) {
	fmt.Fprintln(i.out, infoStyle.Render(message))
}
