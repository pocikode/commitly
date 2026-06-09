package git

import (
	"context"
	"os"
	"path"
	"path/filepath"
	"strings"
)

// IgnoreFileName is the per-repo ignore list for diff filtering.
const IgnoreFileName = ".commitlyignore"

// DefaultLockFiles are dependency lock files always excluded from the diff sent
// to the LLM: large, noisy, and low-signal.
var DefaultLockFiles = []string{
	"package-lock.json",
	"yarn.lock",
	"pnpm-lock.yaml",
	"bun.lockb",
	"go.sum",
	"Cargo.lock",
	"composer.lock",
	"Gemfile.lock",
	"poetry.lock",
	"Pipfile.lock",
	"flake.lock",
}

// Ignore decides which staged paths to drop from the diff.
type Ignore struct {
	patterns  []string
	lockFiles map[string]bool
}

// LoadIgnore builds an Ignore from the repo's .commitlyignore (if present)
// plus the default lock-file set. A missing ignore file is not an error.
func LoadIgnore(dir string) (*Ignore, error) {
	ig := &Ignore{lockFiles: map[string]bool{}}
	for _, f := range DefaultLockFiles {
		ig.lockFiles[f] = true
	}

	data, err := os.ReadFile(filepath.Join(dir, IgnoreFileName))
	if err != nil {
		if os.IsNotExist(err) {
			return ig, nil
		}
		return nil, err
	}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		ig.patterns = append(ig.patterns, line)
	}
	return ig, nil
}

// Match reports whether a path should be excluded from the diff.
func (i *Ignore) Match(p string) bool {
	base := path.Base(p)
	if i.lockFiles[base] {
		return true
	}
	for _, pat := range i.patterns {
		// Match against both the full path and the basename so patterns like
		// "*.snap" or "dist/*" both work.
		if ok, _ := path.Match(pat, p); ok {
			return true
		}
		if ok, _ := path.Match(pat, base); ok {
			return true
		}
		// Directory prefix pattern, e.g. "vendor/" or "vendor".
		dir := strings.TrimSuffix(pat, "/")
		if p == dir || strings.HasPrefix(p, dir+"/") {
			return true
		}
	}
	return false
}

// Filter returns the subset of paths not matched by the ignore rules.
func (i *Ignore) Filter(paths []string) []string {
	out := make([]string, 0, len(paths))
	for _, p := range paths {
		if !i.Match(p) {
			out = append(out, p)
		}
	}
	return out
}

// DiffFiltered returns the staged diff with ignored/lock files removed, along
// with the list of files actually included.
func (g *Git) DiffFiltered(ctx context.Context, ig *Ignore) (string, []string, error) {
	files, err := g.StagedFiles(ctx)
	if err != nil {
		return "", nil, err
	}
	kept := ig.Filter(files)
	if len(kept) == 0 {
		return "", nil, nil
	}
	diff, err := g.StagedDiff(ctx, kept...)
	if err != nil {
		return "", nil, err
	}
	return diff, kept, nil
}
