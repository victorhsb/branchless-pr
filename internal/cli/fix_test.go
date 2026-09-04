package cli

import (
	"strings"
	"testing"

	"github.com/victorhsb/branchless-pr/internal/config"
	"github.com/victorhsb/branchless-pr/internal/git"
	"github.com/victorhsb/branchless-pr/internal/pr"
	"github.com/victorhsb/branchless-pr/internal/shell/shelltest"
)

func TestFixCmdExposesFlags(t *testing.T) {
	cmd := fixCmd()

	if got := cmd.Use; got != "fix" {
		t.Fatalf("fix Use = %q, want fix", got)
	}

	for _, name := range []string{"pr", "replace", "dry-run"} {
		f := cmd.Flags().Lookup(name)
		if f == nil {
			t.Fatalf("--%s flag not registered on fix command", name)
		}
		if f.Value.Type() != "bool" && f.Value.Type() != "int" {
			t.Fatalf("--%s flag type = %q, want bool or int", name, f.Value.Type())
		}
	}
}

func TestFixDryRunReportsNoAmend(t *testing.T) {
	repoDir := t.TempDir()
	headSHA := "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	fixRun := newFixGitRunner(t, headSHA, "Hello world", false, true)
	gitRepo := git.New(repoDir, fixRun)

	out := captureStdout(t, func() {
		app := &AppContext{
			Config: config.Defaults(),
			Git:    gitRepo,
			PR:     newFixPRClient(t),
			Args: CommonArgs{
				Base:   headSHA,
				Head:   "HEAD",
				Remote: "origin",
				Target: "main",
			},
			RepoRoot:   repoDir,
			Username:   "testuser",
			OrigBranch: "main",
		}
		err := fixImpl(app, fixOptions{PRNumber: 42, DryRun: true})
		if err != nil {
			t.Fatalf("fixImpl returned error: %v", err)
		}
	})

	if !strings.Contains(out, "PR URL:") {
		t.Fatalf("dry-run output missing PR URL, got:\n%s", out)
	}
	if !strings.Contains(out, "No commit was changed") {
		t.Fatalf("dry-run output missing 'No commit was changed', got:\n%s", out)
	}

	// Verify no amend was attempted through the injected Git boundary.
	log := shellCallsLog(fixRun)
	if strings.Contains(log, "commit --amend") {
		t.Fatalf("dry-run should not amend; git log:\n%s", log)
	}
}

func TestFixAlreadyFixedReportsNoop(t *testing.T) {
	repoDir := t.TempDir()
	headSHA := "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	gitRepo := git.New(repoDir, newFixGitRunner(
		t,
		headSHA,
		"initial\n\nstack-info: PR: https://github.com/test/repo/pull/42, branch: feature",
		false,
		true,
	))

	out := captureStdout(t, func() {
		app := &AppContext{
			Config: config.Defaults(),
			Git:    gitRepo,
			PR:     newFixPRClient(t),
			Args: CommonArgs{
				Base:   headSHA,
				Head:   "HEAD",
				Remote: "origin",
				Target: "main",
			},
			RepoRoot:   repoDir,
			Username:   "testuser",
			OrigBranch: "main",
		}
		err := fixImpl(app, fixOptions{PRNumber: 42})
		if err != nil {
			t.Fatalf("fixImpl returned error: %v", err)
		}
	})

	if !strings.Contains(out, "already fixed") {
		t.Fatalf("expected 'already fixed' in output, got:\n%s", out)
	}
}

func TestFixRefusesDifferentMetadataWithoutReplace(t *testing.T) {
	repoDir := t.TempDir()
	headSHA := "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	gitRepo := git.New(repoDir, newFixGitRunner(
		t,
		headSHA,
		"initial\n\nstack-info: PR: https://github.com/test/repo/pull/1, branch: old-branch",
		false,
		false,
	))

	err := fixImpl(&AppContext{
		Config: config.Defaults(),
		Git:    gitRepo,
		PR:     newFixPRClient(t),
		Args: CommonArgs{
			Base:   headSHA,
			Head:   "HEAD",
			Remote: "origin",
			Target: "main",
		},
		RepoRoot:   repoDir,
		Username:   "testuser",
		OrigBranch: "main",
	}, fixOptions{PRNumber: 42})
	if err == nil {
		t.Fatal("expected error for different metadata without --replace")
	}
	if !strings.Contains(err.Error(), "already has different stack metadata") {
		t.Fatalf("expected different metadata error, got: %v", err)
	}
}

func TestFixReplaceOverwritesDifferentMetadata(t *testing.T) {
	repoDir := t.TempDir()
	headSHA := "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	fixRun := newFixGitRunner(
		t,
		headSHA,
		"initial\n\nstack-info: PR: https://github.com/test/repo/pull/1, branch: old-branch",
		true,
		true,
	)
	gitRepo := git.New(repoDir, fixRun)

	out := captureStdout(t, func() {
		app := &AppContext{
			Config: config.Defaults(),
			Git:    gitRepo,
			PR:     newFixPRClient(t),
			Args: CommonArgs{
				Base:   headSHA,
				Head:   "HEAD",
				Remote: "origin",
				Target: "main",
			},
			RepoRoot:   repoDir,
			Username:   "testuser",
			OrigBranch: "main",
		}
		err := fixImpl(app, fixOptions{PRNumber: 42, Replace: true})
		if err != nil {
			t.Fatalf("fixImpl returned error: %v", err)
		}
	})

	if !strings.Contains(out, "Fixed stack metadata") {
		t.Fatalf("expected success output, got:\n%s", out)
	}

	log := shellCallsLog(fixRun)
	if !strings.Contains(log, "commit --amend") {
		t.Fatalf("expected amend in git log:\n%s", log)
	}
}

func newFixGitRunner(t *testing.T, headSHA, commitMsg string, amend, discover bool) *shelltest.Fake {
	t.Helper()
	responses := []shelltest.Response{
		{Match: shelltest.Exact("git", "rev-parse", "--git-path", "rebase-merge")},
		{Match: shelltest.Exact("git", "rev-parse", "--git-path", "rebase-apply")},
		{Match: shelltest.Exact("git", "rev-parse", "--git-path", "MERGE_HEAD")},
		{Match: shelltest.Exact("git", "rev-parse", "--git-path", "sequencer/todo")},
		{Match: shelltest.Exact("git", "rev-parse", "--git-path", "CHERRY_PICK_HEAD")},
		{
			Match:  shelltest.Exact("git", "rev-parse", "--verify", "HEAD"),
			Stdout: headSHA + "\n",
		},
		{
			Match:  shelltest.Exact("git", "log", "-1", "--pretty=%B"),
			Stdout: commitMsg + "\n",
		},
	}
	if amend {
		responses = append(responses, shelltest.Response{
			Match: shelltest.Exact("git", "commit", "--amend", "-F", "-"),
		})
	}
	if discover {
		responses = append(responses, shelltest.Response{
			Match: shelltest.Prefix("git", "rev-list", "--header"),
		})
	}
	return shelltest.New(t, responses...)
}

func newFixPRClient(t *testing.T) *pr.Client {
	t.Helper()
	run := shelltest.New(t, shelltest.Response{
		Match:  shelltest.Prefix("gh", "pr", "view", "42"),
		Stdout: `{"url":"https://github.com/test/repo/pull/42","headRefName":"feature","baseRefName":"main","headRefOid":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","number":42,"state":"OPEN","body":"","title":"Test PR","mergeStateStatus":"CLEAN","isDraft":false}`,
	})
	return pr.NewClient(run)
}

func TestBuildFixedMessageAppendsMetadata(t *testing.T) {
	msg := buildFixedMessage("Hello world\n", "https://github.com/test/repo/pull/42", "feature")
	want := "Hello world\n\nstack-info: PR: https://github.com/test/repo/pull/42, branch: feature\n"
	if msg != want {
		t.Fatalf("buildFixedMessage = %q, want %q", msg, want)
	}
}

func TestBuildFixedMessageReplacesExisting(t *testing.T) {
	msg := buildFixedMessage("Hello world\n\nstack-info: PR: https://old, branch: old\n", "https://github.com/test/repo/pull/42", "feature")
	want := "Hello world\n\nstack-info: PR: https://github.com/test/repo/pull/42, branch: feature\n"
	if msg != want {
		t.Fatalf("buildFixedMessage = %q, want %q", msg, want)
	}
}

func TestPluralize(t *testing.T) {
	if pluralize(1, "y is", "ies are") != "y is" {
		t.Fatal("pluralize(1) != y is")
	}
	if pluralize(2, "y is", "ies are") != "ies are" {
		t.Fatal("pluralize(2) != entries are")
	}
}
