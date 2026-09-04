package invocation

import (
	"bytes"
	"path/filepath"
	"testing"

	"github.com/victorhsb/branchless-pr/internal/shell/shelltest"
)

func TestBootstrapUsesInjectedRunnerForDiscoveryAndContext(t *testing.T) {
	t.Setenv("STACKPR_CONFIG", filepath.Join(t.TempDir(), "missing.cfg"))
	run := shelltest.New(t,
		shelltest.Response{Match: shelltest.Exact("gh")},
		shelltest.Response{
			Match:  shelltest.Exact("git", "rev-parse", "--show-toplevel"),
			Stdout: "/repo\n",
		},
		shelltest.Response{
			Match:  shelltest.Exact("git", "rev-parse", "--abbrev-ref", "HEAD"),
			Stdout: "feature\n",
		},
		shelltest.Response{
			Match:    shelltest.Exact("git", "branchless", "query", "-r", "stack()"),
			ExitCode: 1,
		},
		shelltest.Response{
			Match:  shelltest.Exact("gh", "api", "graphql", "-f", "query=query{viewer{login}}"),
			Stdout: `{"data":{"viewer":{"login":"octocat"}}}`,
		},
	)

	bootstrap, err := NewBootstrap(run, false)
	if err != nil {
		t.Fatalf("NewBootstrap() error = %v", err)
	}
	app, err := bootstrap.Start(BootstrapOptions{
		Args: CommonArgs{
			Head:   "HEAD",
			Remote: "origin",
			Target: "main",
		},
		Policy: CommandPolicy{ConfigOnly: true, AllowsDirty: true},
		Stderr: &bytes.Buffer{},
	})
	if err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	if app.RepoRoot != "/repo" || app.Git.Dir != "/repo" {
		t.Fatalf("repository context = (%q, %q), want /repo", app.RepoRoot, app.Git.Dir)
	}
	if app.OrigBranch != "feature" || app.Username != "octocat" {
		t.Fatalf("identity = (%q, %q), want (feature, octocat)", app.OrigBranch, app.Username)
	}
	for i, call := range run.Calls() {
		if i >= 2 && call.Args[0] == "git" && call.Opts.Dir != "/repo" {
			t.Fatalf("call %d Dir = %q, want /repo", i, call.Opts.Dir)
		}
	}
}

func TestNewBootstrapDiscoversRepoOnceForDefaultConfigPath(t *testing.T) {
	t.Setenv("STACKPR_CONFIG", "")
	run := shelltest.New(t, shelltest.Response{
		Match:  shelltest.Exact("git", "rev-parse", "--show-toplevel"),
		Stdout: t.TempDir() + "\n",
	})

	bootstrap, err := NewBootstrap(run, false)
	if err != nil {
		t.Fatalf("NewBootstrap() error = %v", err)
	}
	if bootstrap.RepoRoot() == "" {
		t.Fatal("RepoRoot() is empty")
	}
}

func TestAgentBootstrapDoesNotInvokeRunner(t *testing.T) {
	run := shelltest.New(t)
	bootstrap, err := NewBootstrap(run, true)
	if err != nil {
		t.Fatalf("NewBootstrap() error = %v", err)
	}
	app, err := bootstrap.Start(BootstrapOptions{Policy: CommandPolicy{AgentOnly: true}})
	if err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	if app.Config == nil || app.Git == nil {
		t.Fatalf("agent context = %+v, want config and Git dependencies", app)
	}
}
