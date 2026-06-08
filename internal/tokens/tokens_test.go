package tokens

import (
	"strings"
	"testing"
)

func TestEstimate(t *testing.T) {
	if Estimate("") != 0 {
		t.Error("empty should be 0 tokens")
	}
	// 8 chars -> 2 tokens at 4 chars/token.
	if got := Estimate("abcdefgh"); got != 2 {
		t.Errorf("Estimate = %d, want 2", got)
	}
	// Rounds up.
	if got := Estimate("abcde"); got != 2 {
		t.Errorf("Estimate = %d, want 2", got)
	}
}

func TestFits(t *testing.T) {
	if !Fits("abcd", 1) {
		t.Error("4 chars should fit in 1 token")
	}
	if Fits("abcdefghi", 1) {
		t.Error("9 chars should not fit in 1 token")
	}
}

func makeDiff(files ...string) string {
	var b strings.Builder
	for _, f := range files {
		b.WriteString("diff --git a/" + f + " b/" + f + "\n")
		b.WriteString("--- a/" + f + "\n+++ b/" + f + "\n")
		b.WriteString("+some added line content here\n")
	}
	return b.String()
}

func TestSplitByFile(t *testing.T) {
	diff := makeDiff("a.go", "b.go", "c.go")
	segs := SplitByFile(diff)
	if len(segs) != 3 {
		t.Fatalf("got %d segments, want 3", len(segs))
	}
	for i, want := range []string{"a.go", "b.go", "c.go"} {
		if !strings.Contains(segs[i], want) {
			t.Errorf("segment %d missing %q: %q", i, want, segs[i])
		}
	}
}

func TestSplitByFileNoHeader(t *testing.T) {
	if segs := SplitByFile("just some text"); len(segs) != 1 {
		t.Errorf("want 1 segment, got %d", len(segs))
	}
	if segs := SplitByFile("   "); segs != nil {
		t.Errorf("blank should give nil, got %v", segs)
	}
}

func TestTruncate(t *testing.T) {
	text := strings.Repeat("x", 400) // 100 tokens
	out, truncated := Truncate(text, 10)
	if !truncated {
		t.Fatal("expected truncation")
	}
	if !strings.Contains(out, "truncated") {
		t.Errorf("expected notice in output")
	}
	// Within budget -> untouched.
	out, truncated = Truncate("short", 100)
	if truncated || out != "short" {
		t.Errorf("short text should be untouched")
	}
}

func TestFitDiffWholeFits(t *testing.T) {
	diff := makeDiff("a.go")
	out, changed := FitDiff(diff, 10_000)
	if changed || out != diff {
		t.Errorf("diff within budget should be untouched")
	}
}

func TestFitDiffDropsLaterFiles(t *testing.T) {
	diff := makeDiff("a.go", "b.go", "c.go")
	// Budget big enough for one segment but not all three.
	per := Estimate(SplitByFile(diff)[0])
	out, changed := FitDiff(diff, per+1)
	if !changed {
		t.Fatal("expected files to be dropped")
	}
	if !strings.Contains(out, "a.go") {
		t.Errorf("first file should be kept: %q", out)
	}
	if strings.Contains(out, "c.go") {
		t.Errorf("later files should be dropped: %q", out)
	}
}

func TestFitDiffTruncatesSingleHugeFile(t *testing.T) {
	// One file that alone overflows the budget.
	big := "diff --git a/x b/x\n" + strings.Repeat("+line of content\n", 200)
	out, changed := FitDiff(big, 5)
	if !changed {
		t.Fatal("expected truncation")
	}
	if Estimate(out) > 5+Estimate("\n... [diff truncated to fit token budget]") {
		t.Errorf("output not bounded by budget: %d tokens", Estimate(out))
	}
}
