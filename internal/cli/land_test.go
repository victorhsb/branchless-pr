package cli

import (
	"errors"
	"fmt"
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

func TestLandBottomOnlyChecksOutBottomFromRemoteBranch(t *testing.T) {
	gitRun := shelltest.New(t,
		shelltest.Response{Match: shelltest.Exact("git", "fetch", "--prune", "--", "origin")},
		shelltest.Response{
			Match: shelltest.Exact("git", "checkout", "origin/alice/stack/1", "-B", "alice/stack/1"),
			Err:   errors.New("stop after checkout"),
		},
	)
	app := &AppContext{
		Args: CommonArgs{Remote: "origin", Target: "main"},
		Git:  git.New("", gitRun),
		PR:   pr.NewClient(shelltest.New(t)),
	}
	err := landBottomOnly(app, stack.Stack{
		entryForLandTest("alice/stack/1", "https://github.com/acme/widget/pull/1"),
	})
	if err == nil || !strings.Contains(err.Error(), "stop after checkout") {
		t.Fatalf("error = %v, want checkout sentinel", err)
	}
}

func TestLandBottomOnlyChecksOutRemainingEntryFromRemoteBranch(t *testing.T) {
	gitRun := shelltest.New(t,
		shelltest.Response{Match: shelltest.Exact("git", "fetch", "--prune", "--", "origin")},
		shelltest.Response{Match: shelltest.Exact("git", "checkout", "origin/alice/stack/1", "-B", "alice/stack/1")},
		shelltest.Response{Match: shelltest.Exact("git", "fetch", "--prune", "--", "origin")},
		shelltest.Response{
			Match: shelltest.Exact("git", "checkout", "origin/alice/stack/2", "-B", "alice/stack/2"),
			Err:   errors.New("stop after remaining checkout"),
		},
	)
	ghRun := shelltest.New(t,
		shelltest.Response{Match: shelltest.Exact("gh", "pr", "edit", "https://github.com/acme/widget/pull/1", "-B", "main")},
		shelltest.Response{Match: shelltest.Exact("gh", "pr", "merge", "https://github.com/acme/widget/pull/1", "--squash", "-t", "land test (#1)", "-F", "-")},
	)
	app := &AppContext{
		Args: CommonArgs{Remote: "origin", Target: "main"},
		Git:  git.New("", gitRun),
		PR:   pr.NewClient(ghRun),
	}
	err := landBottomOnly(app, stack.Stack{
		entryForLandTest("alice/stack/1", "https://github.com/acme/widget/pull/1"),
		entryForLandTest("alice/stack/2", "https://github.com/acme/widget/pull/2"),
	})
	if err == nil || !strings.Contains(err.Error(), "stop after remaining checkout") {
		t.Fatalf("error = %v, want remaining checkout sentinel", err)
	}
}

func TestLandBottomStopsBeforeCleanupOnFailure(t *testing.T) {
	const head = "alice/stack/1"
	const url = "https://github.com/acme/widget/pull/1"
	steps := []shelltest.Response{
		{Name: "fetch", Match: shelltest.Exact("git", "fetch", "--prune", "--", "origin")},
		{Name: "checkout bottom", Match: shelltest.Exact("git", "checkout", "origin/"+head, "-B", head)},
		{Name: "retarget bottom", Match: shelltest.Exact("gh", "pr", "edit", url, "-B", "main")},
		{Name: "squash merge", Match: shelltest.Exact("gh", "pr", "merge", url, "--squash", "-t", "land test (#1)", "-F", "-")},
		{Name: "fetch remaining", Match: shelltest.Exact("git", "fetch", "--prune", "--", "origin")},
		{Name: "checkout remaining", Match: shelltest.Exact("git", "checkout", "origin/alice/stack/2", "-B", "alice/stack/2")},
		{Name: "rebase remaining", Match: shelltest.Exact("git", "rebase", "--committer-date-is-author-date", "origin/main", "alice/stack/2")},
		{Name: "push remaining", Match: shelltest.Exact("git", "push", "-f", "--", "origin", "alice/stack/2:alice/stack/2")},
		{Name: "retarget remaining", Match: shelltest.Exact("gh", "pr", "edit", "https://github.com/acme/widget/pull/2", "-B", "main")},
		{Name: "refresh merged target", Match: shelltest.Exact("git", "fetch", "--prune", "--", "origin")},
		{Name: "restore original", Match: shelltest.Exact("git", "checkout", "feature")},
	}
	for stop := range steps {
		t.Run(steps[stop].Name, func(t *testing.T) {
			sentinel := errors.New("landing denied")
			responses := append([]shelltest.Response(nil), steps[:stop+1]...)
			responses[stop].Err = sentinel
			run := shelltest.New(t, responses...)
			e := entryForLandTest(head, url)
			e.Commit.Body = "user body\n\nstack-info: PR: " + url + ", branch: " + head + "\n"
			err := landBottomOnly(&AppContext{Git: git.New("", run), PR: pr.NewClient(run), OrigBranch: "feature", Args: CommonArgs{Remote: "origin", Target: "main"}}, stack.Stack{e, entryForLandTest("alice/stack/2", "https://github.com/acme/widget/pull/2")})
			if !errors.Is(err, sentinel) {
				t.Fatalf("error=%v", err)
			}
			for _, call := range run.Calls() {
				if len(call.Args) > 2 && call.Args[0] == "gh" && call.Args[2] == "merge" && string(call.Opts.Stdin) != "user body" {
					t.Fatalf("squash body includes metadata: %q", call.Opts.Stdin)
				}
			}
		})
	}
}

func TestLandCleanupRefreshesTargetBeforeDeletingBranches(t *testing.T) {
	for _, failFetch := range []bool{false, true} {
		t.Run(fmt.Sprintf("fetch fails=%v", failFetch), func(t *testing.T) {
			sentinel := errors.New("fetch denied")
			responses := []shelltest.Response{{Match: shelltest.Exact("git", "fetch", "--prune", "--", "origin")}}
			if failFetch {
				responses[0].Err = sentinel
			} else {
				responses = append(responses,
					shelltest.Response{Match: shelltest.Exact("git", "checkout", "feature")},
					shelltest.Response{Match: shelltest.Exact("git", "branch", "-D", "alice/stack/1")},
					shelltest.Response{Match: shelltest.Exact("git", "show-ref", "-q", "refs/heads/main")},
					shelltest.Response{Match: shelltest.Exact("git", "rebase", "origin/main", "main")},
					shelltest.Response{Match: shelltest.Exact("git", "rebase", "origin/main", "feature")},
				)
			}
			run := shelltest.New(t, responses...)
			err := landCleanup(&AppContext{Git: git.New("", run), OrigBranch: "feature", Args: CommonArgs{Remote: "origin", Target: "main"}}, stack.Stack{entryForLandTest("alice/stack/1", "https://github.com/acme/widget/pull/1")})
			if failFetch {
				if !errors.Is(err, sentinel) {
					t.Fatalf("error=%v", err)
				}
			} else if err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestLandNativePreflightBlocksMutations(t *testing.T) {
	var commits []string
	for _, n := range []int{2, 1} {
		sha := strings.Repeat(fmt.Sprint(n), 40)
		commits = append(commits, strings.Join([]string{sha, "tree " + strings.Repeat("3", 40), "author Test User <test@example.com> 1 +0000", "committer Test User <test@example.com> 1 +0000", "", fmt.Sprintf("    change %d", n), "", fmt.Sprintf("    stack-info: PR: https://github.com/acme/widget/pull/%d, branch: alice/stack/%d", n, n)}, "\n"))
	}
	// A membership read failure must stop before fetch, metadata rewriting,
	// merging, force-pushing, or branch deletion.
	responses := []shelltest.Response{}
	responses = append(responses, shelltest.Response{Match: shelltest.Exact("git", "rev-list", "--header", "^main", "HEAD"), Stdout: strings.Join(commits, "\x00")})
	responses = append(responses,
		shelltest.Response{Match: shelltest.Exact("git", "remote", "get-url", "--", "origin"), Stdout: "https://github.com/acme/widget.git"},
		shelltest.Response{Match: shelltest.Exact("gh", "api", "--include", "--method", "GET", "repos/acme/widget"), Stdout: "HTTP/2.0 403 Forbidden\n\n{\"message\":\"access denied\"}", ExitCode: 1},
	)
	run := shelltest.New(t, responses...)
	cfg := config.Defaults()
	cfg.Set("github", "native_stacks", "required")
	app := &AppContext{Config: cfg, Git: git.New("", run), PR: pr.NewClient(run), OrigBranch: "feature", Args: CommonArgs{Base: "main", Head: "HEAD", Target: "main", Remote: "origin", BranchNameTemplate: "$USERNAME/stack/$ID"}}
	err := landImpl(app, "bottom-only")
	if err == nil || !strings.Contains(err.Error(), "access denied") {
		t.Fatalf("error=%v", err)
	}
}
