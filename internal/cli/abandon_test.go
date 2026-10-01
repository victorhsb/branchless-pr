package cli

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/victorhsb/branchless-pr/internal/config"
	"github.com/victorhsb/branchless-pr/internal/git"
	"github.com/victorhsb/branchless-pr/internal/pr"
	"github.com/victorhsb/branchless-pr/internal/shell"
	"github.com/victorhsb/branchless-pr/internal/shell/shelltest"
	"github.com/victorhsb/branchless-pr/internal/stack"
)

func TestAbandonChecksOutGeneratedBranchFromCommit(t *testing.T) {
	const sha = "1111111111111111111111111111111111111111"
	const head = "alice/stack/1"
	raw := strings.Join([]string{
		sha,
		"tree 2222222222222222222222222222222222222222",
		"author Test User <test@example.com> 1 +0000",
		"committer Test User <test@example.com> 1 +0000",
		"",
		"    test commit",
		"",
		"    stack-info: PR: https://github.com/acme/widget/pull/1, branch: " + head,
	}, "\n")
	run := shelltest.New(t,
		shelltest.Response{
			Match:  shelltest.Exact("git", "rev-list", "--header", "^main", "HEAD"),
			Stdout: raw,
		},
		shelltest.Response{Match: shelltest.Exact("git", "fetch", "--prune", "--", "origin")},
		shelltest.Response{Match: shelltest.Exact("git", "ls-remote", "--heads", "--", "origin")},
		shelltest.Response{
			Match: shelltest.Exact("git", "checkout", sha, "-B", head),
			Err:   errors.New("stop after checkout"),
		},
	)
	cfg := config.Defaults()
	cfg.Set("github", "native_stacks", "off")
	err := abandonImpl(&AppContext{
		Config:     cfg,
		Git:        git.New("", run),
		Username:   "alice",
		OrigBranch: "feature",
		Args: CommonArgs{
			Base:               "main",
			Head:               "HEAD",
			Remote:             "origin",
			Target:             "main",
			BranchNameTemplate: "$USERNAME/stack",
		},
	})
	if err == nil || !strings.Contains(err.Error(), "stop after checkout") {
		t.Fatalf("error = %v, want checkout sentinel", err)
	}
}

func TestAbandonStopsBeforeBranchDeletionOnFailure(t *testing.T) {
	const sha = "1111111111111111111111111111111111111111"
	const head = "alice/stack/1"
	raw := strings.Join([]string{sha, "tree 2222222222222222222222222222222222222222", "author Test User <test@example.com> 1 +0000", "committer Test User <test@example.com> 1 +0000", "", "    change", "", "    user body", "", "    stack-info: PR: https://github.com/acme/widget/pull/1, branch: " + head}, "\n")
	steps := []shelltest.Response{
		{Name: "discover", Match: shelltest.Exact("git", "rev-list", "--header", "^main", "HEAD"), Stdout: raw},
		{Name: "fetch", Match: shelltest.Exact("git", "fetch", "--prune", "--", "origin")},
		{Name: "remote branches", Match: shelltest.Exact("git", "ls-remote", "--heads", "--", "origin")},
		{Name: "materialize branch", Match: shelltest.Exact("git", "checkout", sha, "-B", head)},
		{Name: "checkout generated branch", Match: shelltest.Exact("git", "checkout", head)},
		{Name: "strip metadata", Match: func(args []string, opts shell.RunOpts) bool {
			return shelltest.Exact("git", "commit", "--amend", "-F", "-")(args, opts) && strings.Contains(string(opts.Stdin), "user body") && !strings.Contains(string(opts.Stdin), "stack-info:")
		}},
		{Name: "resolve new tip", Match: shelltest.Exact("git", "rev-parse", "--verify", head), Stdout: sha},
		{Name: "rebase original", Match: shelltest.Exact("git", "rebase", "--committer-date-is-author-date", sha, "feature")},
	}
	for stop := range steps {
		t.Run(steps[stop].Name, func(t *testing.T) {
			sentinel := errors.New("abandon denied")
			responses := append([]shelltest.Response(nil), steps[:stop+1]...)
			responses[stop].Err = sentinel
			run := shelltest.New(t, responses...)
			cfg := config.Defaults()
			cfg.Set("github", "native_stacks", "off")
			err := abandonImpl(&AppContext{Config: cfg, Git: git.New("", run), Username: "alice", OrigBranch: "feature", Args: CommonArgs{Base: "main", Head: "HEAD", Target: "main", Remote: "origin", BranchNameTemplate: "$USERNAME/stack/$ID"}})
			if !errors.Is(err, sentinel) {
				t.Fatalf("error=%v", err)
			}
		})
	}
}

func TestAbandonNativePreflightBlocksMutations(t *testing.T) {
	var commits []string
	for _, n := range []int{2, 1} {
		sha := strings.Repeat(fmt.Sprint(n), 40)
		commits = append(commits, strings.Join([]string{sha, "tree " + strings.Repeat("3", 40), "author Test User <test@example.com> 1 +0000", "committer Test User <test@example.com> 1 +0000", "", fmt.Sprintf("    change %d", n), "", fmt.Sprintf("    stack-info: PR: https://github.com/acme/widget/pull/%d, branch: alice/stack/%d", n, n)}, "\n"))
	}
	// A membership read failure must stop before fetch, metadata rewriting,
	// merging, force-pushing, or branch deletion.
	responses := []shelltest.Response{}
	responses = append(responses, shelltest.Response{Match: shelltest.Exact("git", "rev-list", "--header", "^main", "HEAD"), Stdout: strings.Join(commits, "\x00")})
	responses = append(responses,
		shelltest.Response{Match: shelltest.Exact("git", "remote", "get-url", "--", "origin"), Stdout: "https://github.com/acme/widget.git"},
		shelltest.Response{Match: shelltest.Exact("gh", "api", "--include", "--method", "GET", "repos/acme/widget"), Stdout: "HTTP/2.0 403 Forbidden\n\n{\"message\":\"access denied\"}", ExitCode: 1},
	)
	run := shelltest.New(t, responses...)
	cfg := config.Defaults()
	cfg.Set("github", "native_stacks", "required")
	app := &AppContext{Config: cfg, Git: git.New("", run), PR: pr.NewClient(run), OrigBranch: "feature", Args: CommonArgs{Base: "main", Head: "HEAD", Target: "main", Remote: "origin", BranchNameTemplate: "$USERNAME/stack/$ID"}}
	err := abandonImpl(app)
	if err == nil || !strings.Contains(err.Error(), "access denied") {
		t.Fatalf("error=%v", err)
	}
}

func TestNativeAbandonAllowsAlreadyUnstackedPRs(t *testing.T) {
	for _, mode := range []string{"auto", "required"} {
		t.Run(mode, func(t *testing.T) {
			responses := []shelltest.Response{
				{Match: shelltest.Exact("git", "remote", "get-url", "--", "origin"), Stdout: "https://github.com/acme/widget.git"},
				{Match: shelltest.Exact("gh", "api", "--include", "--method", "GET", "repos/acme/widget"), Stdout: "HTTP/2.0 200 OK\n\n{}"},
				{Match: shelltest.Exact("gh", "api", "--include", "--method", "GET", "repos/acme/widget/stacks?per_page=1"), Stdout: "HTTP/2.0 200 OK\n\n[]"},
			}
			var st stack.Stack
			for _, n := range []int{1, 2} {
				responses = append(responses, shelltest.Response{
					Match:  shelltest.Exact("gh", "api", "--include", "--method", "GET", fmt.Sprintf("repos/acme/widget/pulls/%d", n)),
					Stdout: fmt.Sprintf(`HTTP/2.0 200 OK`+"\n\n"+`{"number":%d,"state":"open","draft":false,"merged_at":null,"head":{"ref":"branch-%d","sha":"head-sha"},"base":{"ref":"main","sha":"base-sha"},"stack":null}`, n, n),
				})
				st = append(st, entryForLandTest(fmt.Sprintf("alice/stack/%d", n), fmt.Sprintf("https://github.com/acme/widget/pull/%d", n)))
			}
			run := shelltest.New(t, responses...)
			cfg := config.Defaults()
			cfg.Set("github", "native_stacks", mode)
			app := &AppContext{Config: cfg, Git: git.New("", run), PR: pr.NewClient(run), Args: CommonArgs{Remote: "origin"}}
			if err := nativeAbandonPreflight(app, st); err != nil {
				t.Fatalf("already-unstacked PRs should permit cleanup: %v", err)
			}
		})
	}
}
