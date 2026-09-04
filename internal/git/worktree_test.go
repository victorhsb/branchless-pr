package git

import (
	"errors"
	"strings"
	"testing"

	"github.com/victorhsb/branchless-pr/internal/shell"
	"github.com/victorhsb/branchless-pr/internal/shell/shelltest"
)

func TestReposKeepRunnerAndDirectoryStateIsolated(t *testing.T) {
	for _, tc := range []struct {
		name string
		dir  string
		root string
	}{
		{name: "first", dir: "/work/one", root: "/repo/one"},
		{name: "second", dir: "/work/two", root: "/repo/two"},
	} {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			run := shelltest.New(t, shelltest.Response{
				Match: func(args []string, opts shell.RunOpts) bool {
					return shelltest.Exact("git", "rev-parse", "--show-toplevel")(args, opts) && opts.Dir == tc.dir
				},
				Stdout: tc.root,
			})

			got, err := New(tc.dir, run).RepoRoot()
			if err != nil {
				t.Fatal(err)
			}
			if got != tc.root {
				t.Fatalf("RepoRoot = %q, want %q", got, tc.root)
			}
		})
	}
}

func TestHasTrackedChanges(t *testing.T) {
	for _, tc := range []struct {
		name   string
		status string
		want   bool
	}{
		{name: "clean"},
		{name: "untracked only", status: "?? one.txt\n?? two.txt"},
		{name: "tracked", status: " M tracked.txt", want: true},
		{name: "mixed", status: "?? new.txt\nM  tracked.txt", want: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			run := shelltest.New(t, shelltest.Response{
				Match:  shelltest.Exact("git", "status", "--porcelain"),
				Stdout: tc.status,
			})
			got, err := New("", run).HasTrackedChanges()
			if err != nil {
				t.Fatal(err)
			}
			if got != tc.want {
				t.Fatalf("HasTrackedChanges = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestCheckoutUsesBranchFirstSignature(t *testing.T) {
	run := shelltest.New(t, shelltest.Response{
		Match: shelltest.Exact("git", "checkout", "HEAD", "-B", "feature"),
	})
	if err := New("", run).Checkout("feature", "HEAD"); err != nil {
		t.Fatal(err)
	}
}

func TestForceUpdateBranchCreatesMissingBranch(t *testing.T) {
	repo := initTestRepo(t)
	sha := commitTestFile(t, repo, "one.txt", "one")
	gitRepo := New(repo, nil)

	if err := gitRepo.ForceUpdateBranch("stack/one", sha); err != nil {
		t.Fatalf("ForceUpdateBranch returned error: %v", err)
	}
	got, err := gitRepo.RevParse("stack/one")
	if err != nil {
		t.Fatalf("RevParse returned error: %v", err)
	}
	if got != sha {
		t.Fatalf("stack/one = %s, want %s", got, sha)
	}
	if branch, err := gitRepo.CurrentBranchName(); err != nil || branch != "main" {
		t.Fatalf("current branch = %q, %v; want main", branch, err)
	}
}

func TestForceUpdateBranchResetsExistingBranch(t *testing.T) {
	repo := initTestRepo(t)
	oldSHA := commitTestFile(t, repo, "one.txt", "one")
	newSHA := commitTestFile(t, repo, "two.txt", "two")
	gitRepo := New(repo, nil)

	if err := gitRepo.ForceUpdateBranch("stack/one", newSHA); err != nil {
		t.Fatalf("ForceUpdateBranch create returned error: %v", err)
	}
	if err := gitRepo.ForceUpdateBranch("stack/one", oldSHA); err != nil {
		t.Fatalf("ForceUpdateBranch reset returned error: %v", err)
	}
	got, err := gitRepo.RevParse("stack/one")
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
	runGitForTest(t, repo, "switch", "-c", "stack/one")

	if err := New(repo, nil).ForceUpdateBranch("stack/one", sha); err != nil {
		t.Fatalf("ForceUpdateBranch returned error: %v", err)
	}
}

func TestForceUpdateBranchRejectsMovingCurrentBranch(t *testing.T) {
	repo := initTestRepo(t)
	oldSHA := commitTestFile(t, repo, "one.txt", "one")
	commitTestFile(t, repo, "two.txt", "two")
	runGitForTest(t, repo, "switch", "-c", "stack/one")

	err := New(repo, nil).ForceUpdateBranch("stack/one", oldSHA)
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

	err := New(repo, nil).ForceUpdateBranch("bad branch", "HEAD")
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
