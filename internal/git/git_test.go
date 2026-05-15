package git

import (
	"os"
	"path/filepath"
	"testing"
)

func TestIsFullSHA(t *testing.T) {
	cases := []struct {
		in   string
		want bool
	}{
		{"0123456789abcdef0123456789abcdef01234567", true},
		{"0123456789ABCDEF0123456789abcdef01234567", false}, // upper-case rejected
		{"short", false},
		{"", false},
		{"0123456789abcdef0123456789abcdef0123456g", false}, // non-hex
	}
	for _, c := range cases {
		if got := IsFullSHA(c.in); got != c.want {
			t.Errorf("IsFullSHA(%q) = %v, want %v", c.in, got, c.want)
		}
	}
}

func TestUsernameOverride(t *testing.T) {
	u := "TestBot"
	DefaultConfig().SetUsernameOverride(&u)
	t.Cleanup(func() { DefaultConfig().SetUsernameOverride(nil) })

	got, err := GetGHUsername()
	if err != nil {
		t.Fatalf("unexpected: %v", err)
	}
	if got != u {
		t.Fatalf("expected %q, got %q", u, got)
	}
}

func TestBranchlessStackHeadReturnsTopCommit(t *testing.T) {
	bin := t.TempDir()
	fakeGit := filepath.Join(bin, "git")
	const bottom = "1111111111111111111111111111111111111111"
	const top = "2222222222222222222222222222222222222222"
	script := "#!/bin/sh\n" +
		"if [ \"$1\" = branchless ] && [ \"$2\" = query ] && [ \"$3\" = -r ] && [ \"$4\" = 'stack()' ]; then\n" +
		"  printf '%s\\n%s\\n' " + bottom + " " + top + "\n" +
		"  exit 0\n" +
		"fi\n" +
		"exit 1\n"
	if err := os.WriteFile(fakeGit, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))

	got, ok := BranchlessStackHead()
	if !ok {
		t.Fatalf("expected branchless stack head")
	}
	if got != top {
		t.Fatalf("BranchlessStackHead = %q, want %q", got, top)
	}
}

func TestBranchlessStackHeadReturnsFalseWhenUnavailable(t *testing.T) {
	bin := t.TempDir()
	fakeGit := filepath.Join(bin, "git")
	if err := os.WriteFile(fakeGit, []byte("#!/bin/sh\nexit 1\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))

	if got, ok := BranchlessStackHead(); ok || got != "" {
		t.Fatalf("BranchlessStackHead = %q, %v; want empty, false", got, ok)
	}
}

func TestIsRebaseInProgressDetectsRebaseMerge(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, ".git", "rebase-merge"), 0o755); err != nil {
		t.Fatal(err)
	}
	// Need a fake repo root resolved from this dir. We can't easily without
	// initializing a real git repo, so just check the function reads from
	// disk relative to RepoRoot - which falls back to false.
	// Instead, initialize a tiny repo so RepoRoot works.
	if err := os.WriteFile(filepath.Join(dir, ".git", "HEAD"), []byte("ref: refs/heads/main\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	cwd, _ := os.Getwd()
	t.Cleanup(func() { _ = os.Chdir(cwd) })
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}

	if !IsRebaseInProgress(dir) {
		t.Fatalf("expected rebase-in-progress to be detected when .git/rebase-merge exists")
	}
}

func TestIsRebaseInProgressDetectsRebaseApply(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, ".git", "rebase-apply"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, ".git", "HEAD"), []byte("ref: refs/heads/main\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	cwd, _ := os.Getwd()
	t.Cleanup(func() { _ = os.Chdir(cwd) })
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}

	if !IsRebaseInProgress(dir) {
		t.Fatalf("expected rebase-in-progress to be detected when .git/rebase-apply exists")
	}
}
