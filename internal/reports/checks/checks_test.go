package checks

import (
	"bytes"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/victorhsb/branchless-pr/internal/pr"
)

func TestRunChecksRejectsUnknownFormat(t *testing.T) {
	err := RunWithFetcher(commentsTestApp(t.TempDir(), "base", "head"), Options{Format: "xml"}, &bytes.Buffer{}, nil)
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
	got := filterChecks(checks, Options{RequiredOnly: true})
	if len(got) != 2 || got[0].ID != "required-failed" || got[1].ID != "required-ok" {
		t.Fatalf("required-only = %#v", got)
	}
	got = filterChecks(checks, Options{FailedOnly: true})
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
	if err := Write(&out, report, "json", false); err != nil {
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
	if err := Write(&out, report, "text", false); err != nil {
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

	report, err := Build(app, Options{}, func(prRef string) (*pr.PullRequestChecks, error) {
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

	filtered, err := Build(app, Options{FailedOnly: true}, func(prRef string) (*pr.PullRequestChecks, error) {
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

	_, err = Build(app, Options{PRNumber: 999}, func(string) (*pr.PullRequestChecks, error) {
		t.Fatal("fetcher should not be called for unmatched PR")
		return nil, nil
	})
	if err == nil || !strings.Contains(err.Error(), "no stack entry is associated with pull request #999") {
		t.Fatalf("unmatched PR error = %v", err)
	}

	_, err = Build(app, Options{}, func(string) (*pr.PullRequestChecks, error) {
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
	report, err := Build(app, Options{}, func(string) (*pr.PullRequestChecks, error) {
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

// -- summary-first tests -------------------------------------------------------

func TestClassifyCheckBuckets(t *testing.T) {
	tests := []struct {
		name  string
		check pr.Check
		want  checkBucket
	}{
		{"failed", pr.Check{Conclusion: "failure"}, bucketFailing},
		{"error", pr.Check{Conclusion: "error"}, bucketFailing},
		{"success", pr.Check{Conclusion: "success"}, bucketPassing},
		{"skipped", pr.Check{Conclusion: "skipped"}, bucketSkipped},
		{"neutral", pr.Check{Conclusion: "neutral"}, bucketSkipped},
		{"cancelled", pr.Check{Conclusion: "cancelled"}, bucketSkipped},
		{"in_progress", pr.Check{Status: "in_progress"}, bucketInProgress},
		{"pending", pr.Check{Status: "pending"}, bucketPending},
		{"queued", pr.Check{Status: "queued"}, bucketPending},
		{"waiting", pr.Check{Status: "waiting"}, bucketPending},
		{"completed_success", pr.Check{Status: "completed", Conclusion: "success"}, bucketPassing},
		{"completed_unknown", pr.Check{Status: "completed", Conclusion: "cancelled"}, bucketSkipped},
		{"action_required", pr.Check{Conclusion: "action_required"}, bucketFailing},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := classifyCheck(tc.check)
			if got != tc.want {
				t.Fatalf("classifyCheck(%+v) = %q, want %q", tc.check, got, tc.want)
			}
		})
	}
}

func TestCollapseVisibleChecks(t *testing.T) {
	checks := []pr.Check{
		{ID: "a", Name: "Build", Status: "completed", Conclusion: "success"},
		{ID: "a", Name: "Build", Status: "in_progress"},
		{ID: "b", Name: "Test", Status: "completed", Conclusion: "failure"},
		{ID: "b", Name: "Test", Status: "completed", Conclusion: "skipped"},
		{ID: "c", Name: "Lint", Status: "pending"},
	}
	collapsed := collapseVisibleChecks(checks)
	if len(collapsed) != 3 {
		t.Fatalf("collapsed len = %d, want 3", len(collapsed))
	}
	// a: most actionable is in-progress
	if collapsed[0].Identity != "a" || collapsed[0].Bucket != bucketInProgress || collapsed[0].Count != 2 {
		t.Fatalf("a collapsed = %+v", collapsed[0])
	}
	// b: most actionable is failing
	if collapsed[1].Identity != "b" || collapsed[1].Bucket != bucketFailing || collapsed[1].Count != 2 {
		t.Fatalf("b collapsed = %+v", collapsed[1])
	}
	// c: pending
	if collapsed[2].Identity != "c" || collapsed[2].Bucket != bucketPending || collapsed[2].Count != 1 {
		t.Fatalf("c collapsed = %+v", collapsed[2])
	}
}

func TestSummarizePRChecks(t *testing.T) {
	checks := []pr.Check{
		{ID: "a", Name: "Build", Conclusion: "success"},
		{ID: "b", Name: "Test", Conclusion: "failure"},
		{ID: "c", Name: "Lint", Conclusion: "skipped"},
		{ID: "d", Name: "Deploy", Status: "in_progress"},
		{ID: "b", Name: "Test", Conclusion: "failure"}, // duplicate failing
	}
	s := summarizePRChecks(checks)
	if s.Total != 5 {
		t.Fatalf("total = %d, want 5", s.Total)
	}
	if s.Passing != 1 || s.Failing != 2 || s.Skipped != 1 || s.InProgress != 1 || s.Pending != 0 || s.Unknown != 0 {
		t.Fatalf("counts = %+v", s)
	}
	if len(s.FailedIDs) != 1 || s.FailedIDs[0] != "b" {
		t.Fatalf("failed IDs = %v, want [b]", s.FailedIDs)
	}
	if len(s.FailedNames) != 1 || s.FailedNames[0] != "Test" {
		t.Fatalf("failed names = %v, want [Test]", s.FailedNames)
	}
}

func TestWriteChecksTextSummaryFirst(t *testing.T) {
	report := &checksReport{
		SchemaVersion: "1",
		Command:       "stack-pr checks",
		Range:         checksRange{Base: "main", Head: "HEAD", Remote: "origin", Target: "main"},
		Stack: []checksStackEntry{
			{Index: 1, Commit: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", ShortSHA: "aaaaaaaaaaaa", Title: "First", Status: "fetched"},
		},
		PullRequests: []checksPullRequestReport{{
			Index: 1, Commit: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", ShortSHA: "aaaaaaaaaaaa", Title: "First", Status: "fetched", PRNumber: 7,
			Checks: []pr.Check{
				{ID: "github-actions:ci.yml:test", Provider: pr.CheckProviderGitHubActions, Name: "test", Status: "completed", Conclusion: "failure", Required: pr.RequiredUnknown, URL: "https://example.test/check"},
				{ID: "github-actions:ci.yml:lint", Provider: pr.CheckProviderGitHubActions, Name: "lint", Status: "completed", Conclusion: "success", Required: pr.RequiredFalse},
			},
		}},
		FailedChecks: []failedCheckSummary{
			{ID: "github-actions:ci.yml:test", PRNumber: 7, Commit: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", ShortSHA: "aaaaaaaaaaaa", Name: "test", Conclusion: "failure", URL: "https://example.test/check"},
		},
	}
	var out bytes.Buffer
	if err := Write(&out, report, "text", false); err != nil {
		t.Fatal(err)
	}
	text := out.String()

	// Stack coverage
	if !strings.Contains(text, "Stack:") {
		t.Fatalf("missing stack coverage:\n%s", text)
	}

	// Failed checks prominent
	if !strings.Contains(text, "## Failed checks") {
		t.Fatalf("missing failed checks section:\n%s", text)
	}

	// Per-PR roll-up
	if !strings.Contains(text, "Roll-up:") {
		t.Fatalf("missing roll-up:\n%s", text)
	}
	if !strings.Contains(text, "1 failing") || !strings.Contains(text, "1 passing") {
		t.Fatalf("roll-up counts wrong:\n%s", text)
	}
	if !strings.Contains(text, "Failed: test") {
		t.Fatalf("missing failed list:\n%s", text)
	}

	// Default collapsed checks shown
	if !strings.Contains(text, "- failing `github-actions:ci.yml:test` test") {
		t.Fatalf("missing collapsed check line:\n%s", text)
	}
	if !strings.Contains(text, "- passing `github-actions:ci.yml:lint` lint") {
		t.Fatalf("missing collapsed check line:\n%s", text)
	}

	// required: unknown omitted in default text
	if strings.Contains(text, "required: unknown") {
		t.Fatalf("default text should not contain 'required: unknown':\n%s", text)
	}
}

func TestWriteChecksTextVerbose(t *testing.T) {
	report := &checksReport{
		SchemaVersion: "1",
		Command:       "stack-pr checks",
		Range:         checksRange{Base: "main", Head: "HEAD", Remote: "origin", Target: "main"},
		Stack: []checksStackEntry{
			{Index: 1, Commit: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", ShortSHA: "aaaaaaaaaaaa", Title: "First", Status: "fetched"},
		},
		PullRequests: []checksPullRequestReport{{
			Index: 1, Commit: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", ShortSHA: "aaaaaaaaaaaa", Title: "First", Status: "fetched", PRNumber: 7,
			Checks: []pr.Check{
				{ID: "github-actions:ci.yml:test", Provider: pr.CheckProviderGitHubActions, Name: "test", Status: "completed", Conclusion: "failure", Required: pr.RequiredUnknown, URL: "https://example.test/check"},
				{ID: "github-actions:ci.yml:lint", Provider: pr.CheckProviderGitHubActions, Name: "lint", Status: "completed", Conclusion: "success", Required: pr.RequiredFalse},
			},
		}},
		FailedChecks: []failedCheckSummary{
			{ID: "github-actions:ci.yml:test", PRNumber: 7, Commit: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", ShortSHA: "aaaaaaaaaaaa", Name: "test", Conclusion: "failure", URL: "https://example.test/check"},
		},
	}
	var out bytes.Buffer
	if err := Write(&out, report, "text", true); err != nil {
		t.Fatal(err)
	}
	text := out.String()

	// Verbose still has summary
	if !strings.Contains(text, "Roll-up:") {
		t.Fatalf("verbose missing roll-up:\n%s", text)
	}

	// Verbose renders full per-check detail
	if !strings.Contains(text, "Checks:") {
		t.Fatalf("verbose missing Checks section:\n%s", text)
	}
	if !strings.Contains(text, "github-actions:ci.yml:test") {
		t.Fatalf("verbose missing check ID:\n%s", text)
	}
	if !strings.Contains(text, "github-actions:ci.yml:lint") {
		t.Fatalf("verbose missing check ID:\n%s", text)
	}

	// Verbose preserves required state, including unknown
	foundUnknown := false
	for _, line := range strings.Split(text, "\n") {
		if strings.Contains(line, "github-actions:ci.yml:test") && strings.Contains(line, "required: unknown") {
			foundUnknown = true
		}
		if strings.Contains(line, "github-actions:ci.yml:lint") && strings.Contains(line, "required: false") {
			// ok
		}
	}
	if !foundUnknown {
		t.Fatalf("verbose should preserve required: unknown for test:\n%s", text)
	}
}

func TestWriteChecksTextEmptyChecks(t *testing.T) {
	report := &checksReport{
		SchemaVersion: "1",
		Command:       "stack-pr checks",
		Range:         checksRange{Base: "main", Head: "HEAD", Remote: "origin", Target: "main"},
		Stack: []checksStackEntry{
			{Index: 1, Commit: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", ShortSHA: "aaaaaaaaaaaa", Title: "First", Status: "fetched"},
		},
		PullRequests: []checksPullRequestReport{{
			Index: 1, Commit: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", ShortSHA: "aaaaaaaaaaaa", Title: "First", Status: "fetched", PRNumber: 7,
		}},
	}
	var out bytes.Buffer
	if err := Write(&out, report, "text", false); err != nil {
		t.Fatal(err)
	}
	text := out.String()
	if !strings.Contains(text, "No checks were found") {
		t.Fatalf("missing no-checks message:\n%s", text)
	}
}

func TestJSONPreservesRequiredState(t *testing.T) {
	report := &checksReport{
		SchemaVersion: "1",
		Command:       "stack-pr checks",
		Repository:    "/repo",
		Range:         checksRange{Base: "main", Head: "HEAD", Remote: "origin", Target: "main"},
		Stack: []checksStackEntry{
			{Index: 1, Commit: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", ShortSHA: "aaaaaaaaaaaa", Title: "First", Status: "fetched"},
		},
		PullRequests: []checksPullRequestReport{{
			Index: 1, Commit: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", ShortSHA: "aaaaaaaaaaaa", Title: "First", Status: "fetched",
			Checks: []pr.Check{
				{ID: "a", Name: "A", Conclusion: "failure", Required: pr.RequiredUnknown},
				{ID: "b", Name: "B", Conclusion: "success", Required: pr.RequiredFalse},
			},
		}},
	}
	var out bytes.Buffer
	if err := Write(&out, report, "json", false); err != nil {
		t.Fatal(err)
	}
	var payload struct {
		PullRequests []struct {
			Checks []struct {
				Required string `json:"required"`
			} `json:"checks"`
		} `json:"pull_requests"`
	}
	if err := json.Unmarshal(out.Bytes(), &payload); err != nil {
		t.Fatalf("invalid json: %v\n%s", err, out.String())
	}
	got := payload.PullRequests[0].Checks
	if len(got) != 2 {
		t.Fatalf("checks len = %d, want 2", len(got))
	}
	if got[0].Required != "unknown" || got[1].Required != "false" {
		t.Fatalf("required = %v, want [unknown false]", []string{got[0].Required, got[1].Required})
	}
}
