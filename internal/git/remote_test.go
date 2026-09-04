package git

import (
	"strings"
	"testing"

	"github.com/victorhsb/branchless-pr/internal/shell/shelltest"
)

func TestResolveRemoteRefs(t *testing.T) {
	run := shelltest.New(t, shelltest.Response{
		Match:  shelltest.Exact("git", "ls-remote", "--heads", "--", "origin", "refs/heads/foo", "refs/heads/bar"),
		Stdout: "abc123\trefs/heads/foo\n",
	})

	got, err := New("", run).ResolveRemoteRefs("origin", "foo", "bar")
	if err != nil {
		t.Fatal(err)
	}
	if got["foo"] != "abc123" {
		t.Errorf("foo = %q, want abc123", got["foo"])
	}
	if _, ok := got["bar"]; ok {
		t.Errorf("bar should be absent")
	}
}

func TestRemoteBranchesUsesValidatedSharedParser(t *testing.T) {
	run := shelltest.New(t, shelltest.Response{
		Match: shelltest.Exact("git", "ls-remote", "--heads", "--", "origin"),
		Stdout: strings.Join([]string{
			"def456\trefs/heads/z-last",
			"malformed",
			"abc123\trefs/heads/a-first",
			"fff999\trefs/tags/not-a-branch",
		}, "\n"),
	})

	got, err := New("", run).RemoteBranches("origin")
	if err != nil {
		t.Fatal(err)
	}
	if want := []string{"a-first", "z-last"}; strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("RemoteBranches = %v, want %v", got, want)
	}
}

func TestForcePushWithLeaseUsesAtomicExplicitExpectations(t *testing.T) {
	run := shelltest.New(t, shelltest.Response{
		Match: shelltest.Exact(
			"git",
			"push",
			"--atomic",
			"--force-with-lease=refs/heads/foo:abc123",
			"--force-with-lease=refs/heads/bar:",
			"--",
			"origin",
			"foo:refs/heads/foo",
			"bar:refs/heads/bar",
		),
	})

	err := New("", run).ForcePushWithLease("origin", map[string]string{
		"foo": "abc123",
		"bar": "",
	}, "foo", "bar")
	if err != nil {
		t.Fatal(err)
	}
}

func TestParseRepoSlug(t *testing.T) {
	cases := []struct {
		name      string
		url       string
		wantOwner string
		wantRepo  string
		wantErr   bool
	}{
		{name: "https with .git", url: "https://github.com/acme/widget.git", wantOwner: "acme", wantRepo: "widget"},
		{name: "https without .git", url: "https://github.com/acme/widget", wantOwner: "acme", wantRepo: "widget"},
		{name: "https trailing slash", url: "https://github.com/acme/widget/", wantOwner: "acme", wantRepo: "widget"},
		{name: "ssh with .git", url: "git@github.com:acme/widget.git", wantOwner: "acme", wantRepo: "widget"},
		{name: "ssh without .git", url: "git@github.com:acme/widget", wantOwner: "acme", wantRepo: "widget"},
		{name: "ssh:// scheme", url: "ssh://git@github.com/acme/widget.git", wantOwner: "acme", wantRepo: "widget"},
		{name: "whitespace tolerated", url: "  https://github.com/acme/widget.git\n", wantOwner: "acme", wantRepo: "widget"},
		{name: "empty", url: "", wantErr: true},
		{name: "unsupported scheme", url: "file:///tmp/repo", wantErr: true},
		{name: "missing repo", url: "https://github.com/acme", wantErr: true},
		{name: "missing owner", url: "https://github.com//repo.git", wantErr: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			owner, repo, err := parseRepoSlug(tc.url)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("parseRepoSlug(%q) = (%q, %q, nil); want error", tc.url, owner, repo)
				}
				return
			}
			if err != nil {
				t.Fatalf("parseRepoSlug(%q) returned error: %v", tc.url, err)
			}
			if owner != tc.wantOwner || repo != tc.wantRepo {
				t.Fatalf("parseRepoSlug(%q) = (%q, %q); want (%q, %q)", tc.url, owner, repo, tc.wantOwner, tc.wantRepo)
			}
		})
	}
}
