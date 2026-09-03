package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	gitpkg "github.com/victorhsb/branchless-pr/internal/git"
	"github.com/victorhsb/branchless-pr/internal/shell"
	"github.com/victorhsb/branchless-pr/internal/shell/shelltest"
)

func TestAutomaticStashRestoredAcrossInvocationLifecycle(t *testing.T) {
	username := "stash-test-user"
	gitpkg.DefaultConfig().SetUsernameOverride(&username)
	t.Cleanup(func() { gitpkg.DefaultConfig().SetUsernameOverride(nil) })

	tests := []struct {
		name             string
		args             []string
		operationMarker  string
		forceDirtyStatus bool
		wantError        string
	}{
		{
			name:      "successful command",
			args:      []string{"submit", "--stash", "--base", "HEAD"},
			wantError: "",
		},
		{
			name:             "clean check failure",
			args:             []string{"submit", "--stash", "--base", "HEAD"},
			forceDirtyStatus: true,
			wantError:        "working tree is not clean",
		},
		{
			name:      "target validation failure",
			args:      []string{"submit", "--stash", "--base", "HEAD", "--target", "missing"},
			wantError: "target branch missing",
		},
		{
			name:      "merge-base failure",
			args:      []string{"submit", "--stash", "--head", "missing-head"},
			wantError: "unable to deduce base merge-base",
		},
		{
			name:            "command failure",
			args:            []string{"submit", "--stash", "--base", "HEAD"},
			operationMarker: "rebase-merge",
			wantError:       "Rebase in progress",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo, realGit := setupStashLifecycleRepo(t)
			if tt.operationMarker != "" {
				createGitOperationMarker(t, realGit, repo, tt.operationMarker)
			}
			runner := installStashLifecycleCommands(t, tt.forceDirtyStatus, false)
			chdirForTest(t, repo)

			_, err := executeRootForTest(tt.args, runner)
			if tt.wantError == "" {
				if err != nil {
					t.Fatalf("execute submit: %v", err)
				}
			} else if err == nil || !strings.Contains(err.Error(), tt.wantError) {
				t.Fatalf("error = %v, want substring %q", err, tt.wantError)
			}

			assertOriginalWorkingTreeRestored(t, realGit, repo)
		})
	}
}

func TestPreRunPreservesInitializationAndStashRestoreFailures(t *testing.T) {
	username := "stash-test-user"
	gitpkg.DefaultConfig().SetUsernameOverride(&username)
	t.Cleanup(func() { gitpkg.DefaultConfig().SetUsernameOverride(nil) })

	repo, realGit := setupStashLifecycleRepo(t)
	runner := installStashLifecycleCommands(t, false, true)
	chdirForTest(t, repo)

	_, err := executeRootForTest([]string{"submit", "--stash", "--base", "HEAD", "--target", "missing"}, runner)
	if err == nil {
		t.Fatal("expected target validation and stash restoration error")
	}
	for _, want := range []string{"target branch missing", "failed to restore automatic stash", "stash_apply"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("error = %v, want substring %q", err, want)
		}
	}
	if got := gitOutputForStashTest(t, realGit, repo, "stash", "list"); got == "" {
		t.Fatal("failed stash apply should leave the automatic stash available for recovery")
	}
}

func TestRestoreStashDoesNothingWhenInvocationCreatedNoStash(t *testing.T) {
	app := &AppContext{}
	t.Setenv("PATH", t.TempDir())
	if err := app.RestoreStash(); err != nil {
		t.Fatalf("RestoreStash without recorded stash: %v", err)
	}
}

func TestAutomaticStashRestorationPreservesPreExistingUserStash(t *testing.T) {
	username := "stash-test-user"
	gitpkg.DefaultConfig().SetUsernameOverride(&username)
	t.Cleanup(func() { gitpkg.DefaultConfig().SetUsernameOverride(nil) })

	repo, realGit := setupStashLifecycleRepo(t)
	runGitForStashTest(t, realGit, repo, "stash", "push", "-m", "pre-existing user stash")
	userOID := gitOutputForStashTest(t, realGit, repo, "rev-parse", "refs/stash^{commit}")
	if err := os.WriteFile(filepath.Join(repo, "tracked.txt"), []byte("automatic changes\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	runner := installStashLifecycleCommands(t, false, false)
	chdirForTest(t, repo)

	_, err := executeRootForTest([]string{"submit", "--stash", "--base", "HEAD", "--target", "missing"}, runner)
	if err == nil || !strings.Contains(err.Error(), "target branch missing") {
		t.Fatalf("error = %v, want target validation failure", err)
	}
	contents, err := os.ReadFile(filepath.Join(repo, "tracked.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if got, want := string(contents), "automatic changes\n"; got != want {
		t.Fatalf("tracked file = %q, want %q", got, want)
	}
	if got := gitOutputForStashTest(t, realGit, repo, "stash", "list", "--format=%H"); got != userOID {
		t.Fatalf("remaining stash = %q, want pre-existing user stash %q", got, userOID)
	}
}

func TestRestoreStashConsumesIdentityOnlyAfterSuccess(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		repo, _ := setupStashLifecycleRepo(t)
		gitRepo := gitpkg.New(repo, shell.Default{})
		ref, err := gitRepo.StashSave("automatic")
		if err != nil {
			t.Fatal(err)
		}
		app := &AppContext{Git: gitRepo, AutomaticStash: ref}

		if err := app.RestoreStash(); err != nil {
			t.Fatalf("RestoreStash: %v", err)
		}
		if !app.AutomaticStash.IsZero() {
			t.Fatalf("successful restoration retained identity %q", app.AutomaticStash.OID)
		}
	})

	t.Run("conflict", func(t *testing.T) {
		repo, realGit := setupStashLifecycleRepo(t)
		gitRepo := gitpkg.New(repo, shell.Default{})
		ref, err := gitRepo.StashSave("automatic")
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(repo, "tracked.txt"), []byte("conflicting committed changes\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		runGitForStashTest(t, realGit, repo, "add", "tracked.txt")
		runGitForStashTest(t, realGit, repo, "commit", "-m", "conflict")
		app := &AppContext{Git: gitRepo, AutomaticStash: ref}

		if err := app.RestoreStash(); err == nil {
			t.Fatal("expected restoration conflict")
		}
		if app.AutomaticStash != ref {
			t.Fatalf("conflicting restoration identity = %+v, want retained %+v", app.AutomaticStash, ref)
		}
	})
}

func setupStashLifecycleRepo(t *testing.T) (repo, realGit string) {
	t.Helper()
	realGit = findExecutableForTest(t, "git")
	parent := t.TempDir()
	repo = filepath.Join(parent, "repo")
	remote := filepath.Join(parent, "remote.git")
	if err := os.MkdirAll(repo, 0o755); err != nil {
		t.Fatal(err)
	}
	runGitForStashTest(t, realGit, repo, "init", "-b", "main")
	runGitForStashTest(t, realGit, repo, "config", "user.name", "Test User")
	runGitForStashTest(t, realGit, repo, "config", "user.email", "test@example.com")
	tracked := filepath.Join(repo, "tracked.txt")
	if err := os.WriteFile(tracked, []byte("original\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	runGitForStashTest(t, realGit, repo, "add", "tracked.txt")
	runGitForStashTest(t, realGit, repo, "commit", "-m", "initial")
	if err := os.MkdirAll(remote, 0o755); err != nil {
		t.Fatal(err)
	}
	runGitForStashTest(t, realGit, remote, "init", "--bare")
	runGitForStashTest(t, realGit, repo, "remote", "add", "origin", remote)
	runGitForStashTest(t, realGit, repo, "push", "-u", "origin", "main")
	if err := os.WriteFile(tracked, []byte("user changes\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	return repo, realGit
}

func installStashLifecycleCommands(t *testing.T, forceDirtyStatus, failStashRestore bool) shell.Runner {
	t.Helper()
	bin := t.TempDir()
	ghScript := "#!/bin/sh\nexit 0\n"
	if err := os.WriteFile(filepath.Join(bin, "gh"), []byte(ghScript), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
	return stashLifecycleRunner{
		forceDirtyStatus: forceDirtyStatus,
		failStashRestore: failStashRestore,
	}
}

type stashLifecycleRunner struct {
	forceDirtyStatus bool
	failStashRestore bool
}

func (r stashLifecycleRunner) Output(args []string, opts shell.RunOpts) (string, error) {
	if r.forceDirtyStatus && equalArgs(args, "git", "status", "--porcelain") {
		return " M tracked.txt", nil
	}
	return (shell.Default{}).Output(args, opts)
}

func (r stashLifecycleRunner) Run(args []string, opts shell.RunOpts) ([]byte, []byte, error) {
	if r.failStashRestore && len(args) >= 3 && args[0] == "git" && args[1] == "stash" && args[2] == "apply" {
		return nil, []byte("forced stash apply failure\n"), &shelltest.ExitError{Code: 42}
	}
	return (shell.Default{}).Run(args, opts)
}

func equalArgs(got []string, want ...string) bool {
	if len(got) != len(want) {
		return false
	}
	for i := range want {
		if got[i] != want[i] {
			return false
		}
	}
	return true
}

func createGitOperationMarker(t *testing.T, realGit, repo, marker string) {
	t.Helper()
	path := gitOutputForStashTest(t, realGit, repo, "rev-parse", "--git-path", marker)
	if !filepath.IsAbs(path) {
		path = filepath.Join(repo, path)
	}
	if err := os.MkdirAll(path, 0o755); err != nil {
		t.Fatalf("create Git operation marker: %v", err)
	}
}

func assertOriginalWorkingTreeRestored(t *testing.T, realGit, repo string) {
	t.Helper()
	contents, err := os.ReadFile(filepath.Join(repo, "tracked.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if got, want := string(contents), "user changes\n"; got != want {
		t.Fatalf("tracked file = %q, want restored contents %q", got, want)
	}
	if got := gitOutputForStashTest(t, realGit, repo, "stash", "list"); got != "" {
		t.Fatalf("automatic stash was not consumed: %s", got)
	}
}

func runGitForStashTest(t *testing.T, gitPath, repo string, args ...string) {
	t.Helper()
	if _, err := shell.Output(append([]string{gitPath}, args...), shell.RunOpts{Dir: repo}); err != nil {
		t.Fatalf("git %v: %v", args, err)
	}
}

func gitOutputForStashTest(t *testing.T, gitPath, repo string, args ...string) string {
	t.Helper()
	out, err := shell.Output(append([]string{gitPath}, args...), shell.RunOpts{Dir: repo})
	if err != nil {
		t.Fatalf("git %v: %v", args, err)
	}
	return out
}

func findExecutableForTest(t *testing.T, name string) string {
	t.Helper()
	for _, dir := range filepath.SplitList(os.Getenv("PATH")) {
		candidate := filepath.Join(dir, name)
		info, err := os.Stat(candidate)
		if err == nil && !info.IsDir() && info.Mode()&0o111 != 0 {
			return candidate
		}
	}
	t.Fatalf("%s not found on PATH", name)
	return ""
}
