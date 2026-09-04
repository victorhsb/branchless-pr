package pr

import (
	"errors"
	"strings"
	"testing"

	"github.com/victorhsb/branchless-pr/internal/shell/shelltest"
)

func TestMergeRebaseInvokesGh(t *testing.T) {
	run := shelltest.New(t, shelltest.Response{
		Match: shelltest.Exact("gh", "pr", "merge", "https://github.com/acme/repo/pull/42", "--rebase"),
	})

	if err := NewClient(run).MergeRebase("https://github.com/acme/repo/pull/42"); err != nil {
		t.Fatalf("MergeRebase returned error: %v", err)
	}
}

func TestRebaseMergeAllowedReturnsTrue(t *testing.T) {
	run := shelltest.New(t, shelltest.Response{
		Match:  shelltest.Prefix("gh", "api", "graphql", "-f", "query=query($owner: String!, $repo: String!) { repository(owner: $owner, name: $repo) { rebaseMergeAllowed } }"),
		Stdout: `{"data":{"repository":{"rebaseMergeAllowed":true}}}`,
	})
	got, err := NewClient(run).RebaseMergeAllowed("acme", "widget")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !got {
		t.Fatalf("expected true, got false")
	}
	args := strings.Join(run.Calls()[0].Args, "\n")
	if !strings.Contains(args, "owner=acme") || !strings.Contains(args, "repo=widget") {
		t.Fatalf("args missing repository fields: %v", run.Calls()[0].Args)
	}
}

func TestRebaseMergeAllowedReturnsFalse(t *testing.T) {
	run := shelltest.New(t, shelltest.Response{
		Match:  shelltest.Prefix("gh", "api", "graphql"),
		Stdout: `{"data":{"repository":{"rebaseMergeAllowed":false}}}`,
	})
	got, err := NewClient(run).RebaseMergeAllowed("acme", "widget")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got {
		t.Fatalf("expected false, got true")
	}
}

func TestRebaseMergeAllowedPropagatesAPIError(t *testing.T) {
	run := shelltest.New(t, shelltest.Response{
		Match: shelltest.Prefix("gh", "api", "graphql"),
		Err:   errors.New("boom"),
	})
	_, err := NewClient(run).RebaseMergeAllowed("acme", "widget")
	if err == nil || !strings.Contains(err.Error(), "boom") {
		t.Fatalf("expected error containing %q, got %v", "boom", err)
	}
}

func TestRebaseMergeAllowedSurfacesGraphQLErrors(t *testing.T) {
	run := shelltest.New(t, shelltest.Response{
		Match:  shelltest.Prefix("gh", "api", "graphql"),
		Stdout: `{"errors":[{"message":"Could not resolve to a Repository"}]}`,
	})
	_, err := NewClient(run).RebaseMergeAllowed("acme", "widget")
	if err == nil || !strings.Contains(err.Error(), "Could not resolve") {
		t.Fatalf("expected graphql error, got %v", err)
	}
}

func TestRebaseMergeAllowedSurfacesParseErrors(t *testing.T) {
	run := shelltest.New(t, shelltest.Response{
		Match:  shelltest.Prefix("gh", "api", "graphql"),
		Stdout: `{not-json}`,
	})
	_, err := NewClient(run).RebaseMergeAllowed("acme", "widget")
	if err == nil || !strings.Contains(err.Error(), "parse") {
		t.Fatalf("expected parse error, got %v", err)
	}
}

func TestMergeRebaseAutoInvokesGh(t *testing.T) {
	run := shelltest.New(t, shelltest.Response{
		Match: shelltest.Exact("gh", "pr", "merge", "https://github.com/acme/repo/pull/42", "--rebase", "--auto"),
	})

	if err := NewClient(run).MergeRebaseAuto("https://github.com/acme/repo/pull/42"); err != nil {
		t.Fatalf("MergeRebaseAuto returned error: %v", err)
	}
}

func TestMergeRebaseAutoNormalizesDisabledQueueError(t *testing.T) {
	run := shelltest.New(t, shelltest.Response{
		Match:    shelltest.Exact("gh", "pr", "merge", "https://github.com/acme/repo/pull/42", "--rebase", "--auto"),
		Stderr:   "merge queue is not enabled for this branch\n",
		ExitCode: 1,
	})

	err := NewClient(run).MergeRebaseAuto("https://github.com/acme/repo/pull/42")
	if err == nil {
		t.Fatalf("expected error")
	}
	want := "ERROR: --whole-stack only works for repositories with merge queue enabled"
	if err.Error() != want {
		t.Fatalf("error = %q, want %q", err.Error(), want)
	}
}

func TestMergeQueueEnabledReturnsEnabled(t *testing.T) {
	run := shelltest.New(t, shelltest.Response{
		Match:  shelltest.Exact("gh", "api", "repos/acme/widget/rules/branches/main"),
		Stdout: `[{"type":"merge_queue","parameters":{}}]`,
	})
	got, err := NewClient(run).MergeQueueEnabled("acme", "widget", "main")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != MergeQueueStatusEnabled {
		t.Fatalf("status = %q, want %q", got, MergeQueueStatusEnabled)
	}
}

func TestMergeQueueEnabledReturnsDisabled(t *testing.T) {
	run := shelltest.New(t, shelltest.Response{
		Match:  shelltest.Exact("gh", "api", "repos/acme/widget/rules/branches/main"),
		Stdout: `[{"type":"pull_request","parameters":{}}]`,
	})
	got, err := NewClient(run).MergeQueueEnabled("acme", "widget", "main")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != MergeQueueStatusDisabled {
		t.Fatalf("status = %q, want %q", got, MergeQueueStatusDisabled)
	}
}

func TestMergeQueueEnabledReturnsUnknownOnAPIError(t *testing.T) {
	run := shelltest.New(t, shelltest.Response{
		Match:    shelltest.Exact("gh", "api", "repos/acme/widget/rules/branches/main"),
		ExitCode: 1,
	})
	got, err := NewClient(run).MergeQueueEnabled("acme", "widget", "main")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != MergeQueueStatusUnknown {
		t.Fatalf("status = %q, want %q", got, MergeQueueStatusUnknown)
	}
}

func TestMergeQueueEnabledReturnsUnknownOnParseError(t *testing.T) {
	run := shelltest.New(t, shelltest.Response{
		Match:  shelltest.Exact("gh", "api", "repos/acme/widget/rules/branches/main"),
		Stdout: `{not-json}`,
	})
	got, err := NewClient(run).MergeQueueEnabled("acme", "widget", "main")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != MergeQueueStatusUnknown {
		t.Fatalf("status = %q, want %q", got, MergeQueueStatusUnknown)
	}
}
