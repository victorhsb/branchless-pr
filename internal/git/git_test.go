package git

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/victorhsb/branchless-pr/internal/shell"
	"github.com/victorhsb/branchless-pr/internal/shell/shelltest"
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

func TestResolveRemoteRefs(t *testing.T) {
	run := shelltest.New(t, shelltest.Response{
		Match:  shelltest.Exact("git", "ls-remote", "--heads", "--", "origin", "refs/heads/foo", "refs/heads/bar"),
		Stdout: "abc123\trefs/heads/foo\n",
	})

	got, err := New("", run).ResolveRemoteRefs("origin", "foo", "bar")
	if err != nil {
		t.Fatal(err)
	}
	if got["foo"] != "abc123" {
		t.Errorf("foo = %q, want abc123", got["foo"])
	}
	if _, ok := got["bar"]; ok {
		t.Errorf("bar should be absent")
	}
}

func TestRemoteBranchesUsesValidatedSharedParser(t *testing.T) {
	run := shelltest.New(t, shelltest.Response{
		Match: shelltest.Exact("git", "ls-remote", "--heads", "--", "origin"),
		Stdout: strings.Join([]string{
			"def456\trefs/heads/z-last",
			"malformed",
			"abc123\trefs/heads/a-first",
			"fff999\trefs/tags/not-a-branch",
		}, "\n"),
	})

	got, err := New("", run).RemoteBranches("origin")
	if err != nil {
		t.Fatal(err)
	}
	if want := []string{"a-first", "z-last"}; strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("RemoteBranches = %v, want %v", got, want)
	}
}

func TestForcePushWithLeaseUsesAtomicExplicitExpectations(t *testing.T) {
	run := shelltest.New(t, shelltest.Response{
		Match: shelltest.Exact(
			"git",
			"push",
			"--atomic",
			"--force-with-lease=refs/heads/foo:abc123",
			"--force-with-lease=refs/heads/bar:",
			"--",
			"origin",
			"foo:refs/heads/foo",
			"bar:refs/heads/bar",
		),
	})

	err := New("", run).ForcePushWithLease("origin", map[string]string{
		"foo": "abc123",
		"bar": "",
	}, "foo", "bar")
	if err != nil {
		t.Fatal(err)
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

func TestGitOperationDetectionRecognizesMarkers(t *testing.T) {
	tests := []struct {
		name      string
		marker    string
		directory bool
		detect    func(*Repo) bool
	}{
		{name: "rebase merge", marker: "rebase-merge", directory: true, detect: func(r *Repo) bool { return r.IsRebaseInProgress() }},
		{name: "rebase apply", marker: "rebase-apply", directory: true, detect: func(r *Repo) bool { return r.IsRebaseInProgress() }},
		{name: "merge", marker: "MERGE_HEAD", detect: func(r *Repo) bool { return r.IsMergeInProgress() }},
		{name: "cherry-pick", marker: "CHERRY_PICK_HEAD", detect: func(r *Repo) bool { return r.IsCherryPickInProgress() }},
		{name: "sequencer", marker: "sequencer/todo", detect: func(r *Repo) bool { return r.IsCherryPickInProgress() }},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := initTestRepo(t)
			writeOperationMarker(t, repo, tt.marker, tt.directory)
			gitRepo := New(repo, nil)

			if !tt.detect(gitRepo) {
				t.Fatalf("%s was not detected in %s", tt.marker, repo)
			}
			if !gitRepo.AnySequencerInProgress() {
				t.Fatalf("aggregate operation detection missed %s", tt.marker)
			}
		})
	}
}

func TestAnySequencerInProgressReturnsFalseWithoutMarkers(t *testing.T) {
	repo := initTestRepo(t)
	if New(repo, nil).AnySequencerInProgress() {
		t.Fatal("operation reported active in repository without operation markers")
	}
}

func TestGitOperationDetectionSupportsRepositoryLayouts(t *testing.T) {
	t.Run("repository subdirectory", func(t *testing.T) {
		repo := initTestRepo(t)
		subdir := filepath.Join(repo, "nested", "directory")
		if err := os.MkdirAll(subdir, 0o755); err != nil {
			t.Fatal(err)
		}
		writeOperationMarker(t, repo, "rebase-merge", true)
		if !New(subdir, nil).IsRebaseInProgress() {
			t.Fatal("rebase was not detected from a repository subdirectory")
		}
	})

	t.Run("linked worktree", func(t *testing.T) {
		repo := initTestRepo(t)
		commitTestFile(t, repo, "initial.txt", "initial")
		linked := filepath.Join(t.TempDir(), "linked")
		runGitForTest(t, repo, "worktree", "add", "-b", "linked", linked)
		writeOperationMarker(t, linked, "MERGE_HEAD", false)

		if !New(linked, nil).IsMergeInProgress() {
			t.Fatal("merge was not detected in a linked worktree")
		}
	})

	t.Run("submodule", func(t *testing.T) {
		source := initTestRepo(t)
		commitTestFile(t, source, "initial.txt", "initial")
		super := initTestRepo(t)
		child := filepath.Join(super, "modules", "child")
		runGitForTest(t, super, "-c", "protocol.file.allow=always", "submodule", "add", source, "modules/child")
		writeOperationMarker(t, child, "CHERRY_PICK_HEAD", false)

		if !New(child, nil).IsCherryPickInProgress() {
			t.Fatal("cherry-pick was not detected in a submodule")
		}
	})

	t.Run("separate git directory", func(t *testing.T) {
		parent := t.TempDir()
		worktree := filepath.Join(parent, "worktree")
		gitDir := filepath.Join(parent, "metadata")
		runGitForTest(t, parent, "init", "-b", "main", "--separate-git-dir", gitDir, worktree)
		writeOperationMarker(t, worktree, "sequencer/todo", false)

		if !New(worktree, nil).IsCherryPickInProgress() {
			t.Fatal("sequencer was not detected with a separate Git directory")
		}
	})
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
	out, err := (shell.Default{}).Output([]string{"git", "rev-parse", "HEAD"}, shell.RunOpts{Dir: repo})
	if err != nil {
		t.Fatalf("git rev-parse HEAD: %v", err)
	}
	return out
}

func runGitForTest(t *testing.T, repo string, args ...string) {
	t.Helper()
	cmd := append([]string{"git"}, args...)
	if _, err := (shell.Default{}).Output(cmd, shell.RunOpts{Dir: repo}); err != nil {
		t.Fatalf("%v: %v", cmd, err)
	}
}

func writeOperationMarker(t *testing.T, repo, marker string, directory bool) {
	t.Helper()
	path, err := (shell.Default{}).Output(
		[]string{"git", "rev-parse", "--git-path", marker},
		shell.RunOpts{Dir: repo},
	)
	if err != nil {
		t.Fatalf("resolve Git path %s: %v", marker, err)
	}
	if !filepath.IsAbs(path) {
		path = filepath.Join(repo, path)
	}
	if directory {
		if err := os.MkdirAll(path, 0o755); err != nil {
			t.Fatalf("create operation directory %s: %v", path, err)
		}
		return
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("create operation marker parent: %v", err)
	}
	if err := os.WriteFile(path, []byte("test operation marker\n"), 0o644); err != nil {
		t.Fatalf("write operation marker %s: %v", path, err)
	}
}
