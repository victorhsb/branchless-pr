package git

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/victorhsb/branchless-pr/internal/shell"
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

func TestForceUpdateBranchCreatesMissingBranch(t *testing.T) {
	repo := initTestRepo(t)
	sha := commitTestFile(t, repo, "one.txt", "one")
	withWorkingDir(t, repo)

	if err := ForceUpdateBranch("stack/one", sha); err != nil {
		t.Fatalf("ForceUpdateBranch returned error: %v", err)
	}
	got, err := RevParse("stack/one")
	if err != nil {
		t.Fatalf("RevParse returned error: %v", err)
	}
	if got != sha {
		t.Fatalf("stack/one = %s, want %s", got, sha)
	}
	if branch, err := CurrentBranchName(); err != nil || branch != "main" {
		t.Fatalf("current branch = %q, %v; want main", branch, err)
	}
}

func TestForceUpdateBranchResetsExistingBranch(t *testing.T) {
	repo := initTestRepo(t)
	oldSHA := commitTestFile(t, repo, "one.txt", "one")
	newSHA := commitTestFile(t, repo, "two.txt", "two")
	withWorkingDir(t, repo)

	if err := ForceUpdateBranch("stack/one", newSHA); err != nil {
		t.Fatalf("ForceUpdateBranch create returned error: %v", err)
	}
	if err := ForceUpdateBranch("stack/one", oldSHA); err != nil {
		t.Fatalf("ForceUpdateBranch reset returned error: %v", err)
	}
	got, err := RevParse("stack/one")
	if err != nil {
		t.Fatalf("RevParse returned error: %v", err)
	}
	if got != oldSHA {
		t.Fatalf("stack/one = %s, want %s", got, oldSHA)
	}
}

func TestForceUpdateBranchSkipsCurrentBranchWhenAlreadyAtStartPoint(t *testing.T) {
	repo := initTestRepo(t)
	sha := commitTestFile(t, repo, "one.txt", "one")
	withWorkingDir(t, repo)
	runGitForTest(t, repo, "switch", "-c", "stack/one")

	if err := ForceUpdateBranch("stack/one", sha); err != nil {
		t.Fatalf("ForceUpdateBranch returned error: %v", err)
	}
}

func TestForceUpdateBranchRejectsMovingCurrentBranch(t *testing.T) {
	repo := initTestRepo(t)
	oldSHA := commitTestFile(t, repo, "one.txt", "one")
	commitTestFile(t, repo, "two.txt", "two")
	withWorkingDir(t, repo)
	runGitForTest(t, repo, "switch", "-c", "stack/one")

	err := ForceUpdateBranch("stack/one", oldSHA)
	if err == nil {
		t.Fatalf("ForceUpdateBranch returned nil error")
	}
	if !strings.Contains(err.Error(), "cannot reset currently checked out branch") {
		t.Fatalf("error = %q, want actionable checked-out branch message", err)
	}
}

func TestForceUpdateBranchWrapsErrors(t *testing.T) {
	repo := initTestRepo(t)
	commitTestFile(t, repo, "one.txt", "one")
	withWorkingDir(t, repo)

	err := ForceUpdateBranch("bad branch", "HEAD")
	if err == nil {
		t.Fatalf("ForceUpdateBranch returned nil error")
	}
	var gitErr *Error
	if !errors.As(err, &gitErr) {
		t.Fatalf("error type = %T, want *git.Error", err)
	}
	if gitErr.Op != "force_update_branch" {
		t.Fatalf("git error op = %q, want force_update_branch", gitErr.Op)
	}
}

func initTestRepo(t *testing.T) string {
	t.Helper()
	repo := t.TempDir()
	runGitForTest(t, repo, "init", "-b", "main")
	runGitForTest(t, repo, "config", "user.name", "Test User")
	runGitForTest(t, repo, "config", "user.email", "test@example.com")
	return repo
}

func commitTestFile(t *testing.T, repo, name, contents string) string {
	t.Helper()
	if err := os.WriteFile(filepath.Join(repo, name), []byte(contents), 0o644); err != nil {
		t.Fatalf("write test file: %v", err)
	}
	runGitForTest(t, repo, "add", name)
	runGitForTest(t, repo, "commit", "-m", contents)
	out, err := shell.Output([]string{"git", "rev-parse", "HEAD"}, shell.RunOpts{Dir: repo})
	if err != nil {
		t.Fatalf("git rev-parse HEAD: %v", err)
	}
	return out
}

func runGitForTest(t *testing.T, repo string, args ...string) {
	t.Helper()
	cmd := append([]string{"git"}, args...)
	if _, err := shell.Output(cmd, shell.RunOpts{Dir: repo}); err != nil {
		t.Fatalf("%v: %v", cmd, err)
	}
}

func withWorkingDir(t *testing.T, dir string) {
	t.Helper()
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("get cwd: %v", err)
	}
	t.Cleanup(func() { _ = os.Chdir(cwd) })
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("chdir %s: %v", dir, err)
	}
}
