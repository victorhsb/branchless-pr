package cli

import (
	"errors"
	"strings"
	"testing"

	"github.com/victorhsb/branchless-pr/internal/config"
	"github.com/victorhsb/branchless-pr/internal/git"
	"github.com/victorhsb/branchless-pr/internal/shell/shelltest"
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
