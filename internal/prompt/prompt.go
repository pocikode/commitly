// Package prompt builds the system/user prompts sent to the AI provider and
// post-processes the model's reply. The prompt intent is ported from the
// original opencommit (src/prompts.ts).
package prompt

import (
	"strings"

	"github.com/pocikode/opencommit/internal/config"
)

// identity opens the system prompt, framing the model's role.
const identity = "You are to act as the author of a commit message in git."

// gitmojiHelp is the trimmed gitmoji guide used when emoji is enabled.
const gitmojiHelp = `Use the GitMoji convention to preface the commit. Choose the right emoji (emoji, description):
🐛 Fix a bug; ✨ Introduce new features; 📝 Add or update documentation; 🚀 Deploy stuff; ✅ Add or update tests; ♻️ Refactor code; ⬆️ Upgrade dependencies; 🔧 Add or update configuration; 💡 Add or update comments; ⚡️ Improve performance; 🔥 Remove code or files.`

// conventionalKeywords is used when emoji is disabled.
const conventionalKeywords = "Do not preface the commit with anything except the conventional commit keywords: fix, feat, build, chore, ci, docs, style, refactor, perf, test."

// Options control prompt assembly beyond what config provides.
type Options struct {
	// Context is optional extra user-provided context for the commit.
	Context string
	// Override, when non-empty, replaces the entire built system prompt
	// (custom prompt module / template).
	Override string
}

// System builds the system prompt from config + options. When an override is
// set it is returned verbatim (still allowing the diff to be sent as the user
// message).
func System(cfg config.Config, opts Options) string {
	if strings.TrimSpace(opts.Override) != "" {
		return strings.TrimSpace(opts.Override)
	}

	convention := "Conventional Commit Convention"
	mission := identity + " Your mission is to create a clean, comprehensive commit message per the " +
		convention + " and explain WHAT changed and mainly WHY."
	diffInstruction := "I'll send you the output of 'git diff --staged' and you convert it into a commit message."

	lines := []string{mission, diffInstruction}

	if cfg.Emoji {
		lines = append(lines, gitmojiHelp)
	} else {
		lines = append(lines, conventionalKeywords)
	}

	if cfg.Description {
		lines = append(lines, "Add a short description of WHY the changes were made after the commit message. Don't start it with \"This commit\", just describe the changes.")
	} else {
		lines = append(lines, "Don't add any description to the commit, only the commit message.")
	}

	if cfg.OneLineCommit {
		lines = append(lines, "Craft a concise, single-sentence commit message that captures all changes, emphasizing the primary update. Provide a clear, unified overview in one message.")
	}

	if cfg.OmitScope {
		lines = append(lines, "Do not include a scope in the commit message. Use the format: <type>: <subject>")
	} else {
		lines = append(lines, "Use the Conventional Commits format: <type>(<scope>): <subject>")
	}

	lines = append(lines, "Use the present tense. Lines must not be longer than 74 characters. Use English for the commit message.")

	if ctx := strings.TrimSpace(opts.Context); ctx != "" {
		lines = append(lines, "Additional context provided by the user: <context>"+ctx+"</context>. Consider this context when generating the commit message.")
	}

	return strings.Join(lines, "\n")
}

// Clean post-processes a raw model reply: it strips surrounding code fences and
// backticks, trims whitespace, and collapses to a single line when oneLine is
// set.
func Clean(raw string, oneLine bool) string {
	s := strings.TrimSpace(raw)
	s = stripCodeFence(s)
	s = strings.Trim(s, "`")
	s = strings.TrimSpace(s)

	if oneLine {
		if i := strings.IndexByte(s, '\n'); i >= 0 {
			s = s[:i]
		}
		s = strings.TrimSpace(s)
	}
	return s
}

// stripCodeFence removes a leading ```lang fence and trailing ``` fence if the
// message is wrapped in one.
func stripCodeFence(s string) string {
	if !strings.HasPrefix(s, "```") {
		return s
	}
	// Drop the first line (```lang) and a trailing ``` line.
	lines := strings.Split(s, "\n")
	if len(lines) < 2 {
		return s
	}
	lines = lines[1:]
	for len(lines) > 0 && strings.TrimSpace(lines[len(lines)-1]) == "```" {
		lines = lines[:len(lines)-1]
	}
	return strings.TrimSpace(strings.Join(lines, "\n"))
}
