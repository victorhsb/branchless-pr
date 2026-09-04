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
	got, err := rebaseMergeAllowedWith("acme", "widget", func(query string, fields map[string]string) ([]byte, error) {
		if !strings.Contains(query, "rebaseMergeAllowed") {
			t.Fatalf("query missing rebaseMergeAllowed: %q", query)
		}
		if fields["owner"] != "acme" || fields["repo"] != "widget" {
			t.Fatalf("fields = %v, want acme/widget", fields)
		}
		return []byte(`{"data":{"repository":{"rebaseMergeAllowed":true}}}`), nil
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !got {
		t.Fatalf("expected true, got false")
	}
}

func TestRebaseMergeAllowedReturnsFalse(t *testing.T) {
	got, err := rebaseMergeAllowedWith("acme", "widget", func(query string, fields map[string]string) ([]byte, error) {
		return []byte(`{"data":{"repository":{"rebaseMergeAllowed":false}}}`), nil
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got {
		t.Fatalf("expected false, got true")
	}
}

func TestRebaseMergeAllowedPropagatesAPIError(t *testing.T) {
	want := errors.New("boom")
	_, err := rebaseMergeAllowedWith("acme", "widget", func(query string, fields map[string]string) ([]byte, error) {
		return nil, want
	})
	if err == nil || !strings.Contains(err.Error(), "boom") {
		t.Fatalf("expected error containing %q, got %v", "boom", err)
	}
}

func TestRebaseMergeAllowedSurfacesGraphQLErrors(t *testing.T) {
	_, err := rebaseMergeAllowedWith("acme", "widget", func(query string, fields map[string]string) ([]byte, error) {
		return []byte(`{"errors":[{"message":"Could not resolve to a Repository"}]}`), nil
	})
	if err == nil || !strings.Contains(err.Error(), "Could not resolve") {
		t.Fatalf("expected graphql error, got %v", err)
	}
}

func TestRebaseMergeAllowedSurfacesParseErrors(t *testing.T) {
	_, err := rebaseMergeAllowedWith("acme", "widget", func(query string, fields map[string]string) ([]byte, error) {
		return []byte(`{not-json}`), nil
	})
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
	got, err := mergeQueueEnabledWith("acme", "widget", "main", func(owner, repo, branch string) ([]byte, error) {
		return []byte(`[{"type":"merge_queue","parameters":{}}]`), nil
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != MergeQueueStatusEnabled {
		t.Fatalf("status = %q, want %q", got, MergeQueueStatusEnabled)
	}
}

func TestMergeQueueEnabledReturnsDisabled(t *testing.T) {
	got, err := mergeQueueEnabledWith("acme", "widget", "main", func(owner, repo, branch string) ([]byte, error) {
		return []byte(`[{"type":"pull_request","parameters":{}}]`), nil
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != MergeQueueStatusDisabled {
		t.Fatalf("status = %q, want %q", got, MergeQueueStatusDisabled)
	}
}

func TestMergeQueueEnabledReturnsUnknownOnAPIError(t *testing.T) {
	got, err := mergeQueueEnabledWith("acme", "widget", "main", func(owner, repo, branch string) ([]byte, error) {
		return nil, errors.New("boom")
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != MergeQueueStatusUnknown {
		t.Fatalf("status = %q, want %q", got, MergeQueueStatusUnknown)
	}
}

func TestMergeQueueEnabledReturnsUnknownOnParseError(t *testing.T) {
	got, err := mergeQueueEnabledWith("acme", "widget", "main", func(owner, repo, branch string) ([]byte, error) {
		return []byte(`{not-json}`), nil
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != MergeQueueStatusUnknown {
		t.Fatalf("status = %q, want %q", got, MergeQueueStatusUnknown)
	}
}
