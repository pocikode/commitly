// Package tokens provides approximate token estimation and input-budget
// handling for diffs. Exactness is not critical: we use a ~4
// chars/token heuristic to enforce tokens_max_input.
package tokens

import "strings"

// charsPerToken is the rough average characters per token for English/code.
const charsPerToken = 4

// Estimate returns an approximate token count for text.
func Estimate(text string) int {
	if text == "" {
		return 0
	}
	return (len(text) + charsPerToken - 1) / charsPerToken
}

// Fits reports whether text is within the token budget.
func Fits(text string, maxTokens int) bool {
	return Estimate(text) <= maxTokens
}

// SplitByFile breaks a unified git diff into per-file segments, splitting on
// the "diff --git" header. Any content before the first header is attached to
// the first segment. A diff with no headers returns a single segment.
func SplitByFile(diff string) []string {
	const marker = "diff --git "
	idx := strings.Index(diff, marker)
	if idx == -1 {
		if strings.TrimSpace(diff) == "" {
			return nil
		}
		return []string{diff}
	}

	var segs []string
	rest := diff[idx:]
	for {
		next := strings.Index(rest[len(marker):], marker)
		if next == -1 {
			segs = append(segs, rest)
			break
		}
		cut := len(marker) + next
		segs = append(segs, rest[:cut])
		rest = rest[cut:]
	}
	return segs
}

// Truncate cuts text to fit maxTokens, appending a notice. It returns the
// (possibly shortened) text and whether truncation occurred.
func Truncate(text string, maxTokens int) (string, bool) {
	if Fits(text, maxTokens) {
		return text, false
	}
	const notice = "\n... [diff truncated to fit token budget]"
	limit := maxTokens * charsPerToken
	if limit > len(notice) {
		limit -= len(notice)
	}
	if limit < 0 {
		limit = 0
	}
	if limit > len(text) {
		limit = len(text)
	}
	return text[:limit] + notice, true
}

// FitDiff enforces maxTokens on a diff: keep whole files that fit, dropping
// later files that would overflow the budget; if even the first file
// overflows, truncate it. It returns the budget-fitted diff and whether any
// content was dropped or truncated.
func FitDiff(diff string, maxTokens int) (string, bool) {
	if Fits(diff, maxTokens) {
		return diff, false
	}

	segs := SplitByFile(diff)
	if len(segs) <= 1 {
		return Truncate(diff, maxTokens)
	}

	var b strings.Builder
	used := 0
	dropped := false
	for _, seg := range segs {
		t := Estimate(seg)
		if used+t > maxTokens {
			if used == 0 {
				// First file alone overflows: truncate it.
				out, _ := Truncate(seg, maxTokens)
				return out, true
			}
			dropped = true
			continue
		}
		b.WriteString(seg)
		used += t
	}
	return b.String(), dropped
}
