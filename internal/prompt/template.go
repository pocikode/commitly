package prompt

import (
	"os"
	"strings"
)

// builtInModules are recognized prompt_module names that use the built-in
// system prompt rather than a file override.
var builtInModules = map[string]bool{
	"":                    true,
	"conventional-commit": true,
	"@commitlint":         true,
}

// LoadOverride resolves a prompt_module value into a custom system-prompt
// override. Built-in module names yield ("", false, nil) — the caller then uses
// the config-driven System prompt. Any other value is treated as a path to a
// template file whose contents become the override. A missing file is an error
// so misconfiguration is surfaced rather than silently ignored.
func LoadOverride(promptModule string) (string, bool, error) {
	name := strings.TrimSpace(promptModule)
	if builtInModules[name] {
		return "", false, nil
	}
	data, err := os.ReadFile(name)
	if err != nil {
		return "", false, err
	}
	return string(data), true, nil
}

// ApplyTemplatePlaceholder substitutes the message_template_placeholder in a
// commit template with the generated message. When the placeholder is absent
// from the template, the message is returned as-is (no template in effect).
// Used by the git hook to populate a user's commit-message template.
func ApplyTemplatePlaceholder(template, message, placeholder string) string {
	if placeholder == "" || !strings.Contains(template, placeholder) {
		return message
	}
	return strings.ReplaceAll(template, placeholder, message)
}
