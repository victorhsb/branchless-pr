package git

import (
	"testing"

	"github.com/victorhsb/branchless-pr/internal/shell/shelltest"
)

func TestIsFullSHA(t *testing.T) {
	cases := []struct {
		in   string
		want bool
	}{
		{"0123456789abcdef0123456789abcdef01234567", true},
		{"0123456789ABCDEF0123456789abcdef01234567", false},
		{"short", false},
		{"", false},
		{"0123456789abcdef0123456789abcdef0123456g", false},
	}
	for _, c := range cases {
		if got := IsFullSHA(c.in); got != c.want {
			t.Errorf("IsFullSHA(%q) = %v, want %v", c.in, got, c.want)
		}
	}
}

func TestBranchlessStackHeadReturnsTopCommit(t *testing.T) {
	const bottom = "1111111111111111111111111111111111111111"
	const top = "2222222222222222222222222222222222222222"
	run := shelltest.New(t, shelltest.Response{
		Match:  shelltest.Exact("git", "branchless", "query", "-r", "stack()"),
		Stdout: bottom + "\n" + top + "\n",
	})

	got, ok := New("", run).BranchlessStackHead()
	if !ok {
		t.Fatalf("expected branchless stack head")
	}
	if got != top {
		t.Fatalf("BranchlessStackHead = %q, want %q", got, top)
	}
}

func TestBranchlessStackHeadReturnsFalseWhenUnavailable(t *testing.T) {
	run := shelltest.New(t, shelltest.Response{
		Match:    shelltest.Exact("git", "branchless", "query", "-r", "stack()"),
		ExitCode: 1,
	})

	if got, ok := New("", run).BranchlessStackHead(); ok || got != "" {
		t.Fatalf("BranchlessStackHead = %q, %v; want empty, false", got, ok)
	}
}

func TestRebasePlacesExtrasBeforeUpstream(t *testing.T) {
	run := shelltest.New(t, shelltest.Response{
		Match: shelltest.Exact("git", "rebase", "--committer-date-is-author-date", "main", "feature"),
	})
	if err := New("", run).Rebase("main", "feature", "--committer-date-is-author-date"); err != nil {
		t.Fatal(err)
	}
}
