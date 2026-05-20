package cli

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/victorhsb/branchless-pr/internal/pr"
)

func TestChecksCmdExposesFlags(t *testing.T) {
	cmd := checksCmd()
	if got := cmd.Use; got != "checks" {
		t.Fatalf("Use = %q, want checks", got)
	}
	for _, name := range []string{"format", "failed-only", "required-only", "pr", "commit"} {
		if cmd.Flags().Lookup(name) == nil {
			t.Fatalf("--%s flag not registered", name)
		}
	}
}

func TestRunChecksRejectsUnknownFormat(t *testing.T) {
	err := runChecksWithFetcher(commentsTestApp(t.TempDir(), "base", "head"), checksOptions{format: "xml"}, &bytes.Buffer{}, nil)
	if err == nil || !strings.Contains(err.Error(), `unknown checks format "xml"`) {
		t.Fatalf("err = %v", err)
	}
}

func TestFilterChecksRequiredAndFailed(t *testing.T) {
	checks := []pr.Check{
		{ID: "required-failed", Required: pr.RequiredTrue, Conclusion: "failure"},
		{ID: "optional-failed", Required: pr.RequiredFalse, Conclusion: "failure"},
		{ID: "required-ok", Required: pr.RequiredTrue, Conclusion: "success"},
		{ID: "unknown-pending", Required: pr.RequiredUnknown, Status: "in_progress"},
	}
	got := filterChecks(checks, checksOptions{requiredOnly: true})
	if len(got) != 2 || got[0].ID != "required-failed" || got[1].ID != "required-ok" {
		t.Fatalf("required-only = %#v", got)
	}
	got = filterChecks(checks, checksOptions{failedOnly: true})
	if len(got) != 2 || got[0].ID != "required-failed" || got[1].ID != "optional-failed" {
		t.Fatalf("failed-only = %#v", got)
	}
}

func TestWriteChecksReportJSONIsSingleObject(t *testing.T) {
	report := &checksReport{
		SchemaVersion: "1",
		Command:       "stack-pr checks",
		Repository:    "/repo",
		Range:         checksRange{Base: "main", Head: "HEAD", Remote: "origin", Target: "main"},
		Stack: []checksStackEntry{{
			Index: 1, Commit: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", ShortSHA: "aaaaaaaaaaaa", Title: "First", Status: "fetched",
		}},
		PullRequests: []checksPullRequestReport{{
			Index: 1, Commit: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", ShortSHA: "aaaaaaaaaaaa", Title: "First", Status: "fetched",
			Checks: []pr.Check{{ID: "github-actions:ci.yml:test", Provider: pr.CheckProviderGitHubActions, Name: "test", Status: "completed", Conclusion: "failure", Required: pr.RequiredUnknown}},
		}},
		FailedChecks: []failedCheckSummary{{ID: "github-actions:ci.yml:test", Commit: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", ShortSHA: "aaaaaaaaaaaa", Name: "test", Conclusion: "failure"}},
	}
	var out bytes.Buffer
	if err := writeChecksReport(&out, report, "json"); err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(out.Bytes(), []byte("\x1b")) {
		t.Fatalf("json contains ANSI escape: %q", out.String())
	}
	var payload struct {
		SchemaVersion string `json:"schema_version"`
		Command       string `json:"command"`
		FailedChecks  []struct {
			ID string `json:"id"`
		} `json:"failed_checks"`
		PullRequests []struct {
			Checks []struct {
				ID string `json:"id"`
			} `json:"checks"`
		} `json:"pull_requests"`
	}
	if err := json.Unmarshal(out.Bytes(), &payload); err != nil {
		t.Fatalf("invalid json: %v\n%s", err, out.String())
	}
	if payload.SchemaVersion != "1" || payload.Command != "stack-pr checks" {
		t.Fatalf("payload metadata = %#v", payload)
	}
	if payload.FailedChecks[0].ID != "github-actions:ci.yml:test" || payload.PullRequests[0].Checks[0].ID != "github-actions:ci.yml:test" {
		t.Fatalf("payload checks = %#v", payload)
	}
}

func TestWriteChecksReportTextCoversFailuresAndCommentSummary(t *testing.T) {
	report := &checksReport{
		SchemaVersion: "1",
		Command:       "stack-pr checks",
		Range:         checksRange{Base: "main", Head: "HEAD", Remote: "origin", Target: "main"},
		Stack: []checksStackEntry{{
			Index: 1, Commit: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", ShortSHA: "aaaaaaaaaaaa", Title: "First", Status: "fetched",
		}},
		PullRequests: []checksPullRequestReport{{
			Index: 1, Commit: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", ShortSHA: "aaaaaaaaaaaa", Title: "First", Status: "fetched", PRNumber: 7,
			Checks: []pr.Check{{ID: "github-actions:ci.yml:test", Provider: pr.CheckProviderGitHubActions, Name: "test", Status: "completed", Conclusion: "failure", Required: pr.RequiredUnknown, URL: "https://example.test/check"}},
			CommentSummary: pr.CommentSummary{
				ConversationCount:  1,
				ReviewCount:        1,
				ReviewCommentCount: 1,
				RequestedChanges:   1,
				InspectCommand:     "stack-pr comments",
				Snippets:           []pr.CommentSnippet{{Kind: pr.CommentKindConversation, Author: "alice", Body: "please fix"}},
			},
		}},
		FailedChecks: []failedCheckSummary{{ID: "github-actions:ci.yml:test", PRNumber: 7, Commit: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", ShortSHA: "aaaaaaaaaaaa", Name: "test", Conclusion: "failure", URL: "https://example.test/check"}},
	}
	var out bytes.Buffer
	if err := writeChecksReport(&out, report, "text"); err != nil {
		t.Fatal(err)
	}
	text := out.String()
	for _, want := range []string{"# stack-pr checks", "## Failed checks", "github-actions:ci.yml:test", "Comments:", "stack-pr comments", "please fix"} {
		if !strings.Contains(text, want) {
			t.Fatalf("text missing %q:\n%s", want, text)
		}
	}
}

func TestBuildChecksReportHandlesMissingFilteringFailureAndAuth(t *testing.T) {
	repoDir, base, head := createCommentsTestRepo(t)
	chdirForTest(t, repoDir)
	app := commentsTestApp(repoDir, base, head)

	report, err := buildChecksReport(app, checksOptions{}, func(prRef string) (*pr.PullRequestChecks, error) {
		if strings.Contains(prRef, "/pull/7") {
			return &pr.PullRequestChecks{
				Number: 7,
				URL:    prRef,
				Checks: []pr.Check{
					{ID: "github-actions:ci.yml:test", Provider: pr.CheckProviderGitHubActions, Name: "test", Status: "completed", Conclusion: "failure", Required: pr.RequiredUnknown},
					{ID: "github-actions:ci.yml:lint", Provider: pr.CheckProviderGitHubActions, Name: "lint", Status: "completed", Conclusion: "success", Required: pr.RequiredFalse},
				},
				CommentSummary: pr.CommentSummary{ConversationCount: 1, InspectCommand: "stack-pr comments"},
			}, nil
		}
		return nil, errors.New("not found")
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(report.PullRequests) != 3 {
		t.Fatalf("pull request groups = %d, want 3", len(report.PullRequests))
	}
	if report.PullRequests[0].Status != "missing" {
		t.Fatalf("first status = %q, want missing", report.PullRequests[0].Status)
	}
	if report.PullRequests[1].Status != "fetched" || len(report.PullRequests[1].Checks) != 2 || len(report.FailedChecks) != 1 {
		t.Fatalf("second entry/failed summary = %#v / %#v", report.PullRequests[1], report.FailedChecks)
	}
	if report.PullRequests[2].Status != "failed" {
		t.Fatalf("third status = %q, want failed", report.PullRequests[2].Status)
	}

	filtered, err := buildChecksReport(app, checksOptions{failedOnly: true}, func(prRef string) (*pr.PullRequestChecks, error) {
		return &pr.PullRequestChecks{
			Number: 7,
			URL:    prRef,
			Checks: []pr.Check{
				{ID: "github-actions:ci.yml:test", Provider: pr.CheckProviderGitHubActions, Name: "test", Status: "completed", Conclusion: "failure", Required: pr.RequiredUnknown},
				{ID: "github-actions:ci.yml:lint", Provider: pr.CheckProviderGitHubActions, Name: "lint", Status: "completed", Conclusion: "success", Required: pr.RequiredFalse},
			},
		}, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(filtered.FailedChecks) != 2 {
		t.Fatalf("failed-only failed summary = %#v", filtered.FailedChecks)
	}
	for _, entry := range filtered.PullRequests {
		for _, check := range entry.Checks {
			if !check.Failed() {
				t.Fatalf("failed-only retained non-failed check: %#v", check)
			}
		}
	}

	_, err = buildChecksReport(app, checksOptions{prNumber: 999}, func(string) (*pr.PullRequestChecks, error) {
		t.Fatal("fetcher should not be called for unmatched PR")
		return nil, nil
	})
	if err == nil || !strings.Contains(err.Error(), "no stack entry is associated with pull request #999") {
		t.Fatalf("unmatched PR error = %v", err)
	}

	_, err = buildChecksReport(app, checksOptions{}, func(string) (*pr.PullRequestChecks, error) {
		return nil, &pr.AuthError{Err: errors.New("authentication required")}
	})
	if err == nil || !pr.IsAuthError(err) {
		t.Fatalf("auth error = %v, want pr.AuthError", err)
	}
}

func TestBuildChecksReportEmptyStackDoesNotFetch(t *testing.T) {
	repoDir, base, _ := createCommentsTestRepo(t)
	chdirForTest(t, repoDir)
	app := commentsTestApp(repoDir, base, base)

	called := false
	report, err := buildChecksReport(app, checksOptions{}, func(string) (*pr.PullRequestChecks, error) {
		called = true
		return nil, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if called {
		t.Fatal("fetcher called for empty stack")
	}
	if len(report.Stack) != 0 || len(report.PullRequests) != 0 {
		t.Fatalf("empty stack report = %#v", report)
	}
}

func TestRootCleanCheckExemptsChecks(t *testing.T) {
	data, err := os.ReadFile("root.go")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), `cmd.Name() != "checks"`) {
		t.Fatal("root clean check does not exempt checks command")
	}
}
