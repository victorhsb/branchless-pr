package cli

import (
	"strings"
	"testing"

	"github.com/victorhsb/branchless-pr/internal/config"
	"github.com/victorhsb/branchless-pr/internal/git"
	"github.com/victorhsb/branchless-pr/internal/invocation"
	"github.com/victorhsb/branchless-pr/internal/pr"
	"github.com/victorhsb/branchless-pr/internal/shell/shelltest"
	"github.com/victorhsb/branchless-pr/internal/stack"
)

func TestEffectiveLandStyle(t *testing.T) {
	cases := []struct {
		name      string
		cfgStyle  string
		flag      bool
		wantStyle string
	}{
		{"default is bottom-only", "", false, "bottom-only"},
		{"config bottom-only no flag", "bottom-only", false, "bottom-only"},
		{"config whole-stack no flag", "whole-stack", false, "whole-stack"},
		{"flag overrides bottom-only config", "bottom-only", true, "whole-stack"},
		{"flag overrides empty config", "", true, "whole-stack"},
		{"flag overrides whole-stack (still whole-stack)", "whole-stack", true, "whole-stack"},
		{"invalid style falls back to bottom-only", "rebase-merge", false, "bottom-only"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cfg := config.Defaults()
			cfg.Set("land", "style", tc.cfgStyle)
			app := &AppContext{Config: cfg}
			if got := effectiveLandStyle(app, tc.flag); got != tc.wantStyle {
				t.Fatalf("effectiveLandStyle(%q, %v) = %q, want %q", tc.cfgStyle, tc.flag, got, tc.wantStyle)
			}
		})
	}
}

// TestLandCmdRegistersWholeStackFlag checks the --whole-stack flag is wired.
func TestLandCmdRegistersWholeStackFlag(t *testing.T) {
	cmd := landCmd()
	f := cmd.Flags().Lookup("whole-stack")
	if f == nil {
		t.Fatalf("--whole-stack flag not registered on land command")
	}
	if f.Value.Type() != "bool" {
		t.Fatalf("--whole-stack type = %q, want bool", f.Value.Type())
	}
	if f.DefValue != "false" {
		t.Fatalf("--whole-stack default = %q, want false", f.DefValue)
	}
}

// installFakeShellForLand sets up process-free Git and GitHub runners.
// rebaseMergeAllowed controls the GraphQL response.
// mergeQueueEnabled controls the rules API response (true -> returns a
// merge_queue rule, false -> returns empty array).
func installFakeShellForLand(t *testing.T, rebaseMergeAllowed, mergeQueueEnabled bool) (*shelltest.Fake, *shelltest.Fake) {
	t.Helper()

	allowed := "false"
	if rebaseMergeAllowed {
		allowed = "true"
	}
	mqRules := "[]"
	if mergeQueueEnabled {
		mqRules = `[{"type":"merge_queue","parameters":{"merge_method":"rebase_or_merge"}}]`
	}

	ghResponses := []shelltest.Response{{
		Match:  shelltest.Prefix("gh", "api", "graphql"),
		Stdout: `{"data":{"repository":{"rebaseMergeAllowed":` + allowed + `}}}`,
	}}
	if rebaseMergeAllowed {
		ghResponses = append(ghResponses, shelltest.Response{
			Match:  shelltest.Exact("gh", "api", "repos/acme/widget/rules/branches/main"),
			Stdout: mqRules,
		})
	}
	if rebaseMergeAllowed && mergeQueueEnabled {
		ghResponses = append(ghResponses,
			shelltest.Response{Match: shelltest.Prefix("gh", "pr", "edit")},
			shelltest.Response{Match: shelltest.Prefix("gh", "pr", "merge")},
		)
	}
	ghRun := shelltest.New(t, ghResponses...)

	responses := []shelltest.Response{{
		Match:  shelltest.Exact("git", "remote", "get-url", "--", "origin"),
		Stdout: "https://github.com/acme/widget.git\n",
	}}
	if rebaseMergeAllowed && mergeQueueEnabled {
		responses = append(responses,
			shelltest.Response{Match: shelltest.Exact("git", "fetch", "--prune", "--", "origin")},
			shelltest.Response{Match: shelltest.Exact("git", "checkout", "feature")},
		)
	}
	gitRun := shelltest.New(t, responses...)
	return gitRun, ghRun
}

func entryForLandTest(head, prURL string) *stack.Entry {
	e := &stack.Entry{Commit: &stack.Header{Title: "land test"}}
	e.SetHead(head)
	e.SetPR(prURL)
	return e
}

func shellCallsLog(run *shelltest.Fake) string {
	var lines []string
	for _, call := range run.Calls() {
		args := call.Args
		if len(args) > 0 {
			args = args[1:]
		}
		lines = append(lines, strings.Join(args, " "))
	}
	return strings.Join(lines, "\n")
}

func TestLandWholeStackSingleEntry(t *testing.T) {
	gitRun, ghRun := installFakeShellForLand(t, true, true)

	app := &invocation.AppContext{
		Args:       invocation.CommonArgs{Remote: "origin", Target: "main"},
		Git:        git.New("", gitRun),
		PR:         pr.NewClient(ghRun),
		OrigBranch: "feature",
	}
	tip := entryForLandTest("alice/stack/1", "https://github.com/acme/widget/pull/1")
	st := stack.Stack{tip}

	out := captureStdout(t, func() {
		if err := landWholeStackImpl(app, st); err != nil {
			t.Fatalf("landWholeStackImpl returned error: %v", err)
		}
	})

	if !strings.Contains(out, "Whole-stack landing has been queued") {
		t.Fatalf("expected queued message in output, got:\n%s", out)
	}

	gh := shellCallsLog(ghRun)
	mustContain(t, gh, "api graphql")
	mustContain(t, gh, "api repos/acme/widget/rules/branches/main")
	mustContain(t, gh, "pr edit https://github.com/acme/widget/pull/1 -B main")
	mustContain(t, gh, "pr merge https://github.com/acme/widget/pull/1 --rebase --auto")

	gitLog := shellCallsLog(gitRun)
	mustContain(t, gitLog, "remote get-url -- origin")
	mustContain(t, gitLog, "fetch --prune -- origin")
	mustContain(t, gitLog, "checkout feature")
	// Queued whole-stack mode does NOT delete branches or rebase.
	if strings.Contains(gitLog, "branch -D") {
		t.Fatalf("did not expect branch deletion in queued mode, git log:\n%s", gitLog)
	}
	if strings.Contains(gitLog, "rebase") {
		t.Fatalf("did not expect rebase in queued mode, git log:\n%s", gitLog)
	}
}

func TestLandWholeStackMultiEntryRetargetsTip(t *testing.T) {
	gitRun, ghRun := installFakeShellForLand(t, true, true)

	app := &invocation.AppContext{
		Args:       invocation.CommonArgs{Remote: "origin", Target: "main"},
		Git:        git.New("", gitRun),
		PR:         pr.NewClient(ghRun),
		OrigBranch: "feature",
	}
	bottom := entryForLandTest("alice/stack/1", "https://github.com/acme/widget/pull/1")
	middle := entryForLandTest("alice/stack/2", "https://github.com/acme/widget/pull/2")
	tip := entryForLandTest("alice/stack/3", "https://github.com/acme/widget/pull/3")
	st := stack.Stack{bottom, middle, tip}

	captureStdout(t, func() {
		if err := landWholeStackImpl(app, st); err != nil {
			t.Fatalf("landWholeStackImpl returned error: %v", err)
		}
	})

	gh := shellCallsLog(ghRun)
	// Only the tip PR is edited and queued for merge.
	mustContain(t, gh, "pr edit https://github.com/acme/widget/pull/3 -B main")
	mustContain(t, gh, "pr merge https://github.com/acme/widget/pull/3 --rebase --auto")
	if strings.Contains(gh, "pr merge https://github.com/acme/widget/pull/1") ||
		strings.Contains(gh, "pr merge https://github.com/acme/widget/pull/2") {
		t.Fatalf("unexpected merge of non-tip PR in log:\n%s", gh)
	}
	if strings.Contains(gh, "--squash") {
		t.Fatalf("whole-stack should not invoke --squash:\n%s", gh)
	}

	gitLog := shellCallsLog(gitRun)
	// Queued mode does NOT delete local branches or rebase.
	if strings.Contains(gitLog, "branch -D") {
		t.Fatalf("did not expect branch deletion in queued mode, git log:\n%s", gitLog)
	}
	if strings.Contains(gitLog, "rebase") {
		t.Fatalf("did not expect rebase in queued mode, git log:\n%s", gitLog)
	}
	// No per-entry rebase/push for intermediate branches.
	if strings.Contains(gitLog, "push -f origin alice/stack/1:alice/stack/1") {
		t.Fatalf("did not expect intermediate force-push, log:\n%s", gitLog)
	}
}

func TestLandWholeStackRejectedWhenRebaseDisallowed(t *testing.T) {
	gitRun, ghRun := installFakeShellForLand(t, false, false)

	app := &invocation.AppContext{
		Args:       invocation.CommonArgs{Remote: "origin", Target: "main"},
		Git:        git.New("", gitRun),
		PR:         pr.NewClient(ghRun),
		OrigBranch: "feature",
	}
	tip := entryForLandTest("alice/stack/1", "https://github.com/acme/widget/pull/1")
	st := stack.Stack{tip}

	err := landWholeStackImpl(app, st)
	if err == nil {
		t.Fatalf("expected error when rebase merge is disallowed")
	}
	if !strings.Contains(err.Error(), "does not allow rebase merges") {
		t.Fatalf("error = %v, want guidance about rebase merges", err)
	}

	// No mutating gh/git calls should have happened.
	gh := shellCallsLog(ghRun)
	if strings.Contains(gh, "pr edit") || strings.Contains(gh, "pr merge") {
		t.Fatalf("expected no PR edits/merges when rebase disallowed, gh log:\n%s", gh)
	}
	gitLog := shellCallsLog(gitRun)
	if strings.Contains(gitLog, "fetch") || strings.Contains(gitLog, "checkout") {
		t.Fatalf("expected no fetch/checkout when rebase disallowed, git log:\n%s", gitLog)
	}
}

func TestLandWholeStackRejectedWhenMergeQueueDisabled(t *testing.T) {
	gitRun, ghRun := installFakeShellForLand(t, true, false)

	app := &invocation.AppContext{
		Args:       invocation.CommonArgs{Remote: "origin", Target: "main"},
		Git:        git.New("", gitRun),
		PR:         pr.NewClient(ghRun),
		OrigBranch: "feature",
	}
	tip := entryForLandTest("alice/stack/1", "https://github.com/acme/widget/pull/1")
	st := stack.Stack{tip}

	err := landWholeStackImpl(app, st)
	if err == nil {
		t.Fatalf("expected error when merge queue is disabled")
	}
	if !strings.Contains(err.Error(), "--whole-stack only works for repositories with merge queue enabled") {
		t.Fatalf("error = %v, want merge-queue error", err)
	}

	// No mutating gh/git calls should have happened after the rules check.
	gh := shellCallsLog(ghRun)
	if strings.Contains(gh, "pr edit") || strings.Contains(gh, "pr merge") {
		t.Fatalf("expected no PR edits/merges when merge queue disabled, gh log:\n%s", gh)
	}
	gitLog := shellCallsLog(gitRun)
	if strings.Contains(gitLog, "fetch") || strings.Contains(gitLog, "checkout") {
		t.Fatalf("expected no fetch/checkout when merge queue disabled, git log:\n%s", gitLog)
	}
}

func TestLandWholeStackUnknownMergeQueueProceedsAndNormalizes(t *testing.T) {
	gitRun := shelltest.New(t,
		shelltest.Response{
			Match:  shelltest.Exact("git", "remote", "get-url", "--", "origin"),
			Stdout: "https://github.com/acme/widget.git\n",
		},
		shelltest.Response{Match: shelltest.Exact("git", "fetch", "--prune", "--", "origin")},
	)
	ghRun := shelltest.New(t,
		shelltest.Response{
			Match:  shelltest.Prefix("gh", "api", "graphql"),
			Stdout: `{"data":{"repository":{"rebaseMergeAllowed":true}}}`,
		},
		shelltest.Response{
			Match:    shelltest.Exact("gh", "api", "repos/acme/widget/rules/branches/main"),
			Stdout:   `{"message":"Not Found"}`,
			ExitCode: 1,
		},
		shelltest.Response{Match: shelltest.Prefix("gh", "pr", "edit")},
		shelltest.Response{
			Match:    shelltest.Prefix("gh", "pr", "merge"),
			Stderr:   "merge queue is not enabled for this branch\n",
			ExitCode: 1,
		},
	)

	app := &invocation.AppContext{
		Args:       invocation.CommonArgs{Remote: "origin", Target: "main"},
		Git:        git.New("", gitRun),
		PR:         pr.NewClient(ghRun),
		OrigBranch: "feature",
	}
	tip := entryForLandTest("alice/stack/1", "https://github.com/acme/widget/pull/1")
	st := stack.Stack{tip}

	err := landWholeStackImpl(app, st)
	if err == nil {
		t.Fatalf("expected error when merge queue is disabled")
	}
	if !strings.Contains(err.Error(), "--whole-stack only works for repositories with merge queue enabled") {
		t.Fatalf("error = %v, want normalized merge-queue error", err)
	}
}
