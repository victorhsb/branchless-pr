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
	runErrs     map[string]error
	lookPathErr error
	calls       [][]string
}

func (f *fakeRunner) Output(args []string, opts shell.RunOpts) (string, error) {
	f.calls = append(f.calls, args)
	if args[0] == "gh" {
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
	if args[0] == "gh" {
		return nil, nil, errors.New("unexpected gh invocation")
	}
	key := strings.Join(args, "\x00")
	return nil, nil, f.runErrs[key]
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
	}}
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
