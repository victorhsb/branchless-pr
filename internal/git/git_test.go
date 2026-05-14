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
