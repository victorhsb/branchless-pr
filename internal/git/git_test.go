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

func TestGitOperationDetectionRecognizesMarkers(t *testing.T) {
	tests := []struct {
		name      string
		marker    string
		directory bool
		detect    func(...string) bool
	}{
		{name: "rebase merge", marker: "rebase-merge", directory: true, detect: IsRebaseInProgress},
		{name: "rebase apply", marker: "rebase-apply", directory: true, detect: IsRebaseInProgress},
		{name: "merge", marker: "MERGE_HEAD", detect: IsMergeInProgress},
		{name: "cherry-pick", marker: "CHERRY_PICK_HEAD", detect: IsCherryPickInProgress},
		{name: "sequencer", marker: "sequencer/todo", detect: IsCherryPickInProgress},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := initTestRepo(t)
			writeOperationMarker(t, repo, tt.marker, tt.directory)

			if !tt.detect(repo) {
				t.Fatalf("%s was not detected in %s", tt.marker, repo)
			}
			if !AnySequencerInProgress(repo) {
				t.Fatalf("aggregate operation detection missed %s", tt.marker)
			}
		})
	}
}

func TestAnySequencerInProgressReturnsFalseWithoutMarkers(t *testing.T) {
	repo := initTestRepo(t)
	if AnySequencerInProgress(repo) {
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
		withWorkingDir(t, subdir)

		if !IsRebaseInProgress() {
			t.Fatal("rebase was not detected from a repository subdirectory")
		}
	})

	t.Run("linked worktree", func(t *testing.T) {
		repo := initTestRepo(t)
		commitTestFile(t, repo, "initial.txt", "initial")
		linked := filepath.Join(t.TempDir(), "linked")
		runGitForTest(t, repo, "worktree", "add", "-b", "linked", linked)
		writeOperationMarker(t, linked, "MERGE_HEAD", false)

		if !IsMergeInProgress(linked) {
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

		if !IsCherryPickInProgress(child) {
			t.Fatal("cherry-pick was not detected in a submodule")
		}
	})

	t.Run("separate git directory", func(t *testing.T) {
		parent := t.TempDir()
		worktree := filepath.Join(parent, "worktree")
		gitDir := filepath.Join(parent, "metadata")
		runGitForTest(t, parent, "init", "-b", "main", "--separate-git-dir", gitDir, worktree)
		writeOperationMarker(t, worktree, "sequencer/todo", false)

		if !IsCherryPickInProgress(worktree) {
			t.Fatal("sequencer was not detected with a separate Git directory")
		}
	})
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

func writeOperationMarker(t *testing.T, repo, marker string, directory bool) {
	t.Helper()
	path, err := shell.Output(
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
