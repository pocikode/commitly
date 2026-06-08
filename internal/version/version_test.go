package version

import (
	"strings"
	"testing"
)

func TestString(t *testing.T) {
	Version = "v1.2.3"
	Commit = "abc1234"
	Date = "2026-01-01"
	s := String()
	for _, want := range []string{"v1.2.3", "abc1234", "2026-01-01"} {
		if !strings.Contains(s, want) {
			t.Errorf("String() = %q, missing %q", s, want)
		}
	}
}

func TestStringDefaults(t *testing.T) {
	Version, Commit, Date = "dev", "none", "unknown"
	if !strings.HasPrefix(String(), "dev ") {
		t.Errorf("default String() = %q", String())
	}
}
