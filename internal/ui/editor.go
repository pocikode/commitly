package ui

import (
	"os"
	"os/exec"
	"strings"

	"github.com/charmbracelet/huh"
)

// editorRunner runs an editor command against a file path. Overridable in
// tests; defaults to running the resolved editor attached to the terminal.
var editorRunner = func(editor, path string) error {
	parts := strings.Fields(editor)
	parts = append(parts, path)
	cmd := exec.Command(parts[0], parts[1:]...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// resolveEditor returns the user's editor from $VISUAL or $EDITOR, or "" if
// neither is set.
func resolveEditor() string {
	if v := os.Getenv("VISUAL"); v != "" {
		return v
	}
	return os.Getenv("EDITOR")
}

// EditMessage edits a commit message. It opens $VISUAL/$EDITOR on a temp file
// when available; otherwise it falls back to
// an inline TUI text area.
func EditMessage(message string) (string, error) {
	editor := resolveEditor()
	if editor == "" {
		return inlineEdit(message)
	}
	return externalEdit(editor, message)
}

// externalEdit writes message to a temp file, opens it in editor, and returns
// the edited contents.
func externalEdit(editor, message string) (string, error) {
	f, err := os.CreateTemp("", "cly-commit-*.txt")
	if err != nil {
		return "", err
	}
	path := f.Name()
	defer os.Remove(path)

	if _, err := f.WriteString(message); err != nil {
		f.Close()
		return "", err
	}
	if err := f.Close(); err != nil {
		return "", err
	}

	if err := editorRunner(editor, path); err != nil {
		return "", err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return strings.TrimRight(string(data), "\n"), nil
}

// inlineEdit edits the message via a TUI text area.
func inlineEdit(message string) (string, error) {
	edited := message
	err := huh.NewText().
		Title("Edit commit message").
		Value(&edited).
		Run()
	if err != nil {
		return message, nil
	}
	return strings.TrimRight(edited, "\n"), nil
}
