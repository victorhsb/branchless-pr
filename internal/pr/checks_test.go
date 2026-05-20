package pr

import "testing"

func TestParsePRChecksNormalizesCheckRunsStatusesAndComments(t *testing.T) {
	raw := []byte(`{
	  "number": 7,
	  "url": "https://github.com/acme/widgets/pull/7",
	  "headRefName": "alice/stack/7",
	  "baseRefName": "main",
	  "headRefOid": "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
	  "statusCheckRollup": [
	    {
	      "__typename": "CheckRun",
	      "databaseId": 101,
	      "name": "test",
	      "workflowName": "ci.yml",
	      "status": "COMPLETED",
	      "conclusion": "FAILURE",
	      "detailsUrl": "https://github.com/acme/widgets/actions/runs/1/job/101",
	      "startedAt": "2026-05-20T10:00:00Z",
	      "completedAt": "2026-05-20T10:05:00Z"
	    },
	    {
	      "__typename": "StatusContext",
	      "context": "codecov/project",
	      "state": "SUCCESS",
	      "targetUrl": "https://codecov.example/build/1",
	      "required": false
	    },
	    {
	      "__typename": "CheckRun",
	      "databaseId": 102,
	      "name": "lint",
	      "workflowName": "ci.yml",
	      "status": "IN_PROGRESS",
	      "detailsUrl": "https://github.com/acme/widgets/actions/runs/1/job/102",
	      "required": true
	    }
	  ],
	  "comments": [
	    {
	      "id": "IC_1",
	      "author": {"login": "alice"},
	      "body": "Please check the failing test",
	      "url": "https://github.com/acme/widgets/pull/7#issuecomment-1"
	    }
	  ],
	  "reviews": [
	    {
	      "id": "PRR_1",
	      "author": {"login": "bob"},
	      "body": "Changes requested",
	      "url": "https://github.com/acme/widgets/pull/7#pullrequestreview-1",
	      "state": "CHANGES_REQUESTED",
	      "comments": [
	        {
	          "id": "PRRC_1",
	          "author": {"login": "bob"},
	          "body": "fix this",
	          "url": "https://github.com/acme/widgets/pull/7#discussion_r1"
	        }
	      ]
	    }
	  ]
	}`)

	got, err := ParsePRChecks(raw)
	if err != nil {
		t.Fatal(err)
	}
	if got.Number != 7 || got.HeadRefName != "alice/stack/7" || got.HeadSHA == "" {
		t.Fatalf("metadata = %#v", got)
	}
	if len(got.Checks) != 3 {
		t.Fatalf("checks = %d, want 3: %#v", len(got.Checks), got.Checks)
	}
	byID := map[string]Check{}
	for _, check := range got.Checks {
		byID[check.ID] = check
	}
	test := byID["github-actions:ci.yml:test"]
	if test.Provider != CheckProviderGitHubActions || test.ProviderID != "101" || test.Conclusion != "failure" || !test.Failed() {
		t.Fatalf("test check = %#v", test)
	}
	lint := byID["github-actions:ci.yml:lint"]
	if lint.Required != RequiredTrue || lint.Status != "in_progress" {
		t.Fatalf("lint check = %#v", lint)
	}
	codecov := byID["github-status:codecov/project"]
	if codecov.Provider != CheckProviderGitHubStatus || codecov.Required != RequiredFalse || codecov.Status != "completed" || codecov.Conclusion != "success" {
		t.Fatalf("status context = %#v", codecov)
	}
	if got.CommentSummary.ConversationCount != 1 || got.CommentSummary.ReviewCount != 1 || got.CommentSummary.ReviewCommentCount != 1 || got.CommentSummary.RequestedChanges != 1 {
		t.Fatalf("comment summary = %#v", got.CommentSummary)
	}
	if len(got.CommentSummary.Snippets) == 0 || got.CommentSummary.InspectCommand != "stack-pr comments" {
		t.Fatalf("comment snippets/hint = %#v", got.CommentSummary)
	}
}

func TestSemanticCheckIDIsDeterministic(t *testing.T) {
	got := SemanticCheckID(CheckProviderGitHubActions, "CI Workflow", "", "Go Test / linux")
	want := "github-actions:ci-workflow:go-test-/-linux"
	if got != want {
		t.Fatalf("SemanticCheckID = %q, want %q", got, want)
	}
	if got2 := SemanticCheckID(CheckProviderGitHubActions, "CI Workflow", "", "Go Test / linux"); got2 != got {
		t.Fatalf("SemanticCheckID not deterministic: %q vs %q", got, got2)
	}
}
