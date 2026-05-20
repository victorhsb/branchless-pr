package pr

import (
	"strings"
	"testing"
)

func TestParsePRCommentSummaryNormalizesCommentsReviewsAndReviewComments(t *testing.T) {
	raw := []byte(`{
	  "number": 7,
	  "url": "https://github.com/acme/widgets/pull/7",
	  "comments": [
	    {
	      "id": "IC_1",
	      "author": {"login": "alice"},
	      "body": "Looks good overall",
	      "url": "https://github.com/acme/widgets/pull/7#issuecomment-1",
	      "createdAt": "2026-05-20T10:00:00Z",
	      "updatedAt": "2026-05-20T10:01:00Z"
	    }
	  ],
	  "reviews": [
	    {
	      "id": "PRR_1",
	      "author": {"login": "bob"},
	      "body": "Please fix this",
	      "url": "https://github.com/acme/widgets/pull/7#pullrequestreview-1",
	      "state": "CHANGES_REQUESTED",
	      "submittedAt": "2026-05-20T11:00:00Z",
	      "comments": [
	        {
	          "id": "PRRC_1",
	          "author": {"login": "bob"},
	          "body": "nit",
	          "url": "https://github.com/acme/widgets/pull/7#discussion_r1",
	          "createdAt": "2026-05-20T11:01:00Z",
	          "updatedAt": "2026-05-20T11:02:00Z",
	          "path": "main.go",
	          "line": 12
	        }
	      ]
	    }
	  ]
	}`)

	got, err := ParsePRCommentSummary(raw)
	if err != nil {
		t.Fatal(err)
	}
	if got.Number != 7 || got.URL != "https://github.com/acme/widgets/pull/7" {
		t.Fatalf("metadata = %#v", got)
	}
	if len(got.Items) != 3 {
		t.Fatalf("items = %d, want 3", len(got.Items))
	}
	if got.Items[0].Kind != CommentKindConversation || got.Items[0].Author != "alice" {
		t.Fatalf("conversation item = %#v", got.Items[0])
	}
	if got.Items[1].Kind != CommentKindReview || got.Items[1].State != "CHANGES_REQUESTED" {
		t.Fatalf("review item = %#v", got.Items[1])
	}
	if got.Items[2].Kind != CommentKindReviewComment || got.Items[2].Location == nil || got.Items[2].Location.Path != "main.go" {
		t.Fatalf("review comment item = %#v", got.Items[2])
	}
}

func TestParseReviewThreadsNormalizesResolutionAndReplies(t *testing.T) {
	raw := []byte(`{
	  "data": {
	    "repository": {
	      "pullRequest": {
	        "reviewThreads": {
	          "nodes": [
	            {
	              "id": "PRRT_1",
	              "isResolved": false,
	              "isOutdated": true,
	              "path": "main.go",
	              "line": 44,
	              "originalLine": 40,
	              "startLine": 41,
	              "originalStartLine": 39,
	              "diffSide": "RIGHT",
	              "startDiffSide": "RIGHT",
	              "comments": {
	                "nodes": [
	                  {
	                    "id": "PRRC_1",
	                    "author": {"login": "carol"},
	                    "body": "Can this be simpler?",
	                    "url": "https://github.com/acme/widgets/pull/7#discussion_r2",
	                    "createdAt": "2026-05-20T12:00:00Z",
	                    "updatedAt": "2026-05-20T12:01:00Z",
	                    "path": "main.go",
	                    "line": 44
	                  }
	                ]
	              }
	            }
	          ]
	        }
	      }
	    }
	  }
	}`)

	got, err := ParseReviewThreads(7, raw)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 {
		t.Fatalf("threads = %d, want 1", len(got))
	}
	thread := got[0]
	if thread.Kind != CommentKindReviewThread || thread.Resolved == nil || *thread.Resolved {
		t.Fatalf("thread resolution = %#v", thread)
	}
	if thread.Author != "carol" || !strings.Contains(thread.Body, "simpler") {
		t.Fatalf("thread summary fields = %#v", thread)
	}
	if thread.Location == nil || thread.Location.Path != "main.go" || !thread.Location.Outdated {
		t.Fatalf("thread location = %#v", thread.Location)
	}
	if len(thread.Replies) != 1 || thread.Replies[0].Kind != CommentKindReviewComment {
		t.Fatalf("thread replies = %#v", thread.Replies)
	}
}
