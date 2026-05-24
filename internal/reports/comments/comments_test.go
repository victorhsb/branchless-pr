package comments

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/victorhsb/branchless-pr/internal/config"
	"github.com/victorhsb/branchless-pr/internal/pr"
)

func TestParseCommentKindsRejectsUnknown(t *testing.T) {
	_, err := parseCommentKinds("conversation,nope")
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), `unknown comments kind "nope"`) {
		t.Fatalf("error = %q", err.Error())
	}
}

func TestFilterCommentItemsUnresolvedAuthorAndKind(t *testing.T) {
	resolved := true
	unresolved := false
	items := []pr.CommentItem{
		{Kind: pr.CommentKindConversation, Author: "alice", Body: "general"},
		{Kind: pr.CommentKindReview, Author: "bob", State: "CHANGES_REQUESTED", Body: "needs work"},
		{Kind: pr.CommentKindReviewThread, Author: "carol", Resolved: &resolved, Body: "done"},
		{Kind: pr.CommentKindReviewThread, Author: "dana", Resolved: &unresolved, Body: "open", Replies: []pr.CommentItem{
			{Kind: pr.CommentKindReviewComment, Author: "erin", Body: "reply"},
		}},
	}
	kinds, err := parseCommentKinds("review,review_thread")
	if err != nil {
		t.Fatal(err)
	}
	got := filterCommentItems(items, Options{UnresolvedOnly: true}, kinds)
	if len(got) != 2 {
		t.Fatalf("filtered len = %d, want 2: %#v", len(got), got)
	}
	if got[0].Kind != pr.CommentKindReview || got[1].Kind != pr.CommentKindReviewThread {
		t.Fatalf("filtered order/kinds = %#v", got)
	}

	got = filterCommentItems(items, Options{Author: "erin"}, map[string]bool{pr.CommentKindReviewThread: true})
	if len(got) != 1 || len(got[0].Replies) != 1 || got[0].Replies[0].Author != "erin" {
		t.Fatalf("author-filtered thread = %#v", got)
	}
}

func TestResolveCommentIgnoredAuthors(t *testing.T) {
	emptyConfig, err := config.Load(filepath.Join(t.TempDir(), ".stack-pr.cfg"))
	if err != nil {
		t.Fatal(err)
	}
	if got := resolveCommentIgnoredAuthors(&AppContext{Config: emptyConfig}); len(got) != 0 {
		t.Fatalf("missing config ignored authors = %#v, want empty", got)
	}

	cfg := config.Defaults()
	cases := []struct {
		name string
		raw  string
		want []string
	}{
		{name: "empty", raw: "", want: nil},
		{name: "single", raw: "ci-bot", want: []string{"ci-bot"}},
		{name: "multi", raw: "ci-bot,release-bot", want: []string{"ci-bot", "release-bot"}},
		{name: "whitespace", raw: " ci-bot, , release-bot ", want: []string{"ci-bot", "release-bot"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cfg.Set("comments", "ignore_authors", tc.raw)
			got := resolveCommentIgnoredAuthors(&AppContext{Config: cfg})
			if strings.Join(got, ",") != strings.Join(tc.want, ",") {
				t.Fatalf("ignored authors = %#v, want %#v", got, tc.want)
			}
		})
	}
}

func TestFilterCommentItemsIgnoredAuthors(t *testing.T) {
	resolved := false
	items := []pr.CommentItem{
		{Kind: pr.CommentKindConversation, Author: "CI-Bot", Body: "status noise"},
		{Kind: pr.CommentKindReview, Author: "release-bot", Body: "release note"},
		{Kind: pr.CommentKindReviewComment, Author: "ci-bot", Body: "line noise"},
		{Kind: pr.CommentKindConversation, Author: "alice", Body: "human"},
		{Kind: pr.CommentKindReviewThread, Author: "ci-bot", Body: "thread noise", Resolved: &resolved, Replies: []pr.CommentItem{
			{Kind: pr.CommentKindReviewComment, Author: "ci-bot", Body: "bot reply"},
			{Kind: pr.CommentKindReviewComment, Author: "bob", Body: "human reply"},
		}},
	}
	kinds, err := parseCommentKinds("")
	if err != nil {
		t.Fatal(err)
	}

	got := filterCommentItems(items, Options{IgnoredAuthors: []string{"ci-bot", "release-bot"}}, kinds)
	if len(got) != 2 {
		t.Fatalf("filtered len = %d, want 2: %#v", len(got), got)
	}
	if got[0].Author != "alice" {
		t.Fatalf("first retained author = %q, want alice", got[0].Author)
	}
	thread := got[1]
	if thread.Kind != pr.CommentKindReviewThread {
		t.Fatalf("second retained kind = %q, want review_thread", thread.Kind)
	}
	if thread.Author != "bob" || thread.Body != "human reply" {
		t.Fatalf("thread metadata = author %q body %q, want bob/human reply", thread.Author, thread.Body)
	}
	if len(thread.Replies) != 1 || thread.Replies[0].Author != "bob" {
		t.Fatalf("thread replies = %#v, want only bob", thread.Replies)
	}
}

func TestFilterCommentItemsDropsAllIgnoredReviewThread(t *testing.T) {
	items := []pr.CommentItem{{
		Kind:   pr.CommentKindReviewThread,
		Author: "ci-bot",
		Body:   "thread noise",
		Replies: []pr.CommentItem{
			{Kind: pr.CommentKindReviewComment, Author: "ci-bot", Body: "bot reply"},
			{Kind: pr.CommentKindReviewComment, Author: "release-bot", Body: "release reply"},
		},
	}}
	kinds, err := parseCommentKinds("")
	if err != nil {
		t.Fatal(err)
	}

	got := filterCommentItems(items, Options{IgnoredAuthors: []string{"ci-bot", "release-bot"}}, kinds)
	if len(got) != 0 {
		t.Fatalf("filtered = %#v, want empty", got)
	}
}

func TestFilterCommentItemsAuthorDoesNotIncludeIgnoredAuthor(t *testing.T) {
	items := []pr.CommentItem{
		{Kind: pr.CommentKindConversation, Author: "ci-bot", Body: "bot"},
		{Kind: pr.CommentKindConversation, Author: "alice", Body: "human"},
	}
	kinds, err := parseCommentKinds("")
	if err != nil {
		t.Fatal(err)
	}

	got := filterCommentItems(items, Options{Author: "ci-bot", IgnoredAuthors: []string{"ci-bot"}}, kinds)
	if len(got) != 0 {
		t.Fatalf("filtered = %#v, want empty", got)
	}
}

func TestWriteCommentsReportJSONIsSingleObject(t *testing.T) {
	report := &commentsReport{
		SchemaVersion: "1",
		Command:       "stack-pr comments",
		Repository:    "/repo",
		Range:         commentsRange{Base: "main", Head: "HEAD", Remote: "origin", Target: "main"},
		Stack: []commentsStackEntry{{
			Index: 1, Commit: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", ShortSHA: "aaaaaaaaaaaa", Title: "First", Status: "fetched",
		}},
		PullRequests: []commentsPullRequestReport{{
			Index: 1, Commit: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", ShortSHA: "aaaaaaaaaaaa", Title: "First", Status: "fetched",
			Comments: []pr.CommentItem{{Kind: pr.CommentKindConversation, Author: "alice", Body: "hello"}},
		}},
	}
	var out bytes.Buffer
	if err := Write(&out, report, "json"); err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(out.Bytes(), []byte("\x1b")) {
		t.Fatalf("json contains ANSI escape: %q", out.String())
	}
	var payload struct {
		SchemaVersion string `json:"schema_version"`
		Command       string `json:"command"`
		PullRequests  []struct {
			Comments []struct {
				Kind string `json:"kind"`
			} `json:"comments"`
		} `json:"pull_requests"`
	}
	if err := json.Unmarshal(out.Bytes(), &payload); err != nil {
		t.Fatalf("invalid json: %v\n%s", err, out.String())
	}
	if payload.SchemaVersion != "1" || payload.Command != "stack-pr comments" {
		t.Fatalf("payload metadata = %#v", payload)
	}
	if payload.PullRequests[0].Comments[0].Kind != pr.CommentKindConversation {
		t.Fatalf("comment kind = %#v", payload.PullRequests[0].Comments[0])
	}
}

func TestWriteCommentsReportTextCoversEmptyMissingAndWarnings(t *testing.T) {
	report := &commentsReport{
		SchemaVersion: "1",
		Command:       "stack-pr comments",
		Range:         commentsRange{Base: "main", Head: "HEAD", Remote: "origin", Target: "main"},
		Stack: []commentsStackEntry{{
			Index: 1, Commit: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", ShortSHA: "aaaaaaaaaaaa", Title: "First", Status: "missing", Error: "missing PR metadata",
		}},
		PullRequests: []commentsPullRequestReport{{
			Index: 1, Commit: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", ShortSHA: "aaaaaaaaaaaa", Title: "First", Status: "missing", Error: "missing PR metadata",
		}},
	}
	var out bytes.Buffer
	if err := Write(&out, report, "text"); err != nil {
		t.Fatal(err)
	}
	text := out.String()
	for _, want := range []string{"# stack-pr comments", "No matching comments were found", "missing PR metadata", "No matching comments."} {
		if !strings.Contains(text, want) {
			t.Fatalf("text missing %q:\n%s", want, text)
		}
	}
}

func TestBuildCommentsReportHandlesMissingEmptyFailureAndAuth(t *testing.T) {
	repoDir, base, head := createCommentsTestRepo(t)
	chdirForTest(t, repoDir)
	app := commentsTestApp(repoDir, base, head)
	kinds, err := parseCommentKinds("")
	if err != nil {
		t.Fatal(err)
	}

	report, err := Build(app, Options{}, kinds, func(prRef string) (*pr.PullRequestComments, error) {
		if strings.Contains(prRef, "/pull/7") {
			return &pr.PullRequestComments{
				Number: 7,
				URL:    prRef,
				Items:  []pr.CommentItem{{Kind: pr.CommentKindConversation, Author: "alice", Body: "hello"}},
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
	if report.PullRequests[1].Status != "fetched" || len(report.PullRequests[1].Comments) != 1 {
		t.Fatalf("second entry = %#v", report.PullRequests[1])
	}
	if report.PullRequests[2].Status != "failed" {
		t.Fatalf("third status = %q, want failed", report.PullRequests[2].Status)
	}

	_, err = Build(app, Options{}, kinds, func(string) (*pr.PullRequestComments, error) {
		return nil, &pr.AuthError{Err: errors.New("authentication required")}
	})
	if err == nil || !pr.IsAuthError(err) {
		t.Fatalf("auth error = %v, want pr.AuthError", err)
	}
}

func TestBuildCommentsReportEmptyStackDoesNotFetch(t *testing.T) {
	repoDir, base, _ := createCommentsTestRepo(t)
	chdirForTest(t, repoDir)
	app := commentsTestApp(repoDir, base, base)
	kinds, err := parseCommentKinds("")
	if err != nil {
		t.Fatal(err)
	}

	called := false
	report, err := Build(app, Options{}, kinds, func(string) (*pr.PullRequestComments, error) {
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

func createCommentsTestRepo(t *testing.T) (repoDir, base, head string) {
	t.Helper()
	repoDir = t.TempDir()
	runGitCmd(t, repoDir, "init", "-b", "main")
	runGitCmd(t, repoDir, "config", "user.name", "Test User")
	runGitCmd(t, repoDir, "config", "user.email", "test@example.com")
	writeTestFile(t, repoDir, "file.txt", "base\n")
	runGitCmd(t, repoDir, "add", "file.txt")
	runGitCmd(t, repoDir, "commit", "-m", "base")
	base = gitOutput(t, repoDir, "rev-parse", "HEAD")

	writeTestFile(t, repoDir, "file.txt", "base\none\n")
	runGitCmd(t, repoDir, "add", "file.txt")
	runGitCmd(t, repoDir, "commit", "-m", "missing metadata")

	writeTestFile(t, repoDir, "file.txt", "base\none\ntwo\n")
	runGitCmd(t, repoDir, "add", "file.txt")
	runGitCmd(t, repoDir, "commit", "-m", "with pr", "-m", "stack-info: PR: https://github.com/acme/widgets/pull/7, branch: alice/stack/7")

	writeTestFile(t, repoDir, "file.txt", "base\none\ntwo\nthree\n")
	runGitCmd(t, repoDir, "add", "file.txt")
	runGitCmd(t, repoDir, "commit", "-m", "failed pr", "-m", "stack-info: PR: https://github.com/acme/widgets/pull/8, branch: alice/stack/8")
	head = gitOutput(t, repoDir, "rev-parse", "HEAD")

	bareRemote := filepath.Join(t.TempDir(), "remote.git")
	runGitCmd(t, "", "init", "--bare", bareRemote)
	runGitCmd(t, repoDir, "remote", "add", "origin", bareRemote)
	return repoDir, base, head
}

func commentsTestApp(repoDir, base, head string) *AppContext {
	return &AppContext{
		Args: CommonArgs{
			Base:               base,
			Head:               head,
			Remote:             "origin",
			Target:             "main",
			BranchNameTemplate: "$USERNAME/stack",
		},
		RepoRoot:   repoDir,
		Username:   "alice",
		OrigBranch: "main",
	}
}

func chdirForTest(t *testing.T, dir string) {
	t.Helper()
	old, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := os.Chdir(old); err != nil {
			t.Fatalf("restore cwd: %v", err)
		}
	})
}

func runGitCmd(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	if dir != "" {
		cmd.Dir = dir
	}
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v failed: %v\n%s", args, err, out)
	}
}

func gitOutput(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("git %v failed: %v", args, err)
	}
	return strings.TrimSpace(string(out))
}

func writeTestFile(t *testing.T, dir, name, body string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}
