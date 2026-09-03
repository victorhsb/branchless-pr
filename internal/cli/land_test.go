package cli

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/victorhsb/branchless-pr/internal/config"
	"github.com/victorhsb/branchless-pr/internal/git"
	"github.com/victorhsb/branchless-pr/internal/invocation"
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

// installFakeShellForLand sets up a fake gh script and an in-memory Git runner.
// rebaseMergeAllowed controls the GraphQL response.
// mergeQueueEnabled controls the rules API response (true -> returns a
// merge_queue rule, false -> returns empty array).
func installFakeShellForLand(t *testing.T, rebaseMergeAllowed, _ bool, mergeQueueEnabled bool) (string, *shelltest.Fake) {
	t.Helper()
	binDir := t.TempDir()
	ghLog := filepath.Join(binDir, "gh.log")

	allowed := "false"
	if rebaseMergeAllowed {
		allowed = "true"
	}
	mqRules := "[]"
	if mergeQueueEnabled {
		mqRules = `[{"type":"merge_queue","parameters":{"merge_method":"rebase_or_merge"}}]`
	}
	ghScript := `#!/bin/sh
printf '%s\n' "$*" >> "$GH_LOG"
if [ "$1" = "api" ] && [ "$2" = "graphql" ]; then
  printf '{"data":{"repository":{"rebaseMergeAllowed":` + allowed + `}}}\n'
  exit 0
fi
if [ "$1" = "api" ] && [ "$2" = "repos/acme/widget/rules/branches/main" ]; then
  printf '` + mqRules + `\n'
  exit 0
fi
exit 0
`
	if err := os.WriteFile(filepath.Join(binDir, "gh"), []byte(ghScript), 0o755); err != nil {
		t.Fatalf("write fake gh: %v", err)
	}

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
	run := shelltest.New(t, responses...)
	t.Setenv("GH_LOG", ghLog)
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))
	return ghLog, run
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
	if runtime.GOOS == "windows" {
		t.Skip("shell script fakes are Unix-only")
	}
	ghLog, run := installFakeShellForLand(t, true, false, true)

	app := &invocation.AppContext{
		Args:       invocation.CommonArgs{Remote: "origin", Target: "main"},
		Git:        git.New("", run),
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

	gh := readTestFile(t, ghLog)
	mustContain(t, gh, "api graphql")
	mustContain(t, gh, "api repos/acme/widget/rules/branches/main")
	mustContain(t, gh, "pr edit https://github.com/acme/widget/pull/1 -B main")
	mustContain(t, gh, "pr merge https://github.com/acme/widget/pull/1 --rebase --auto")

	gitLog := shellCallsLog(run)
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
	if runtime.GOOS == "windows" {
		t.Skip("shell script fakes are Unix-only")
	}
	ghLog, run := installFakeShellForLand(t, true, true, true)

	app := &invocation.AppContext{
		Args:       invocation.CommonArgs{Remote: "origin", Target: "main"},
		Git:        git.New("", run),
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

	gh := readTestFile(t, ghLog)
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

	gitLog := shellCallsLog(run)
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
	if runtime.GOOS == "windows" {
		t.Skip("shell script fakes are Unix-only")
	}
	ghLog, run := installFakeShellForLand(t, false, false, false)

	app := &invocation.AppContext{
		Args:       invocation.CommonArgs{Remote: "origin", Target: "main"},
		Git:        git.New("", run),
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
	gh := readTestFile(t, ghLog)
	if strings.Contains(gh, "pr edit") || strings.Contains(gh, "pr merge") {
		t.Fatalf("expected no PR edits/merges when rebase disallowed, gh log:\n%s", gh)
	}
	gitLog := shellCallsLog(run)
	if strings.Contains(gitLog, "fetch") || strings.Contains(gitLog, "checkout") {
		t.Fatalf("expected no fetch/checkout when rebase disallowed, git log:\n%s", gitLog)
	}
}

func TestLandWholeStackRejectedWhenMergeQueueDisabled(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell script fakes are Unix-only")
	}
	ghLog, run := installFakeShellForLand(t, true, false, false)

	app := &invocation.AppContext{
		Args:       invocation.CommonArgs{Remote: "origin", Target: "main"},
		Git:        git.New("", run),
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
	gh := readTestFile(t, ghLog)
	if strings.Contains(gh, "pr edit") || strings.Contains(gh, "pr merge") {
		t.Fatalf("expected no PR edits/merges when merge queue disabled, gh log:\n%s", gh)
	}
	gitLog := shellCallsLog(run)
	if strings.Contains(gitLog, "fetch") || strings.Contains(gitLog, "checkout") {
		t.Fatalf("expected no fetch/checkout when merge queue disabled, git log:\n%s", gitLog)
	}
}

func TestLandWholeStackUnknownMergeQueueProceedsAndNormalizes(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell script fakes are Unix-only")
	}
	binDir := t.TempDir()
	ghLog := filepath.Join(binDir, "gh.log")

	ghScript := `#!/bin/sh
printf '%s\n' "$*" >> "$GH_LOG"
if [ "$1" = "api" ] && [ "$2" = "graphql" ]; then
  printf '{"data":{"repository":{"rebaseMergeAllowed":true}}}\n'
  exit 0
fi
if [ "$1" = "api" ] && [ "$2" = "repos/acme/widget/rules/branches/main" ]; then
  printf '{"message":"Not Found"}\n'
  exit 1
fi
if [ "$1" = "pr" ] && [ "$2" = "merge" ]; then
  printf 'merge queue is not enabled for this branch\n' >&2
  exit 1
fi
exit 0
`
	if err := os.WriteFile(filepath.Join(binDir, "gh"), []byte(ghScript), 0o755); err != nil {
		t.Fatalf("write fake gh: %v", err)
	}

	run := shelltest.New(t,
		shelltest.Response{
			Match:  shelltest.Exact("git", "remote", "get-url", "--", "origin"),
			Stdout: "https://github.com/acme/widget.git\n",
		},
		shelltest.Response{Match: shelltest.Exact("git", "fetch", "--prune", "--", "origin")},
	)
	t.Setenv("GH_LOG", ghLog)
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))

	app := &invocation.AppContext{
		Args:       invocation.CommonArgs{Remote: "origin", Target: "main"},
		Git:        git.New("", run),
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
