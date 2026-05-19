package diagnose

import (
	"errors"
	"strings"
	"testing"

	"github.com/victorhsb/branchless-pr/internal/shell"
)

type fakeRunner struct {
	outputs     map[string]string
	errors      map[string]error
	runStdout   map[string][]byte
	runStderr   map[string][]byte
	runErrs     map[string]error
	lookPathErr error
	rejectGH    bool
	calls       [][]string
}

func (f *fakeRunner) Output(args []string, opts shell.RunOpts) (string, error) {
	f.calls = append(f.calls, args)
	if args[0] == "gh" && f.rejectGH {
		return "", errors.New("unexpected gh invocation")
	}
	key := strings.Join(args, "\x00")
	if err := f.errors[key]; err != nil {
		return "", err
	}
	return f.outputs[key], nil
}

func (f *fakeRunner) Run(args []string, opts shell.RunOpts) ([]byte, []byte, error) {
	f.calls = append(f.calls, args)
	if args[0] == "gh" && f.rejectGH {
		return nil, nil, errors.New("unexpected gh invocation")
	}
	key := strings.Join(args, "\x00")
	return f.runStdout[key], f.runStderr[key], f.runErrs[key]
}

func (f *fakeRunner) LookPath(file string) (string, error) {
	if f.lookPathErr != nil {
		return "", f.lookPathErr
	}
	return "/usr/bin/" + file, nil
}

func TestOfflineRunUsesOnlyReadOnlyGitCommandsAndDoesNotInvokeGH(t *testing.T) {
	base := strings.Repeat("a", 40)
	f := &fakeRunner{outputs: map[string]string{
		key("git", "rev-parse", "--show-toplevel"):           "/repo",
		key("git", "rev-parse", "--abbrev-ref", "HEAD"):      "feature",
		key("git", "status", "--porcelain"):                  "",
		key("git", "rev-parse", "--git-dir"):                 t.TempDir(),
		key("git", "rev-parse", "--verify", "HEAD"):          strings.Repeat("b", 40),
		key("git", "merge-base", "HEAD", "origin/main"):      base,
		key("git", "rev-parse", "--verify", "origin/main"):   base,
		key("git", "rev-parse", "--verify", base):            base,
		key("git", "rev-list", "--header", "^"+base, "HEAD"): "",
	}, rejectGH: true}
	report := Run(Options{Remote: "origin", Target: "main", Head: "HEAD", BranchNameTemplate: "$USERNAME/stack", Runner: f})
	if report.Repo.Online {
		t.Fatal("online unexpectedly true")
	}
	for _, call := range f.calls {
		if call[0] == "gh" {
			t.Fatalf("offline mode invoked gh: %v", call)
		}
		if call[0] == "git" && mutatingGitCall(call[1:]) {
			t.Fatalf("diagnose used mutating git command: %v", call)
		}
	}
	if _, ok := findCheck(report.Checks, "online_pr_state"); !ok {
		t.Fatal("missing online_pr_state check")
	}
	c, ok := findCheck(report.Checks, "github_availability")
	if !ok {
		t.Fatal("missing github_availability check")
	}
	if c.Status != StatusUnknown || !strings.Contains(c.Message, "--online") {
		t.Fatalf("offline github_availability = %+v", c)
	}
}

func TestOnlineGitHubAvailabilityOK(t *testing.T) {
	f := &fakeRunner{outputs: map[string]string{
		key("git", "rev-parse", "--show-toplevel"): "/repo",
	}}
	report := Run(Options{Online: true, Remote: "origin", Target: "main", Head: "HEAD", BranchNameTemplate: "$USERNAME/stack", Runner: f})
	c, ok := findCheck(report.Checks, "github_availability")
	if !ok {
		t.Fatal("missing github_availability check")
	}
	if c.Status != StatusOK || !strings.Contains(c.Message, "reachable") {
		t.Fatalf("github_availability = %+v", c)
	}
	if !called(f.calls, "gh", "api", "/rate_limit") {
		t.Fatalf("availability probe was not invoked: %v", f.calls)
	}
}

func TestGitHubAvailabilitySkipsProbeWhenGHMissing(t *testing.T) {
	f := &fakeRunner{lookPathErr: errors.New("not found")}
	report := Run(Options{Online: true, Remote: "origin", Target: "main", Head: "HEAD", BranchNameTemplate: "$USERNAME/stack", Runner: f})
	c, ok := findCheck(report.Checks, "github_availability")
	if !ok {
		t.Fatal("missing github_availability check")
	}
	if c.Status != StatusUnknown || !strings.Contains(c.Message, "gh is not installed") {
		t.Fatalf("github_availability = %+v", c)
	}
	if called(f.calls, "gh", "api", "/rate_limit") {
		t.Fatalf("availability probe should not run without gh: %v", f.calls)
	}
}

func TestGitHubAvailabilityOutageIsBlockingAndSkipsPRState(t *testing.T) {
	base := strings.Repeat("a", 40)
	head := strings.Repeat("b", 40)
	f := &fakeRunner{
		outputs: map[string]string{
			key("git", "rev-parse", "--show-toplevel"):           "/repo",
			key("git", "rev-parse", "--abbrev-ref", "HEAD"):      "feature",
			key("git", "status", "--porcelain"):                  "",
			key("git", "rev-parse", "--git-dir"):                 t.TempDir(),
			key("git", "rev-parse", "--verify", "HEAD"):          head,
			key("git", "merge-base", "HEAD", "origin/main"):      base,
			key("git", "rev-parse", "--verify", "origin/main"):   base,
			key("git", "rev-parse", "--verify", base):            base,
			key("git", "rev-list", "--header", "^"+base, "HEAD"): commitHeader(head),
		},
		runStderr: map[string][]byte{
			key("gh", "api", "/rate_limit"): []byte("HTTP 503: Service Unavailable"),
		},
		runErrs: map[string]error{
			key("gh", "api", "/rate_limit"): errors.New("exit status 1"),
		},
	}
	report := Run(Options{Online: true, Remote: "origin", Target: "main", Head: "HEAD", BranchNameTemplate: "$USERNAME/stack", Runner: f})
	c, ok := findCheck(report.Checks, "github_availability")
	if !ok {
		t.Fatal("missing github_availability check")
	}
	if c.Status != StatusBlocking {
		t.Fatalf("github_availability status = %s, want blocking: %+v", c.Status, c)
	}
	for _, want := range []string{"submit", "land", "abandon"} {
		if !contains(c.Blocks, want) {
			t.Fatalf("github_availability blocks %v, want %s", c.Blocks, want)
		}
	}
	if c.SuggestedFix == "" {
		t.Fatalf("missing suggested fix: %+v", c)
	}
	pr, ok := findCheck(report.Checks, "online_pr_state")
	if !ok {
		t.Fatal("missing online_pr_state check")
	}
	if pr.Status != StatusUnknown || !strings.Contains(pr.Message, "GitHub appears unavailable") {
		t.Fatalf("online_pr_state = %+v", pr)
	}
	if calledPrefix(f.calls, "gh", "pr", "view") {
		t.Fatalf("PR state query should be skipped during outage: %v", f.calls)
	}
}

func TestGitHubAvailabilityAuthFailureIsNotOutage(t *testing.T) {
	f := &fakeRunner{
		runStderr: map[string][]byte{
			key("gh", "api", "/rate_limit"): []byte("HTTP 401: authentication required; run gh auth login"),
		},
		runErrs: map[string]error{
			key("gh", "api", "/rate_limit"): errors.New("exit status 1"),
		},
	}
	report := Run(Options{Online: true, Remote: "origin", Target: "main", Head: "HEAD", BranchNameTemplate: "$USERNAME/stack", Runner: f})
	c, ok := findCheck(report.Checks, "github_availability")
	if !ok {
		t.Fatal("missing github_availability check")
	}
	if c.Status == StatusBlocking {
		t.Fatalf("auth failure classified as outage: %+v", c)
	}
	if !strings.Contains(c.Message, "authentication") {
		t.Fatalf("github_availability = %+v", c)
	}
}

func TestOnlinePRStateReportsRepositorySpecificFailureWhenGitHubReachable(t *testing.T) {
	base := strings.Repeat("a", 40)
	head := strings.Repeat("b", 40)
	f := &fakeRunner{
		outputs: map[string]string{
			key("git", "rev-parse", "--show-toplevel"):           "/repo",
			key("git", "rev-parse", "--abbrev-ref", "HEAD"):      "feature",
			key("git", "status", "--porcelain"):                  "",
			key("git", "rev-parse", "--git-dir"):                 t.TempDir(),
			key("git", "rev-parse", "--verify", "HEAD"):          head,
			key("git", "merge-base", "HEAD", "origin/main"):      base,
			key("git", "rev-parse", "--verify", "origin/main"):   base,
			key("git", "rev-parse", "--verify", base):            base,
			key("git", "rev-list", "--header", "^"+base, "HEAD"): commitHeader(head),
		},
		errors: map[string]error{
			key("gh", "pr", "view", "https://github.com/foo/bar/pull/42", "--json", "baseRefName,headRefName,number,state,mergeStateStatus,isDraft"): errors.New("HTTP 404: Not Found"),
		},
	}
	report := Run(Options{Online: true, Remote: "origin", Target: "main", Head: "HEAD", BranchNameTemplate: "$USERNAME/stack", Runner: f})
	avail, ok := findCheck(report.Checks, "github_availability")
	if !ok || avail.Status != StatusOK {
		t.Fatalf("github_availability = %+v, ok=%v", avail, ok)
	}
	pr, ok := findCheck(report.Checks, "online_pr_state")
	if !ok {
		t.Fatal("missing online_pr_state check")
	}
	if pr.Status != StatusUnknown || !strings.Contains(pr.Message, "could not query PR state") {
		t.Fatalf("online_pr_state = %+v", pr)
	}
}

func TestRecommendationDecisionTree(t *testing.T) {
	tests := []struct {
		name string
		rep  Report
		want string
	}{
		{name: "not git", rep: Report{Checks: []CheckEntry{{ID: "git_repository", Status: StatusBlocking}}}, want: "cd <git-repository>"},
		{name: "rebase", rep: Report{Stack: StackSummary{Size: 1}, Checks: []CheckEntry{{ID: "git_repository", Status: StatusOK}, {ID: "rebase_in_progress", Status: StatusBlocking}}}, want: "git rebase --continue | git rebase --abort"},
		{name: "empty", rep: Report{Checks: []CheckEntry{{ID: "git_repository", Status: StatusOK}, {ID: "rebase_in_progress", Status: StatusOK}}}, want: "create commits on top of the target branch"},
		{name: "dirty", rep: Report{Stack: StackSummary{Size: 1}, Checks: []CheckEntry{{ID: "git_repository", Status: StatusOK}, {ID: "rebase_in_progress", Status: StatusOK}, {ID: "working_tree_clean", Status: StatusBlocking}}}, want: "clean the working tree (commit, stash, or revert changes)"},
		{name: "github outage", rep: Report{Stack: StackSummary{Size: 2, EntriesWithPR: 1, EntriesMissingPR: 1}, Checks: []CheckEntry{{ID: "git_repository", Status: StatusOK}, {ID: "rebase_in_progress", Status: StatusOK}, {ID: "working_tree_clean", Status: StatusOK}, {ID: "github_availability", Status: StatusBlocking}}}, want: "wait for GitHub availability or inspect local state only"},
		{name: "missing PRs", rep: Report{Stack: StackSummary{Size: 2, EntriesWithPR: 1, EntriesMissingPR: 1}, Checks: []CheckEntry{{ID: "git_repository", Status: StatusOK}, {ID: "rebase_in_progress", Status: StatusOK}, {ID: "working_tree_clean", Status: StatusOK}}}, want: "stack-pr submit --dry-run"},
		{name: "submitted", rep: Report{Stack: StackSummary{Size: 2, EntriesWithPR: 2}, Checks: []CheckEntry{{ID: "git_repository", Status: StatusOK}, {ID: "rebase_in_progress", Status: StatusOK}, {ID: "working_tree_clean", Status: StatusOK}}}, want: "stack-pr view"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := BuildRecommendation(tt.rep)
			if got.Command != tt.want {
				t.Fatalf("command = %q, want %q", got.Command, tt.want)
			}
			if got.SideEffects && !got.RequiresConfirmation {
				t.Fatalf("side effects without confirmation: %+v", got)
			}
		})
	}
}

func TestGitHubOutageRecommendationPriority(t *testing.T) {
	dirty := BuildRecommendation(Report{
		Stack: StackSummary{Size: 1, EntriesMissingPR: 1},
		Checks: []CheckEntry{
			{ID: "git_repository", Status: StatusOK},
			{ID: "rebase_in_progress", Status: StatusOK},
			{ID: "working_tree_clean", Status: StatusBlocking},
			{ID: "github_availability", Status: StatusBlocking},
		},
	})
	if dirty.Command != "clean the working tree (commit, stash, or revert changes)" {
		t.Fatalf("dirty priority command = %q", dirty.Command)
	}

	outage := BuildRecommendation(Report{
		Stack: StackSummary{Size: 1, EntriesMissingPR: 1},
		Checks: []CheckEntry{
			{ID: "git_repository", Status: StatusOK},
			{ID: "rebase_in_progress", Status: StatusOK},
			{ID: "working_tree_clean", Status: StatusOK},
			{ID: "github_availability", Status: StatusBlocking},
		},
	})
	switch outage.Command {
	case "stack-pr submit", "stack-pr submit --dry-run", "stack-pr land", "stack-pr abandon":
		t.Fatalf("outage recommendation used mutating stack command: %+v", outage)
	}
	if !strings.Contains(outage.Reason, "live GitHub state cannot currently be trusted") {
		t.Fatalf("outage reason does not explain remote trust: %+v", outage)
	}
}

func TestLandIsOnlyPotentialNextAction(t *testing.T) {
	rec := BuildRecommendation(Report{Stack: StackSummary{Size: 1, EntriesWithPR: 1}, Checks: []CheckEntry{{ID: "git_repository", Status: StatusOK}, {ID: "working_tree_clean", Status: StatusOK}, {ID: "rebase_in_progress", Status: StatusOK}}})
	if rec.Command == "stack-pr land" {
		t.Fatal("land was primary recommendation")
	}
	if len(rec.PotentialNextActions) != 1 || rec.PotentialNextActions[0].Command != "stack-pr land" {
		t.Fatalf("missing land potential action: %+v", rec)
	}
	land := rec.PotentialNextActions[0]
	if !land.SideEffects || !land.RequiresConfirmation {
		t.Fatalf("land safety metadata not conservative: %+v", land)
	}
}

func TestOverallStatusAndPanicHarness(t *testing.T) {
	if got := overallStatus([]CheckEntry{{Status: StatusOK}, {Status: StatusWarning}}); got != StatusWarning {
		t.Fatalf("overallStatus warning = %s", got)
	}
	if got := overallStatus([]CheckEntry{{Status: StatusUnknown}, {Status: StatusWarning}}); got != StatusUnknown {
		t.Fatalf("overallStatus unknown = %s", got)
	}
	if got := overallStatus([]CheckEntry{{Status: StatusUnknown}, {Status: StatusBlocking}}); got != StatusBlocking {
		t.Fatalf("overallStatus blocking = %s", got)
	}

	i := &inspector{}
	i.add("panic_check", func() (CheckEntry, error) { panic("boom") })
	c, ok := findCheck(i.report.Checks, "panic_check")
	if !ok || c.Status != StatusUnknown || !strings.Contains(c.Message, "panicked") {
		t.Fatalf("panic not converted to unknown: %+v", i.report.Checks)
	}
}

func TestRenderJSONEnvelopeV1(t *testing.T) {
	report := Report{
		SchemaVersion: SchemaVersion,
		Status:        StatusOK,
		Repo:          RepoContext{Remote: "origin", Target: "main", Head: "HEAD", BranchNameTemplate: "$USERNAME/stack"},
		Stack:         StackSummary{Size: 1, EntriesWithPR: 1},
		Checks:        []CheckEntry{{ID: "git_repository", Status: StatusOK, Message: "inside"}},
		Recommendation: Recommendation{Command: "stack-pr view", Reason: "inspect", SideEffects: false,
			RequiresConfirmation: false},
	}
	out, err := RenderJSON(report)
	if err != nil {
		t.Fatal(err)
	}
	want := `{
  "schema_version": "1",
  "status": "ok",
  "repo": {
    "remote": "origin",
    "target": "main",
    "head": "HEAD",
    "branch_name_template": "$USERNAME/stack",
    "online": false
  },
  "stack": {
    "size": 1,
    "entries_with_pr": 1,
    "entries_missing_pr": 0
  },
  "checks": [
    {
      "id": "git_repository",
      "status": "ok",
      "message": "inside"
    }
  ],
  "recommendation": {
    "command": "stack-pr view",
    "reason": "inspect",
    "side_effects": false,
    "requires_confirmation": false
  }
}`
	if string(out) != want {
		t.Fatalf("JSON changed:\n%s", out)
	}
}

func mutatingGitCall(args []string) bool {
	if len(args) == 0 {
		return false
	}
	switch args[0] {
	case "add", "commit", "checkout", "switch", "rebase", "stash", "push", "reset", "branch", "fetch":
		return true
	}
	return false
}

func key(args ...string) string { return strings.Join(args, "\x00") }

func called(calls [][]string, args ...string) bool {
	for _, call := range calls {
		if len(call) != len(args) {
			continue
		}
		match := true
		for i := range args {
			if call[i] != args[i] {
				match = false
				break
			}
		}
		if match {
			return true
		}
	}
	return false
}

func calledPrefix(calls [][]string, args ...string) bool {
	for _, call := range calls {
		if len(call) < len(args) {
			continue
		}
		match := true
		for i := range args {
			if call[i] != args[i] {
				match = false
				break
			}
		}
		if match {
			return true
		}
	}
	return false
}

func contains(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

func commitHeader(sha string) string {
	return sha + `
tree ` + strings.Repeat("c", 40) + `
author Alice <alice@example.com> 0 +0000

    Add change
    
    stack-info: PR: https://github.com/foo/bar/pull/42, branch: alice/stack/1
`
}
