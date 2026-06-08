package prompt

import "strings"

// commitlintRules is a simplified, opinionated subset of the common
// @commitlint/config-conventional ruleset (simplified subset, not JS-config
// parsing). Injected into the system prompt so generated messages pass typical
// commit linting.
var commitlintRules = []string{
	"The commit message MUST follow these commitlint rules:",
	"- type must be one of: feat, fix, build, chore, ci, docs, style, refactor, perf, test, revert.",
	"- type and scope must be lower-case.",
	"- the subject must not be empty and must not end with a period.",
	"- the header (type(scope): subject) must not exceed 100 characters.",
	"- use the imperative mood in the subject (e.g. \"add\" not \"added\" or \"adds\").",
}

// CommitlintRules returns the simplified commitlint rule block injected into the
// prompt and shown by `oco commitlint`.
func CommitlintRules() string {
	return strings.Join(commitlintRules, "\n")
}
